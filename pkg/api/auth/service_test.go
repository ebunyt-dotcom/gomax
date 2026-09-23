package auth

import (
	"context"
	"testing"

	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

type call struct {
	op      protocol.Opcode
	payload map[string]interface{}
}

func TestMobileLogin2AppliesFlags(t *testing.T) {
	invoker := &sequenceInvoker{}
	_, err := NewAuthService(invoker).MobileLogin2(context.Background(), types.Login2Flags{ProfileEnabled: true}, 123, "hash")
	if err != nil {
		t.Fatal(err)
	}
	payload := invoker.calls[0].payload
	if payload["needProfile"] != true || payload["contactsSync"] != int64(-1) || payload["configHash"] != "hash" {
		t.Fatalf("payload=%#v", payload)
	}
}

type sequenceInvoker struct {
	calls []call
}

func (i *sequenceInvoker) Invoke(_ context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
	i.calls = append(i.calls, call{op: op, payload: payload.(map[string]interface{})})
	if op == protocol.OpAuthCreateTrack {
		return map[string]interface{}{"trackId": "track"}, nil
	}
	return map[string]interface{}{}, nil
}

func TestSetTwoFactorUsesNumericCapabilities(t *testing.T) {
	invoker := &sequenceInvoker{}
	service := NewAuthService(invoker)
	if err := service.SetTwoFactor(context.Background(), "secret", "hint"); err != nil {
		t.Fatal(err)
	}
	last := invoker.calls[len(invoker.calls)-1]
	capabilities, ok := last.payload["expectedCapabilities"].([]TwoFactorAction)
	if !ok || len(capabilities) != 2 || capabilities[0] != TwoFactorSetPassword || capabilities[1] != TwoFactorHint {
		t.Fatalf("capabilities=%#v", last.payload["expectedCapabilities"])
	}
}

func TestRemoveTwoFactorUsesNumericCapability(t *testing.T) {
	invoker := &sequenceInvoker{}
	service := NewAuthService(invoker)
	if err := service.RemoveTwoFactor(context.Background(), "secret"); err != nil {
		t.Fatal(err)
	}
	last := invoker.calls[len(invoker.calls)-1]
	capabilities, ok := last.payload["expectedCapabilities"].([]TwoFactorAction)
	if !ok || len(capabilities) != 1 || capabilities[0] != TwoFactorRemove {
		t.Fatalf("capabilities=%#v", last.payload["expectedCapabilities"])
	}
}
