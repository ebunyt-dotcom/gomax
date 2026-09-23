package chats

import (
	"context"
	"testing"

	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
)

type recordingInvoker struct {
	op      protocol.Opcode
	payload map[string]interface{}
	result  map[string]interface{}
}

func (i *recordingInvoker) Invoke(_ context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
	i.op = op
	i.payload = payload.(map[string]interface{})
	return i.result, nil
}

func TestJoinGroupNormalizesFullLink(t *testing.T) {
	invoker := &recordingInvoker{result: map[string]interface{}{"chat": map[string]interface{}{"id": int64(7)}}}
	chat, err := NewChatService(invoker).JoinGroup(context.Background(), "https://max.ru/join/secret")
	if err != nil {
		t.Fatal(err)
	}
	if invoker.payload["link"] != "join/secret" || chat.ID != 7 {
		t.Fatalf("payload=%#v chat=%#v", invoker.payload, chat)
	}
}

func TestReturnedChatHasBoundConvenienceMethods(t *testing.T) {
	invoker := &recordingInvoker{result: map[string]interface{}{"chat": map[string]interface{}{"id": int64(7), "type": "CHAT", "lastEventTime": int64(12)}}}
	chat, err := NewChatService(invoker).JoinGroup(context.Background(), "join/secret")
	if err != nil {
		t.Fatal(err)
	}
	invoker.result = map[string]interface{}{}
	if err := chat.Delete(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if invoker.op != protocol.OpChatDelete || invoker.payload["lastEventTime"] != int64(12) || invoker.payload["forAll"] != false {
		t.Fatalf("op=%v payload=%#v", invoker.op, invoker.payload)
	}
}

func TestCreateGroupWireEnumsAndDirectMessage(t *testing.T) {
	invoker := &recordingInvoker{result: map[string]interface{}{"id": int64(11), "chatId": int64(7), "chat": map[string]interface{}{"id": int64(7), "title": "Team"}}}
	chat, message, err := NewChatService(invoker).CreateGroupWithMessage(context.Background(), "Team", []int64{3}, true)
	if err != nil {
		t.Fatal(err)
	}
	attach := invoker.payload["message"].(map[string]interface{})["attaches"].([]interface{})[0].(map[string]interface{})
	if attach["event"] != "new" || attach["_type"] != "CONTROL" || chat.ID != 7 || message.ID != 11 {
		t.Fatalf("attach=%#v chat=%#v message=%#v", attach, chat, message)
	}
}

func TestGroupSettingsNormalizesAliases(t *testing.T) {
	invoker := &recordingInvoker{}
	if err := NewChatService(invoker).ChangeGroupSettingsWithOptions(context.Background(), 5, map[string]bool{"onlyAdminCanCall": true}); err != nil {
		t.Fatal(err)
	}
	options := invoker.payload["options"].(map[string]bool)
	if !options["ONLY_ADMIN_CAN_CALL"] {
		t.Fatalf("options=%#v", options)
	}
}
