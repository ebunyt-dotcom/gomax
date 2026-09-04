package auth

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/fingerprint"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/skip2/go-qrcode"
)

// CodeProvider retrieves SMS verification code.
type CodeProvider interface {
	GetCode(ctx context.Context) (string, error)
}

// PasswordProvider retrieves 2FA password.
type PasswordProvider interface {
	GetPassword(ctx context.Context) (string, error)
}

// PasswordProviderWithHint optionally allows 2FA password retrieval with server-provided hint.
type PasswordProviderWithHint interface {
	GetPasswordWithHint(ctx context.Context, hint string) (string, error)
}

// QrHandler handles presentation of QR code.
type QrHandler interface {
	HandleQr(ctx context.Context, qrURL string) error
}

// DeviceInfoProvider allows invokers to supply device ID and calls seed.
type DeviceInfoProvider interface {
	GetDeviceID() string
	GetCallsSeed() int64
}

func readLineWithContext(ctx context.Context, prompt string) (string, error) {
	fmt.Print(prompt)
	type result struct {
		text string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		ch <- result{text: text, err: err}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return "", res.err
		}
		return strings.TrimSpace(res.text), nil
	}
}

// ConsoleCodeProvider prompts user on console for SMS code.
type ConsoleCodeProvider struct{}

// GetCode reads an SMS code from the console.
func (p *ConsoleCodeProvider) GetCode(ctx context.Context) (string, error) {
	return readLineWithContext(ctx, "Enter SMS verification code: ")
}

// ConsolePasswordProvider prompts user on console for 2FA password.
type ConsolePasswordProvider struct{}

// GetPassword reads a 2FA password from the console.
func (p *ConsolePasswordProvider) GetPassword(ctx context.Context) (string, error) {
	return readLineWithContext(ctx, "Enter 2FA password: ")
}

// GetPasswordWithHint prompts for the 2FA password and displays the server hint.
func (p *ConsolePasswordProvider) GetPasswordWithHint(ctx context.Context, hint string) (string, error) {
	if hint != "" {
		return readLineWithContext(ctx, fmt.Sprintf("Enter 2FA password (hint: %s): ", hint))
	}
	return p.GetPassword(ctx)
}

// ConsoleQrHandler prints an ASCII QR code and its source URL to the console.
type ConsoleQrHandler struct{}

// HandleQr prints an ASCII QR code and its URL to the console.
func (h *ConsoleQrHandler) HandleQr(ctx context.Context, qrURL string) error {
	qr, err := qrcode.New(qrURL, qrcode.Medium)
	if err != nil {
		return fmt.Errorf("generate qr code: %w", err)
	}

	fmt.Println("\nОтсканируйте QR-код в приложении MAX:")
	fmt.Println(qr.ToSmallString(false))
	fmt.Printf("QR-ссылка: %s\n\n", qrURL)
	return nil
}

// AuthResult represents successful login result.
type AuthResult struct {
	Token      string
	UserID     int64
	IsRegister bool
}

// SmsAuthFlow handles phone and SMS based authentication.
type SmsAuthFlow struct {
	CodeProvider     CodeProvider
	PasswordProvider PasswordProvider
	DeviceID         string
	CallsSeed        int64
	Arch             string
	FpGen            *fingerprint.FingerprintGenerator
}

// NewSmsAuthFlow creates a new SMS authentication flow.
func NewSmsAuthFlow(codeProvider CodeProvider, pwdProvider PasswordProvider) *SmsAuthFlow {
	if codeProvider == nil {
		codeProvider = &ConsoleCodeProvider{}
	}
	if pwdProvider == nil {
		pwdProvider = &ConsolePasswordProvider{}
	}
	return &SmsAuthFlow{
		CodeProvider:     codeProvider,
		PasswordProvider: pwdProvider,
		Arch:             "arm64-v8a",
		FpGen:            fingerprint.NewFingerprintGenerator(fingerprint.DefaultFingerprint()),
	}
}

// Authenticate executes the SMS login handshake.
func (f *SmsAuthFlow) Authenticate(ctx context.Context, invoker api.Invoker, phone string) (*AuthResult, error) {
	if dip, ok := invoker.(DeviceInfoProvider); ok {
		if f.DeviceID == "" {
			f.DeviceID = dip.GetDeviceID()
		}
		if f.CallsSeed == 0 {
			f.CallsSeed = dip.GetCallsSeed()
		}
	}
	if f.Arch == "" {
		f.Arch = "arm64-v8a"
	}
	if f.FpGen == nil {
		f.FpGen = fingerprint.NewFingerprintGenerator(fingerprint.DefaultFingerprint())
	}

	var fpBytes []byte
	if f.FpGen != nil {
		var fpErr error
		fpBytes, fpErr = f.FpGen.GenerateFingerprint(f.DeviceID, f.CallsSeed, f.Arch)
		if fpErr != nil {
			return nil, fmt.Errorf("generate fingerprint failed: %w", fpErr)
		}
	}

	log.Printf("[gomax] Requesting SMS verification code for %s (START_AUTH, fingerprint=%d bytes)...", phone, len(fpBytes))
	reqPayload := map[string]interface{}{
		"phone": phone,
		"type":  "START_AUTH",
		"mode":  fpBytes,
	}
	reqRes, err := invoker.Invoke(ctx, protocol.OpAuthRequest, reqPayload)
	if err != nil {
		return nil, fmt.Errorf("request auth code failed: %w", err)
	}

	// Validate response does not contain an API validation error or rate limit
	if errVal, ok := reqRes["error"]; ok && errVal != nil {
		msg := reqRes["message"]
		if msg == nil {
			msg = reqRes["localizedMessage"]
		}
		return nil, fmt.Errorf("auth request rejected by server: %v (message: %v)", errVal, msg)
	}
	if errVal, ok := reqRes["err"]; ok && errVal != nil {
		return nil, fmt.Errorf("auth request error from server: %v", errVal)
	}

	// Save challenge token returned by OpAuthRequest
	challengeToken, _ := reqRes["token"].(string)
	if challengeToken == "" {
		if tid, ok := reqRes["trackId"].(string); ok && tid != "" {
			challengeToken = tid
		} else if tid, ok := reqRes["track_id"].(string); ok && tid != "" {
			challengeToken = tid
		}
	}
	if challengeToken == "" {
		return nil, fmt.Errorf("server did not return challenge token in auth request response: %v", reqRes)
	}
	// A token means that Max created an authentication challenge. It does not
	// prove that the SMS provider delivered a message, so keep the server's
	// delivery metadata visible without exposing the token itself.
	log.Printf("[gomax] SMS auth request accepted (codeLength=%v, requestsLeft=%v, requestMaxDuration=%v, altActionDuration=%v); challenge token acquired", reqRes["codeLength"], reqRes["requestCountLeft"], reqRes["requestMaxDuration"], reqRes["altActionDuration"])

	code, err := f.CodeProvider.GetCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("read sms code failed: %w", err)
	}

	log.Printf("[gomax] Verifying SMS code (CHECK_CODE)...")
	submitPayload := map[string]interface{}{
		"token":         challengeToken,
		"verifyCode":    code,
		"authTokenType": "CHECK_CODE",
	}
	res, err := invoker.Invoke(ctx, protocol.OpAuth, submitPayload)

	token, isRegister := extractToken(res)

	// Check for 2FA password challenge (either in response or via error)
	var passwordChallenge map[string]interface{}
	if res != nil {
		if pc, ok := res["passwordChallenge"].(map[string]interface{}); ok {
			passwordChallenge = pc
		} else if pc, ok := res["password_challenge"].(map[string]interface{}); ok {
			passwordChallenge = pc
		}
	}

	hasPasswordTrack := false
	var challengeTrackID string
	var challengeHint string
	if passwordChallenge != nil {
		if tid, ok := passwordChallenge["trackId"].(string); ok && tid != "" {
			challengeTrackID = tid
			hasPasswordTrack = true
		} else if tid, ok := passwordChallenge["track_id"].(string); ok && tid != "" {
			challengeTrackID = tid
			hasPasswordTrack = true
		}
		if h, ok := passwordChallenge["hint"].(string); ok {
			challengeHint = h
		}
	}

	is2FAError := err != nil && (strings.Contains(err.Error(), "PASSWORD") || strings.Contains(err.Error(), "2FA"))

	// Only invoke 2FA if login token wasn't already received and a challenge is present
	if token == "" && (hasPasswordTrack || is2FAError) {
		trackID := challengeTrackID
		if trackID == "" {
			trackID = challengeToken
		}

		if challengeHint != "" {
			log.Printf("[gomax] 2FA password challenge detected (hint: %s); requesting password...", challengeHint)
		} else {
			log.Printf("[gomax] 2FA password challenge detected; requesting password...")
		}

		var pwd string
		var pErr error
		if pwh, ok := f.PasswordProvider.(PasswordProviderWithHint); ok {
			pwd, pErr = pwh.GetPasswordWithHint(ctx, challengeHint)
		} else {
			pwd, pErr = f.PasswordProvider.GetPassword(ctx)
		}
		if pErr != nil {
			return nil, fmt.Errorf("get 2fa password failed: %w", pErr)
		}

		log.Printf("[gomax] Submitting 2FA password (AUTH_LOGIN_CHECK_PASSWORD)...")
		pwdPayload := map[string]interface{}{
			"trackId":  trackID,
			"password": pwd,
		}
		res, err = invoker.Invoke(ctx, protocol.OpAuthLoginCheckPassword, pwdPayload)
		if err != nil {
			// Fallback: try OpAuth with token, phone, and password
			log.Printf("[gomax] OpAuthLoginCheckPassword returned error (%v); trying fallback OpAuth...", err)
			fallbackPayload := map[string]interface{}{
				"token":    trackID,
				"phone":    phone,
				"password": pwd,
			}
			res, err = invoker.Invoke(ctx, protocol.OpAuth, fallbackPayload)
			if err != nil {
				return nil, fmt.Errorf("2fa auth failed: %w", err)
			}
		}

		// Check for 2FA validation error in response
		if errVal, ok := res["error"]; ok && errVal != nil {
			msg := res["message"]
			if msg == nil {
				msg = res["localizedMessage"]
			}
			return nil, fmt.Errorf("2fa password rejected by server: %v (message: %v)", errVal, msg)
		}

		token, isRegister = extractToken(res)
	} else if token == "" {
		// No 2FA and no token: check for error response
		if errVal, ok := res["error"]; ok && errVal != nil {
			msg := res["message"]
			if msg == nil {
				msg = res["localizedMessage"]
			}
			return nil, fmt.Errorf("verification code rejected by server: %v (message: %v)", errVal, msg)
		}
		if errVal, ok := res["err"]; ok && errVal != nil {
			return nil, fmt.Errorf("verification code error from server: %v", errVal)
		}
		if err != nil {
			return nil, fmt.Errorf("submit auth code failed: %w", err)
		}
	}

	if token == "" {
		return nil, fmt.Errorf("authentication completed without login or register token: %v", res)
	}

	if isRegister {
		log.Printf("[gomax] Received registration token for new account")
	} else {
		log.Printf("[gomax] Authentication successful, received session token")
	}

	var uid int64
	if u, ok := res["userId"].(int64); ok {
		uid = u
	} else if uF, ok := res["userId"].(float64); ok {
		uid = int64(uF)
	} else if uI, ok := res["userId"].(int); ok {
		uid = int64(uI)
	} else if uU, ok := res["userId"].(uint64); ok {
		uid = int64(uU)
	} else if u, ok := res["userToken"].(int64); ok {
		uid = u
	} else if uF, ok := res["userToken"].(float64); ok {
		uid = int64(uF)
	}

	return &AuthResult{Token: token, UserID: uid, IsRegister: isRegister}, nil
}

// extractToken retrieves the login or register token from server response attributes.
func extractToken(res map[string]interface{}) (string, bool) {
	if res == nil {
		return "", false
	}

	// 1. Check tokenAttrs / token_attrs (PyMax format: tokenAttrs.LOGIN.token or tokenAttrs.REGISTER.token)
	var attrs map[string]interface{}
	if a, ok := res["tokenAttrs"].(map[string]interface{}); ok {
		attrs = a
	} else if a, ok := res["token_attrs"].(map[string]interface{}); ok {
		attrs = a
	}

	if attrs != nil {
		for _, key := range []string{"LOGIN", "login", "loginToken", "login_token"} {
			if loginObj, ok := attrs[key].(map[string]interface{}); ok {
				for _, tKey := range []string{"token", "value", "t"} {
					if t, ok := loginObj[tKey].(string); ok && t != "" {
						return t, false
					}
				}
			} else if t, ok := attrs[key].(string); ok && t != "" {
				return t, false
			}
		}
		for _, key := range []string{"REGISTER", "register", "registerToken", "register_token"} {
			if regObj, ok := attrs[key].(map[string]interface{}); ok {
				for _, tKey := range []string{"token", "value", "t"} {
					if t, ok := regObj[tKey].(string); ok && t != "" {
						return t, true
					}
				}
			} else if t, ok := attrs[key].(string); ok && t != "" {
				return t, true
			}
		}
	}

	// 2. Direct top-level fields
	for _, key := range []string{"token", "loginToken", "login_token"} {
		if t, ok := res[key].(string); ok && t != "" {
			return t, false
		}
	}
	for _, key := range []string{"registerToken", "register_token"} {
		if t, ok := res[key].(string); ok && t != "" {
			return t, true
		}
	}

	return "", false
}

// QrAuthFlow handles Web QR code authentication.
type QrAuthFlow struct {
	QrHandler        QrHandler
	PasswordProvider PasswordProvider
}

// NewQrAuthFlow creates a new QR authentication flow.
func NewQrAuthFlow(qrHandler QrHandler, pwdProvider PasswordProvider) *QrAuthFlow {
	if qrHandler == nil {
		qrHandler = &ConsoleQrHandler{}
	}
	if pwdProvider == nil {
		pwdProvider = &ConsolePasswordProvider{}
	}
	return &QrAuthFlow{
		QrHandler:        qrHandler,
		PasswordProvider: pwdProvider,
	}
}

// Authenticate polls and completes QR authorization.
func (f *QrAuthFlow) Authenticate(ctx context.Context, invoker api.Invoker) (*AuthResult, error) {
	if f.QrHandler == nil {
		f.QrHandler = &ConsoleQrHandler{}
	}
	if f.PasswordProvider == nil {
		f.PasswordProvider = &ConsolePasswordProvider{}
	}
	// QR login is a three-step protocol: GET_QR, GET_QR_STATUS and
	// LOGIN_BY_QR. It is deliberately kept separate from mobile SMS auth;
	// the server treats AUTH_REQUEST as a phone-auth track only.
	res, err := invoker.Invoke(ctx, protocol.OpGetQr, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("request qr link failed: %w", err)
	}

	trackID, _ := res["trackId"].(string)
	if trackID == "" {
		trackID, _ = res["track_id"].(string)
	}
	qrURL, _ := res["qrLink"].(string)
	if qrURL == "" {
		qrURL, _ = res["qr_link"].(string)
	}
	if qrURL == "" {
		qrURL, _ = res["url"].(string)
	}
	if trackID == "" {
		return nil, fmt.Errorf("qr response did not contain track id: %v", res)
	}
	if qrURL != "" {
		if err := f.QrHandler.HandleQr(ctx, qrURL); err != nil {
			return nil, err
		}
	}

	interval := 2 * time.Second
	if ms, ok := authInt64(res["pollingInterval"]); ok && ms > 0 {
		interval = time.Duration(ms) * time.Millisecond
	}
	expiresAt := time.Now().Add(5 * time.Minute)
	if ms, ok := authInt64(res["expiresAt"]); ok && ms > 0 {
		expiresAt = time.UnixMilli(ms)
	}

	// Poll until the phone confirms the QR code or the track expires.
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(expiresAt) {
				return nil, fmt.Errorf("qr authentication expired")
			}
			statusRes, err := invoker.Invoke(ctx, protocol.OpGetQrStatus, map[string]interface{}{
				"trackId": trackID,
			})
			if err != nil {
				continue
			}
			approved := false
			if status, ok := statusRes["status"].(map[string]interface{}); ok {
				approved, _ = status["loginAvailable"].(bool)
				if !approved {
					approved, _ = status["login_available"].(bool)
				}
			}
			if !approved {
				approved, _ = statusRes["loginAvailable"].(bool)
			}
			if !approved {
				continue
			}

			confirmed, err := invoker.Invoke(ctx, protocol.OpLoginByQr, map[string]interface{}{
				"trackId": trackID,
			})
			if err != nil {
				return nil, fmt.Errorf("confirm qr login failed: %w", err)
			}
			token, _ := extractToken(confirmed)
			if token == "" {
				if challenge, ok := confirmed["passwordChallenge"].(map[string]interface{}); ok {
					passwordTrack, _ := challenge["trackId"].(string)
					hint, _ := challenge["hint"].(string)
					password, pErr := f.password(ctx, hint)
					if pErr != nil {
						return nil, pErr
					}
					confirmed, err = invoker.Invoke(ctx, protocol.OpAuthLoginCheckPassword, map[string]interface{}{
						"trackId": passwordTrack, "password": password,
					})
					if err != nil {
						return nil, fmt.Errorf("qr 2fa verification failed: %w", err)
					}
					token, _ = extractToken(confirmed)
				}
			}
			if token == "" {
				return nil, fmt.Errorf("qr login completed without token: %v", confirmed)
			}
			var uid int64
			if u, ok := authInt64(confirmed["userId"]); ok {
				uid = u
			}
			return &AuthResult{Token: token, UserID: uid}, nil
		}
	}
}

func (f *QrAuthFlow) password(ctx context.Context, hint string) (string, error) {
	if provider, ok := f.PasswordProvider.(PasswordProviderWithHint); ok {
		return provider.GetPasswordWithHint(ctx, hint)
	}
	return f.PasswordProvider.GetPassword(ctx)
}

func authInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float64:
		return int64(v), true
	case string:
		var parsed int64
		if _, err := fmt.Sscan(v, &parsed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}
