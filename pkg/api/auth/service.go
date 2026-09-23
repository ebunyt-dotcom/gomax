// Package auth exposes the authenticated Max RPC methods used by PyMax.
package auth

import (
	"context"
	"fmt"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// AuthService provides direct access to SMS, QR, password and registration
// operations. The high-level auth flows in pkg/auth use the same methods.
type AuthService struct{ invoker api.Invoker }

// TwoFactorAction is the numeric capability identifier used by the Max wire
// protocol when committing a two-factor authentication track.
type TwoFactorAction int

const (
	TwoFactorSetPassword TwoFactorAction = iota
	TwoFactorUpdatePassword
	TwoFactorRestorePassword
	TwoFactorHint
	TwoFactorEmail
	TwoFactorRemove
)

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

func (s *AuthService) RequestCodeResult(ctx context.Context, phone string, mode []byte) (*types.StartAuthResponse, error) {
	res, err := s.RequestCode(ctx, phone, mode)
	if err != nil {
		return nil, err
	}
	result := types.ParseStartAuthResponse(res)
	return &result, nil
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

func (s *AuthService) SendCodeResult(ctx context.Context, token, code string) (*types.CheckCodeResponse, error) {
	res, err := s.SendCode(ctx, token, code)
	if err != nil {
		return nil, err
	}
	result := types.ParseCheckCodeResponse(res)
	return &result, nil
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

func (s *AuthService) CheckPasswordResult(ctx context.Context, trackID, password string) (*types.CheckPasswordResponse, error) {
	res, err := s.CheckPassword(ctx, trackID, password)
	if err != nil {
		return nil, err
	}
	result := types.ParseCheckPasswordResponse(res)
	return &result, nil
}

// RequestQR requests a fresh QR login challenge.
func (s *AuthService) RequestQR(ctx context.Context) (map[string]interface{}, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpGetQr, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("request qr failed: %w", err)
	}
	return res, nil
}

func (s *AuthService) RequestQRResult(ctx context.Context) (*types.RequestQRResponse, error) {
	res, err := s.RequestQR(ctx)
	if err != nil {
		return nil, err
	}
	result := types.ParseRequestQRResponse(res)
	return &result, nil
}

// CheckQR polls a QR login challenge by track ID.
func (s *AuthService) CheckQR(ctx context.Context, trackID string) (map[string]interface{}, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpGetQrStatus, map[string]interface{}{"trackId": trackID})
	if err != nil {
		return nil, fmt.Errorf("check qr failed: %w", err)
	}
	return res, nil
}

func (s *AuthService) CheckQRResult(ctx context.Context, trackID string) (*types.CheckQRResponse, error) {
	res, err := s.CheckQR(ctx, trackID)
	if err != nil {
		return nil, err
	}
	result := types.ParseCheckQRResponse(res)
	return &result, nil
}

// ConfirmQR finalizes a scanned and approved QR challenge.
func (s *AuthService) ConfirmQR(ctx context.Context, trackID string) (map[string]interface{}, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpLoginByQr, map[string]interface{}{"trackId": trackID})
	if err != nil {
		return nil, fmt.Errorf("confirm qr failed: %w", err)
	}
	return res, nil
}

func (s *AuthService) ConfirmQRResult(ctx context.Context, trackID string) (*types.CheckCodeResponse, error) {
	res, err := s.ConfirmQR(ctx, trackID)
	if err != nil {
		return nil, err
	}
	result := types.ParseCheckCodeResponse(res)
	return &result, nil
}

// ApproveQR approves a QR link from an already authenticated mobile client.
func (s *AuthService) ApproveQR(ctx context.Context, qrLink string) error {
	if _, err := s.invoker.Invoke(ctx, protocol.OpAuthQrApprove, map[string]interface{}{"qrLink": qrLink}); err != nil {
		return fmt.Errorf("approve qr failed: %w", err)
	}
	return nil
}

// AuthorizeQRLogin is the PyMax-compatible name for approving a QR login.
func (s *AuthService) AuthorizeQRLogin(ctx context.Context, qrLink string) error {
	return s.ApproveQR(ctx, qrLink)
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
func (s *AuthService) CommitTwoFactor(ctx context.Context, trackID, password, hint string, capabilities []TwoFactorAction) error {
	payload := map[string]interface{}{
		"trackId": trackID, "password": password,
		"expectedCapabilities": capabilities,
	}
	if hint != "" {
		payload["hint"] = hint
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

func (s *AuthService) ConfirmRegistrationResult(ctx context.Context, firstName, lastName, token string) (*types.ConfirmRegistrationResponse, error) {
	res, err := s.ConfirmRegistration(ctx, firstName, lastName, token)
	if err != nil {
		return nil, err
	}
	result := types.ParseConfirmRegistrationResponse(res)
	return &result, nil
}

// Login exposes the authenticated LOGIN operation for custom client flows.
func (s *AuthService) Login(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	return s.invoker.Invoke(ctx, protocol.OpLogin, payload)
}

// MobileLogin and WebLogin expose explicit PyMax-compatible names while
// allowing callers to supply the corresponding sync payload.
func (s *AuthService) MobileLogin(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	return s.Login(ctx, payload)
}

func (s *AuthService) WebLogin(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	return s.Login(ctx, payload)
}

func (s *AuthService) LoginResult(ctx context.Context, payload map[string]interface{}) (*types.LoginResponse, error) {
	res, err := s.Login(ctx, payload)
	if err != nil {
		return nil, err
	}
	result := types.ParseLoginResponse(res)
	return &result, nil
}

func (s *AuthService) Login2Result(ctx context.Context, needProfile bool, contactsSync int64, configHash string) (*types.Login2Response, error) {
	res, err := s.Login2(ctx, needProfile, contactsSync, configHash)
	if err != nil {
		return nil, err
	}
	result := types.ParseLogin2Response(res)
	return &result, nil
}

// MobileLogin2 is the PyMax-compatible typed LOGIN2 entry point.
func (s *AuthService) MobileLogin2(ctx context.Context, flags types.Login2Flags, contactsSync int64, configHash string) (*types.Login2Response, error) {
	if !flags.ContactEnabled {
		contactsSync = -1
	}
	return s.Login2Result(ctx, flags.ProfileEnabled, contactsSync, configHash)
}

func (s *AuthService) IsUpdateAvailable(handshake *types.HandshakeResponse) bool {
	return handshake != nil && handshake.AppUpdateType != 0
}

// Login2 completes the optional profile/contact/config synchronization stage.
func (s *AuthService) Login2(ctx context.Context, needProfile bool, contactsSync int64, configHash string) (map[string]interface{}, error) {
	return s.invoker.Invoke(ctx, protocol.OpLogin2, map[string]interface{}{"needProfile": needProfile, "contactsSync": contactsSync, "configHash": configHash})
}

func (s *AuthService) SetTwoFactor(ctx context.Context, password, hint string) error {
	trackID, err := s.CreateAuthTrack(ctx)
	if err != nil {
		return err
	}
	if err := s.SetPassword(ctx, trackID, password); err != nil {
		return err
	}
	capabilities := []TwoFactorAction{TwoFactorSetPassword}
	if hint != "" {
		if err := s.SetHint(ctx, trackID, hint); err != nil {
			return err
		}
		capabilities = append(capabilities, TwoFactorHint)
	}
	return s.CommitTwoFactor(ctx, trackID, password, hint, capabilities)
}

func (s *AuthService) SetTwoFactorWithEmail(ctx context.Context, password, hint, email string, codeProvider func(context.Context) (string, error)) error {
	trackID, err := s.CreateAuthTrack(ctx)
	if err != nil {
		return err
	}
	if err := s.SetPassword(ctx, trackID, password); err != nil {
		return err
	}
	capabilities := []TwoFactorAction{TwoFactorSetPassword}
	if hint != "" {
		if err := s.SetHint(ctx, trackID, hint); err != nil {
			return err
		}
		capabilities = append(capabilities, TwoFactorHint)
	}
	if email != "" {
		if err := s.RequestEmailCode(ctx, trackID, email); err != nil {
			return err
		}
		if codeProvider == nil {
			return fmt.Errorf("2fa email code provider is required")
		}
		verificationCode, err := codeProvider(ctx)
		if err != nil {
			return err
		}
		if verificationCode == "" {
			return fmt.Errorf("2fa email verification code is required")
		}
		if err := s.VerifyEmailCode(ctx, trackID, verificationCode); err != nil {
			return err
		}
		capabilities = append(capabilities, TwoFactorEmail)
	}
	return s.CommitTwoFactor(ctx, trackID, password, hint, capabilities)
}

func (s *AuthService) RemoveTwoFactor(ctx context.Context, currentPassword string) error {
	trackID, err := s.CreateAuthTrack(ctx)
	if err != nil {
		return err
	}
	if _, err := s.invoker.Invoke(ctx, protocol.OpAuthCheckPassword, map[string]interface{}{"trackId": trackID, "password": currentPassword}); err != nil {
		return err
	}
	_, err = s.invoker.Invoke(ctx, protocol.OpAuthSet2Fa, map[string]interface{}{"trackId": trackID, "remove2fa": true, "expectedCapabilities": []TwoFactorAction{TwoFactorRemove}})
	return err
}

func (s *AuthService) ChangePassword(ctx context.Context, oldPassword, newPassword string) error {
	trackID, err := s.CreateAuthTrack(ctx)
	if err != nil {
		return err
	}
	if _, err := s.invoker.Invoke(ctx, protocol.OpAuthCheckPassword, map[string]interface{}{"trackId": trackID, "password": oldPassword}); err != nil {
		return err
	}
	if err := s.SetPassword(ctx, trackID, newPassword); err != nil {
		return err
	}
	return s.CommitTwoFactor(ctx, trackID, newPassword, "", []TwoFactorAction{TwoFactorUpdatePassword})
}

func (s *AuthService) CheckTwoFactor(ctx context.Context) (bool, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpAuth2FaDetails, map[string]interface{}{})
	if err != nil {
		return false, err
	}
	for _, key := range []string{"enabled", "passwordEnabled", "secondFactorPasswordEnabled"} {
		if enabled, ok := res[key].(bool); ok {
			return enabled, nil
		}
	}
	return false, nil
}
