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
)

// CodeProvider retrieves SMS verification code.
type CodeProvider interface {
	GetCode(ctx context.Context) (string, error)
}

// PasswordProvider retrieves 2FA password.
type PasswordProvider interface {
	GetPassword(ctx context.Context) (string, error)
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

// ConsoleCodeProvider prompts user on console for SMS code.
type ConsoleCodeProvider struct{}

func (p *ConsoleCodeProvider) GetCode(ctx context.Context) (string, error) {
	fmt.Print("Enter SMS verification code: ")
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// ConsolePasswordProvider prompts user on console for 2FA password.
type ConsolePasswordProvider struct{}

func (p *ConsolePasswordProvider) GetPassword(ctx context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// ConsoleQrHandler prints QR code URL to console.
type ConsoleQrHandler struct{}

func (h *ConsoleQrHandler) HandleQr(ctx context.Context, qrURL string) error {
	fmt.Printf("\nScan QR code to login:\n%s\n\n", qrURL)
	return nil
}

// AuthResult represents successful login result.
type AuthResult struct {
	Token  string
	UserID int64
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

	// Save challenge token returned by OpAuthRequest
	challengeToken, _ := reqRes["token"].(string)
	if challengeToken == "" {
		return nil, fmt.Errorf("server did not return challenge token in auth request response: %v", reqRes)
	}
	log.Printf("[gomax] SMS code requested successfully; challenge token acquired")

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

	// Check for 2FA password challenge (either in response or via error)
	var passwordChallenge map[string]interface{}
	if res != nil {
		if pc, ok := res["passwordChallenge"].(map[string]interface{}); ok {
			passwordChallenge = pc
		} else if pc, ok := res["password_challenge"].(map[string]interface{}); ok {
			passwordChallenge = pc
		}
	}

	is2FAError := err != nil && (strings.Contains(err.Error(), "PASSWORD") || strings.Contains(err.Error(), "2FA"))

	if passwordChallenge != nil || is2FAError {
		log.Printf("[gomax] 2FA password challenge detected; requesting password...")
		pwd, pErr := f.PasswordProvider.GetPassword(ctx)
		if pErr != nil {
			return nil, fmt.Errorf("get 2fa password failed: %w", pErr)
		}

		trackID := challengeToken
		if passwordChallenge != nil {
			if tid, ok := passwordChallenge["trackId"].(string); ok && tid != "" {
				trackID = tid
			} else if tid, ok := passwordChallenge["track_id"].(string); ok && tid != "" {
				trackID = tid
			}
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
	} else if err != nil {
		return nil, fmt.Errorf("submit auth code failed: %w", err)
	}

	token, isRegister := extractToken(res)
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
	} else if u, ok := res["userToken"].(int64); ok {
		uid = u
	} else if uF, ok := res["userToken"].(float64); ok {
		uid = int64(uF)
	}

	return &AuthResult{Token: token, UserID: uid}, nil
}

// extractToken retrieves the login or register token from server response attributes.
func extractToken(res map[string]interface{}) (string, bool) {
	if res == nil {
		return "", false
	}

	// 1. Check tokenAttrs (PyMax format: tokenAttrs.LOGIN.token or tokenAttrs.REGISTER.token)
	if attrs, ok := res["tokenAttrs"].(map[string]interface{}); ok {
		for _, key := range []string{"LOGIN", "login"} {
			if loginObj, ok := attrs[key].(map[string]interface{}); ok {
				if t, ok := loginObj["token"].(string); ok && t != "" {
					return t, false
				}
			}
		}
		for _, key := range []string{"REGISTER", "register", "registerToken", "register_token"} {
			if regObj, ok := attrs[key].(map[string]interface{}); ok {
				if t, ok := regObj["token"].(string); ok && t != "" {
					return t, true
				}
			}
		}
	}

	// 2. Direct top-level fields
	if t, ok := res["token"].(string); ok && t != "" {
		return t, false
	}
	if t, ok := res["loginToken"].(string); ok && t != "" {
		return t, false
	}
	if t, ok := res["registerToken"].(string); ok && t != "" {
		return t, true
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
	// Request QR link
	res, err := invoker.Invoke(ctx, protocol.OpAuthRequest, map[string]interface{}{
		"type": "QR",
	})
	if err != nil {
		return nil, fmt.Errorf("request qr link failed: %w", err)
	}

	qrURL, _ := res["url"].(string)
	if qrURL != "" {
		if err := f.QrHandler.HandleQr(ctx, qrURL); err != nil {
			return nil, err
		}
	}

	// Poll until approved or expired
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			statusRes, err := invoker.Invoke(ctx, protocol.OpAuth, map[string]interface{}{
				"type": "POLL_QR",
			})
			if err == nil {
				token, _ := statusRes["token"].(string)
				if token != "" {
					var uid int64
					if u, ok := statusRes["userId"].(int64); ok {
						uid = u
					}
					return &AuthResult{Token: token, UserID: uid}, nil
				}
			}
		}
	}
}
