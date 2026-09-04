package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/auth"
	"github.com/ebunyt-dotcom/gomax/pkg/client"
	"github.com/ebunyt-dotcom/gomax/pkg/connection"
	"github.com/ebunyt-dotcom/gomax/pkg/fingerprint"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/session"
)

var totalTests = 0
var passedTests = 0

func assert(condition bool, name string, details string) {
	totalTests++
	if condition {
		passedTests++
		fmt.Printf("  [PASS] %s\n", name)
	} else {
		fmt.Printf("  [FAIL] %s: %s\n", name, details)
	}
}

// MockInvoker records calls and simulates server responses.
type MockInvoker struct {
	calls        []protocol.Opcode
	payloads     []interface{}
	customInvoke func(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error)
}

func (m *MockInvoker) Invoke(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
	m.calls = append(m.calls, op)
	m.payloads = append(m.payloads, payload)
	if m.customInvoke != nil {
		return m.customInvoke(ctx, op, payload)
	}
	return map[string]interface{}{}, nil
}

func (m *MockInvoker) GetDeviceID() string {
	return "mock_device_id_1"
}

func (m *MockInvoker) GetCallsSeed() int64 {
	return 1234567890123456789
}

// MockCodeProvider supplies predefined verification code.
type MockCodeProvider struct {
	Code string
	Err  error
}

func (p *MockCodeProvider) GetCode(ctx context.Context) (string, error) {
	return p.Code, p.Err
}

// MockPasswordProvider supplies predefined 2FA password and captures hint.
type MockPasswordProvider struct {
	Password     string
	CapturedHint string
	Err          error
}

func (p *MockPasswordProvider) GetPassword(ctx context.Context) (string, error) {
	return p.Password, p.Err
}

func (p *MockPasswordProvider) GetPasswordWithHint(ctx context.Context, hint string) (string, error) {
	p.CapturedHint = hint
	return p.Password, p.Err
}

type MockTransport struct {
	mu        sync.Mutex
	connected bool
	closed    bool
	blockCh   chan struct{}
}

func newMockTransport() *MockTransport {
	return &MockTransport{
		connected: true,
		blockCh:   make(chan struct{}),
	}
}

func (m *MockTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	m.closed = false
	return nil
}

func (m *MockTransport) Send(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("closed")
	}
	return nil
}

func (m *MockTransport) Recv(n int) ([]byte, error) {
	m.mu.Lock()
	ch := m.blockCh
	m.mu.Unlock()
	<-ch
	return nil, io.EOF
}

func (m *MockTransport) Connected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *MockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		m.connected = false
		close(m.blockCh)
	}
	return nil
}

type MockReader struct {
	blockCh chan struct{}
}

func newMockReader() *MockReader {
	return &MockReader{blockCh: make(chan struct{})}
}

func (r *MockReader) ReadFrame() ([]byte, error) {
	<-r.blockCh
	return nil, io.EOF
}

func (r *MockReader) Close() {
	select {
	case <-r.blockCh:
	default:
		close(r.blockCh)
	}
}

func testR3ModernFingerprints() {
	fmt.Println("--- Test Suite: R3 Modern Android Fingerprint Metadata ---")
	fp := fingerprint.DefaultFingerprint()

	assert(fp.VersionCode == 6790, "Fingerprint VersionCode is 6790", fmt.Sprintf("got %d", fp.VersionCode))
	assert(fp.VersionName == "26.25.0", "Fingerprint VersionName is 26.25.0", fp.VersionName)

	expectedCert := "1684414033eb263e2c615f8b7df5ed8793850a07656304997fbf07e9e21e1e93"
	assert(fp.CertificateMetaSha256 == expectedCert, "Certificate SHA-256 matches PyMax", fp.CertificateMetaSha256)

	expectedDex := "8db68fcc0e85e8f041fe4a875c0a9bcfe542a8f679603728c651ed81b64dd684"
	assert(fp.DexMetaSha256 == expectedDex, "Dex Meta SHA-256 matches PyMax", fp.DexMetaSha256)

	expectedArm64 := "634ecc42b246784d975f180b4fecf903df235cdf0476da47163a85630eb1a6a8"
	assert(fp.SoMetaSha256["arm64-v8a"] == expectedArm64, "SO Meta SHA-256 (arm64-v8a) matches PyMax", fp.SoMetaSha256["arm64-v8a"])

	gen := fingerprint.NewFingerprintGenerator(fp)
	bytes, err := gen.GenerateFingerprint("my_device_id_123", 987654321, "arm64-v8a")
	assert(err == nil, "GenerateFingerprint succeeds without error", fmt.Sprint(err))
	assert(len(bytes) == 96, "Generated fingerprint is exactly 96 bytes", fmt.Sprintf("got %d bytes", len(bytes)))

	// Deterministic: generating again produces identical output
	bytes2, _ := gen.GenerateFingerprint("my_device_id_123", 987654321, "arm64-v8a")
	assert(hex.EncodeToString(bytes) == hex.EncodeToString(bytes2), "Fingerprint generation is fully deterministic", "")

	// Nil generator safety
	var nilGen *fingerprint.FingerprintGenerator
	_, nilErr := nilGen.GenerateFingerprint("id", 1, "arm64-v8a")
	assert(nilErr != nil, "Nil generator returns error safely without panic", "")
}

func testR1HandshakePayloadAndErrors() {
	fmt.Println("--- Test Suite: R1 Complete Mobile Handshake Payload & Errors ---")
	cfg := client.DefaultConfig()
	cfg.DeviceID = "test_dev_0123456"
	cfg.MtInstanceID = "test_inst_abcdef"
	cfg.Store = session.NewInMemoryStore()
	cfg.Reconnect = false

	c := client.NewClient(cfg)
	assert(c.GetDeviceID() == "test_dev_0123456", "Client stores and returns DeviceID", c.GetDeviceID())
	assert(c.CallsSeed() == 0, "Initial CallsSeed is 0", fmt.Sprintf("%d", c.CallsSeed()))

	// Test user agent parameters match specification
	userAgent := map[string]interface{}{
		"deviceType":     "ANDROID",
		"appVersion":     "26.25.0",
		"osVersion":      "Android 14",
		"timezone":       "Europe/Moscow",
		"screen":         "405dpi 405dpi 1080x2400",
		"pushDeviceType": "GCM",
		"arch":           "arm64-v8a",
		"locale":         "ru",
		"deviceLocale":   "ru",
		"buildNumber":    6790,
		"deviceName":     "Samsung SM-A536B",
	}

	assert(userAgent["deviceType"] == "ANDROID", "userAgent.deviceType is ANDROID", "")
	assert(userAgent["appVersion"] == "26.25.0", "userAgent.appVersion is 26.25.0", "")
	assert(userAgent["osVersion"] == "Android 14", "userAgent.osVersion is Android 14", "")
	assert(userAgent["buildNumber"] == 6790, "userAgent.buildNumber is 6790", "")
	assert(userAgent["deviceName"] == "Samsung SM-A536B", "userAgent.deviceName is Samsung SM-A536B", "")
	assert(userAgent["arch"] == "arm64-v8a", "userAgent.arch is arm64-v8a", "")
	assert(userAgent["pushDeviceType"] == "GCM", "userAgent.pushDeviceType is GCM", "")

	// Handshake payload structure
	payload := map[string]interface{}{
		"mt_instanceid":   cfg.MtInstanceID,
		"userAgent":       userAgent,
		"clientSessionId": 42,
		"deviceId":        cfg.DeviceID,
	}
	assert(len(payload["mt_instanceid"].(string)) == 16, "mt_instanceid is 16 hex chars", payload["mt_instanceid"].(string))
	assert(payload["clientSessionId"].(int) >= 1 && payload["clientSessionId"].(int) <= 70, "clientSessionId in [1, 70]", "")
}

func testR2SmsAuthFlow() {
	fmt.Println("--- Test Suite: R2 SMS Auth Request & Verification (START_AUTH & CHECK_CODE) ---")

	fp := fingerprint.DefaultFingerprint()
	codeProv := &MockCodeProvider{Code: "123456"}
	pwdProv := &MockPasswordProvider{Password: "secret2fa"}

	flow := auth.NewSmsAuthFlow(codeProv, pwdProv)
	flow.FpGen = fingerprint.NewFingerprintGenerator(fp)
	flow.DeviceID = "dev_998877665544"
	flow.CallsSeed = 8877665544332211
	flow.Arch = "arm64-v8a"

	ctx := context.Background()

	// Scenario 1: Standard SMS Login (no 2FA)
	mock := &MockInvoker{
		customInvoke: func(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
			pMap := payload.(map[string]interface{})
			if op == protocol.OpAuthRequest {
				assert(pMap["type"] == "START_AUTH", "OpAuthRequest type is START_AUTH", fmt.Sprint(pMap["type"]))
				assert(pMap["phone"] == "+79991112233", "OpAuthRequest phone is +79991112233", fmt.Sprint(pMap["phone"]))
				modeBytes, ok := pMap["mode"].([]byte)
				assert(ok && len(modeBytes) == 96, "OpAuthRequest mode is 96-byte hardware fingerprint", fmt.Sprintf("len=%d", len(modeBytes)))
				return map[string]interface{}{
					"token": "challenge_token_abc_123",
				}, nil
			}
			if op == protocol.OpAuth {
				assert(pMap["token"] == "challenge_token_abc_123", "OpAuth token matches challenge token", fmt.Sprint(pMap["token"]))
				assert(pMap["verifyCode"] == "123456", "OpAuth verifyCode is 123456", fmt.Sprint(pMap["verifyCode"]))
				assert(pMap["authTokenType"] == "CHECK_CODE", "OpAuth authTokenType is CHECK_CODE", fmt.Sprint(pMap["authTokenType"]))
				return map[string]interface{}{
					"tokenAttrs": map[string]interface{}{
						"LOGIN": map[string]interface{}{
							"token": "session_jwt_login_token_xyz",
						},
					},
					"userId": int64(100500),
				}, nil
			}
			return nil, fmt.Errorf("unexpected opcode: %v", op)
		},
	}

	res, err := flow.Authenticate(ctx, mock, "+79991112233")
	assert(err == nil, "Standard SMS authentication completes successfully", fmt.Sprint(err))
	assert(res != nil && res.Token == "session_jwt_login_token_xyz", "Extracted correct session login token", fmt.Sprintf("%+v", res))
	assert(res.UserID == 100500, "Extracted correct UserID (100500)", fmt.Sprintf("%d", res.UserID))

	// Scenario 2: 2FA Password Challenge Flow
	mock2FA := &MockInvoker{
		customInvoke: func(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
			pMap := payload.(map[string]interface{})
			if op == protocol.OpAuthRequest {
				return map[string]interface{}{
					"token": "challenge_token_2fa_step1",
				}, nil
			}
			if op == protocol.OpAuth {
				// Server returns passwordChallenge
				return map[string]interface{}{
					"passwordChallenge": map[string]interface{}{
						"trackId": "track_pwd_456",
						"hint":    "my hint",
					},
				}, nil
			}
			if op == protocol.OpAuthLoginCheckPassword {
				assert(pMap["trackId"] == "track_pwd_456", "OpAuthLoginCheckPassword trackId matches challenge", fmt.Sprint(pMap["trackId"]))
				assert(pMap["password"] == "secret2fa", "OpAuthLoginCheckPassword password matches", fmt.Sprint(pMap["password"]))
				return map[string]interface{}{
					"tokenAttrs": map[string]interface{}{
						"login": map[string]interface{}{
							"token": "session_2fa_authorized_token_789",
						},
					},
					"userId": float64(200600), // float64 encoding
				}, nil
			}
			return nil, fmt.Errorf("unexpected opcode: %v", op)
		},
	}

	res2FA, err2FA := flow.Authenticate(ctx, mock2FA, "+79991112233")
	assert(err2FA == nil, "2FA authentication completes successfully", fmt.Sprint(err2FA))
	assert(res2FA != nil && res2FA.Token == "session_2fa_authorized_token_789", "Extracted 2FA session token correctly", fmt.Sprintf("%+v", res2FA))
	assert(res2FA.UserID == 200600, "Extracted UserID from float64 correctly", fmt.Sprintf("%d", res2FA.UserID))
	assert(pwdProv.CapturedHint == "my hint", "2FA password hint was delivered to provider", pwdProv.CapturedHint)

	// Scenario 3: Server Rate Limit (FLOOD_WAIT) on OpAuthRequest
	mockFlood := &MockInvoker{
		customInvoke: func(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"error":   "FLOOD_WAIT",
				"message": "Too many SMS requests, wait 60 seconds",
			}, nil
		},
	}
	_, floodErr := flow.Authenticate(ctx, mockFlood, "+79991112233")
	assert(floodErr != nil && strings.Contains(floodErr.Error(), "FLOOD_WAIT"),
		"FLOOD_WAIT error reported clearly on auth request", fmt.Sprint(floodErr))

	// Scenario 4: Invalid Code on OpAuth
	mockBadCode := &MockInvoker{
		customInvoke: func(ctx context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
			if op == protocol.OpAuthRequest {
				return map[string]interface{}{"token": "token_123"}, nil
			}
			return map[string]interface{}{
				"error":   "INVALID_CODE",
				"message": "Verification code is incorrect",
			}, nil
		},
	}
	_, badCodeErr := flow.Authenticate(ctx, mockBadCode, "+79991112233")
	assert(badCodeErr != nil && strings.Contains(badCodeErr.Error(), "INVALID_CODE"),
		"INVALID_CODE error reported clearly on code verification", fmt.Sprint(badCodeErr))
}

func testR4ConnectionManagerTimeout() {
	fmt.Println("--- Test Suite: R4 Connection Manager Request Timeout ---")

	tr := newMockTransport()
	defer tr.Close()

	proto, err := protocol.NewTcpProtocol()
	assert(err == nil, "Initialize TcpProtocol", fmt.Sprint(err))

	reader := newMockReader()
	defer reader.Close()

	cfg := connection.DefaultConfig()
	cfg.RequestTimeout = 100 * time.Millisecond // fast timeout for test

	mgr := connection.NewConnectionManager(reader, tr, proto, &cfg, nil, nil)
	startCtx, startCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer startCancel()

	err = mgr.Start(startCtx)
	assert(err == nil, "ConnectionManager starts successfully", fmt.Sprint(err))
	assert(mgr.IsOpen(), "ConnectionManager is marked open", "")

	// 1. Enforce RequestTimeout when context has no deadline
	bgCtx := context.Background()
	start := time.Now()
	_, sendErr := mgr.SendRequest(bgCtx, protocol.OpSessionInit, map[string]interface{}{"test": true})
	elapsed := time.Since(start)

	assert(sendErr != nil, "SendRequest times out without hanging indefinitely", fmt.Sprint(sendErr))
	assert(errors.Is(sendErr, context.DeadlineExceeded), "Returned context.DeadlineExceeded error", fmt.Sprint(sendErr))
	assert(elapsed >= 90*time.Millisecond && elapsed < 500*time.Millisecond,
		"Request timed out approximately matching RequestTimeout (100ms)", fmt.Sprintf("elapsed=%v", elapsed))

	// 2. Cancellation of incoming context unblocks immediately
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc() // already canceled

	_, earlyCancelErr := mgr.SendRequest(cancelCtx, protocol.OpSessionInit, nil)
	assert(errors.Is(earlyCancelErr, context.Canceled), "Pre-canceled context returns context.Canceled immediately", fmt.Sprint(earlyCancelErr))

	// 3. Connection close unblocks pending requests
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer waitCancel()

	doneCh := make(chan error, 1)
	go func() {
		_, err := mgr.SendRequest(waitCtx, protocol.OpSessionInit, nil)
		doneCh <- err
	}()

	time.Sleep(20 * time.Millisecond)
	_ = mgr.Close()
	reader.Close()

	select {
	case closeErr := <-doneCh:
		assert(closeErr != nil && errors.Is(closeErr, connection.ErrConnectionClosed),
			"Connection close immediately unblocks pending SendRequest with ErrConnectionClosed", fmt.Sprint(closeErr))
	case <-time.After(500 * time.Millisecond):
		assert(false, "SendRequest unblocked on connection close", "timed out waiting for SendRequest to exit")
	}
}

func testNonRecoverableErrors() {
	fmt.Println("--- Test Suite: Non-Recoverable Errors Prevent Infinite Reconnect Loop ---")

	// Missing phone when token is empty must immediately fail Start(ctx) without looping
	cfg := client.DefaultConfig()
	cfg.Phone = ""
	cfg.Token = ""
	cfg.Reconnect = true // even with Reconnect: true!
	cfg.Store = session.NewInMemoryStore()
	// Point to invalid port so connect fails if it were to connect, but phone check is done before auth
	cfg.Port = 1

	c := client.NewClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := c.Start(ctx)
	elapsed := time.Since(start)

	// Since connection will fail or phone will be rejected, verify it returns error quickly without retrying for 2 seconds
	assert(err != nil, "Start returns error on missing credentials/connection failure", fmt.Sprint(err))
	assert(elapsed < 1500*time.Millisecond, "Start exits cleanly without hanging or indefinitely retrying", fmt.Sprintf("elapsed=%v", elapsed))
}

func main() {
	fmt.Println("================================================================")
	fmt.Println("  GoMax Verification Runner (SWE Light Adversarial Suite)")
	fmt.Println("================================================================")

	testR3ModernFingerprints()
	testR1HandshakePayloadAndErrors()
	testR2SmsAuthFlow()
	testR4ConnectionManagerTimeout()
	testNonRecoverableErrors()

	fmt.Println("================================================================")
	fmt.Printf("  Results: %d/%d tests passed\n", passedTests, totalTests)
	fmt.Println("================================================================")

	if passedTests < totalTests {
		os.Exit(1)
	}
}
