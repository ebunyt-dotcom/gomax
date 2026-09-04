package auth

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
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
	}
}

// Authenticate executes the SMS login handshake.
func (f *SmsAuthFlow) Authenticate(ctx context.Context, invoker api.Invoker, phone string) (*AuthResult, error) {
	// Request code
	reqPayload := map[string]interface{}{
		"phone": phone,
		"type":  "SMS",
	}
	_, err := invoker.Invoke(ctx, protocol.OpAuthRequest, reqPayload)
	if err != nil {
		return nil, fmt.Errorf("request auth code failed: %w", err)
	}

	code, err := f.CodeProvider.GetCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("read sms code failed: %w", err)
	}

	submitPayload := map[string]interface{}{
		"phone": phone,
		"code":  code,
	}
	res, err := invoker.Invoke(ctx, protocol.OpAuth, submitPayload)
	if err != nil {
		// Check for 2FA password requirement
		if strings.Contains(err.Error(), "PASSWORD") || strings.Contains(err.Error(), "2FA") {
			pwd, pErr := f.PasswordProvider.GetPassword(ctx)
			if pErr != nil {
				return nil, pErr
			}
			pwdPayload := map[string]interface{}{
				"phone":    phone,
				"password": pwd,
			}
			res, err = invoker.Invoke(ctx, protocol.OpAuth, pwdPayload)
			if err != nil {
				return nil, fmt.Errorf("2fa auth failed: %w", err)
			}
		} else {
			return nil, fmt.Errorf("submit auth code failed: %w", err)
		}
	}

	token, _ := res["token"].(string)
	var uid int64
	if u, ok := res["userId"].(int64); ok {
		uid = u
	} else if uF, ok := res["userId"].(float64); ok {
		uid = int64(uF)
	}

	return &AuthResult{Token: token, UserID: uid}, nil
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
