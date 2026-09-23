package users

import (
	"context"
	"testing"

	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

func TestImportContactsAcceptsPyMaxContactInfoList(t *testing.T) {
	invoker := &recordingInvoker{}
	if _, err := NewUserService(invoker).ImportContacts(context.Background(), []types.ContactInfo{{Phone: "+79990000000", FirstName: "Max"}}); err != nil {
		t.Fatal(err)
	}
	contacts := invoker.payload["contactList"].(map[string]interface{})
	if contacts["+79990000000"].(map[string]interface{})["firstName"] != "Max" {
		t.Fatalf("contacts=%#v", contacts)
	}
}

type recordingInvoker struct {
	op      protocol.Opcode
	payload map[string]interface{}
	result  map[string]interface{}
}

func (m *recordingInvoker) Invoke(_ context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
	m.op = op
	m.payload, _ = payload.(map[string]interface{})
	return m.result, nil
}

func TestFetchUsersUsesContactInfoWireShapeAndCaches(t *testing.T) {
	inv := &recordingInvoker{result: map[string]interface{}{"contacts": []interface{}{map[string]interface{}{"id": int64(7), "firstName": "Max"}}}}
	service := NewUserService(inv)
	users, err := service.FetchUsers(context.Background(), []int64{7})
	if err != nil {
		t.Fatal(err)
	}
	if inv.op != protocol.OpContactInfo {
		t.Fatalf("opcode=%v", inv.op)
	}
	ids, ok := inv.payload["contactIds"].([]int64)
	if !ok || len(ids) != 1 || ids[0] != 7 {
		t.Fatalf("contactIds=%#v", inv.payload["contactIds"])
	}
	if _, exists := inv.payload["userIds"]; exists {
		t.Fatal("legacy userIds key must not be sent")
	}
	if len(users) != 1 || users[0].FirstName != "Max" {
		t.Fatalf("users=%#v", users)
	}
	cached, err := service.GetCachedUser(context.Background(), 7)
	if err != nil || cached == nil || cached.FirstName != "Max" {
		t.Fatalf("cached=%#v err=%v", cached, err)
	}
}
