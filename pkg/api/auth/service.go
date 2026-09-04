// Package auth exposes the authenticated Max RPC methods used by PyMax.
package auth

import (
	"context"
	"fmt"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
)

// AuthService provides direct access to SMS, QR, password and registration
// operations. The high-level auth flows in pkg/auth use the same methods.
type AuthService struct{ invoker api.Invoker }

// NewAuthService creates an AuthService backed by an RPC invoker.
func NewAuthService(invoker api.Invoker) *AuthService { return &AuthService{invoker: invoker} }

// RequestCode starts SMS authentication. mode is the mobile fingerprint;
// pass nil for web-style or custom authentication requests.
func (s *AuthService) RequestCode(ctx context.Context, phone string, mode []byte) (map[string]interface{}, error) {
	payload := map[string]interface{}{"phone": phone, "type": "START_AUTH", "mode": mode}
	res, err := s.invoker.Invoke(ctx, protocol.OpAuthRequest, payload)
	if err != nil {
		return nil, fmt.Errorf("request auth code failed: %w", err)
	}
	return res, nil
}

// SendCode verifies an SMS code and returns token attributes or a 2FA challenge.
func (s *AuthService) SendCode(ctx context.Context, token, code string) (map[string]interface{}, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpAuth, map[string]interface{}{
		"token": token, "verifyCode": code, "authTokenType": "CHECK_CODE",
	})
	if err != nil {
		return nil, fmt.Errorf("verify auth code failed: %w", err)
	}
	return res, nil
}

// CheckPassword completes a password challenge returned by SMS or QR login.
func (s *AuthService) CheckPassword(ctx context.Context, trackID, password string) (map[string]interface{}, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpAuthLoginCheckPassword, map[string]interface{}{
		"trackId": trackID, "password": password,
	})
	if err != nil {
		return nil, fmt.Errorf("check auth password failed: %w", err)
	}
	return res, nil
}

// RequestQR requests a fresh QR login challenge.
func (s *AuthService) RequestQR(ctx context.Context) (map[string]interface{}, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpGetQr, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("request qr failed: %w", err)
	}
	return res, nil
}

// CheckQR polls a QR login challenge by track ID.
func (s *AuthService) CheckQR(ctx context.Context, trackID string) (map[string]interface{}, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpGetQrStatus, map[string]interface{}{"trackId": trackID})
	if err != nil {
		return nil, fmt.Errorf("check qr failed: %w", err)
	}
	return res, nil
}

// ConfirmQR finalizes a scanned and approved QR challenge.
func (s *AuthService) ConfirmQR(ctx context.Context, trackID string) (map[string]interface{}, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpLoginByQr, map[string]interface{}{"trackId": trackID})
	if err != nil {
		return nil, fmt.Errorf("confirm qr failed: %w", err)
	}
	return res, nil
}

// ApproveQR approves a QR link from an already authenticated mobile client.
func (s *AuthService) ApproveQR(ctx context.Context, qrLink string) error {
	if _, err := s.invoker.Invoke(ctx, protocol.OpAuthQrApprove, map[string]interface{}{"qrLink": qrLink}); err != nil {
		return fmt.Errorf("approve qr failed: %w", err)
	}
	return nil
}

// CreateAuthTrack starts a multi-step 2FA setup/change operation.
func (s *AuthService) CreateAuthTrack(ctx context.Context) (string, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpAuthCreateTrack, map[string]interface{}{"type": 0})
	if err != nil {
		return "", fmt.Errorf("create auth track failed: %w", err)
	}
	if trackID, ok := res["trackId"].(string); ok && trackID != "" {
		return trackID, nil
	}
	return "", fmt.Errorf("create auth track response did not contain trackId")
}

// SetPassword validates and sets a password on an auth track.
func (s *AuthService) SetPassword(ctx context.Context, trackID, password string) error {
	if _, err := s.invoker.Invoke(ctx, protocol.OpAuthValidatePassword, map[string]interface{}{"trackId": trackID, "password": password}); err != nil {
		return fmt.Errorf("set auth password failed: %w", err)
	}
	return nil
}

// SetHint validates a password hint on an auth track.
func (s *AuthService) SetHint(ctx context.Context, trackID, hint string) error {
	if _, err := s.invoker.Invoke(ctx, protocol.OpAuthValidateHint, map[string]interface{}{"trackId": trackID, "hint": hint}); err != nil {
		return fmt.Errorf("set auth hint failed: %w", err)
	}
	return nil
}

// RequestEmailCode starts 2FA email verification.
func (s *AuthService) RequestEmailCode(ctx context.Context, trackID, email string) error {
	if _, err := s.invoker.Invoke(ctx, protocol.OpAuthVerifyEmail, map[string]interface{}{"trackId": trackID, "email": email}); err != nil {
		return fmt.Errorf("request auth email code failed: %w", err)
	}
	return nil
}

// VerifyEmailCode completes 2FA email verification.
func (s *AuthService) VerifyEmailCode(ctx context.Context, trackID, code string) error {
	if _, err := s.invoker.Invoke(ctx, protocol.OpAuthCheckEmail, map[string]interface{}{"trackId": trackID, "verifyCode": code}); err != nil {
		return fmt.Errorf("verify auth email code failed: %w", err)
	}
	return nil
}

// CommitTwoFactor commits an already prepared 2FA auth track.
func (s *AuthService) CommitTwoFactor(ctx context.Context, trackID, password, hint string, capabilities []string) error {
	payload := map[string]interface{}{
		"trackId": trackID, "password": password,
		"hint": hint, "expectedCapabilities": capabilities,
	}
	if _, err := s.invoker.Invoke(ctx, protocol.OpAuthSet2Fa, payload); err != nil {
		return fmt.Errorf("commit 2fa settings failed: %w", err)
	}
	return nil
}

// ConfirmRegistration turns a registration token into a login session.
func (s *AuthService) ConfirmRegistration(ctx context.Context, firstName, lastName, token string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"firstName": firstName, "lastName": lastName,
		"token": token, "tokenType": "REGISTER",
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpAuthConfirm, payload)
	if err != nil {
		return nil, fmt.Errorf("confirm registration failed: %w", err)
	}
	return res, nil
}
