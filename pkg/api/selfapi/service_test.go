package selfapi

import (
	"context"
	"testing"

	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
)

type recordingInvoker struct {
	calls   int
	op      protocol.Opcode
	payload map[string]interface{}
	result  map[string]interface{}
}

func (i *recordingInvoker) Invoke(_ context.Context, op protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
	i.calls++
	i.op = op
	i.payload = payload.(map[string]interface{})
	return i.result, nil
}

func TestChangeProfileSendsAvatarTypeOnce(t *testing.T) {
	invoker := &recordingInvoker{result: map[string]interface{}{"profile": map[string]interface{}{"contact": map[string]interface{}{"id": int64(9)}}}}
	user, err := NewSelfService(invoker).ChangeProfileResult(context.Background(), "Ink", "", "hello", "")
	if err != nil {
		t.Fatal(err)
	}
	if invoker.calls != 1 || invoker.payload["avatarType"] != "USER_AVATAR" || user.ID != 9 {
		t.Fatalf("calls=%d payload=%#v user=%#v", invoker.calls, invoker.payload, user)
	}
}

func TestUpdateFolderIncludesEmptyOptions(t *testing.T) {
	invoker := &recordingInvoker{result: map[string]interface{}{"folder": map[string]interface{}{"id": "folder", "title": "Work"}, "folderSync": int64(3)}}
	update, err := NewSelfService(invoker).UpdateFolderResult(context.Background(), "folder", "Work", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := invoker.payload["options"].([]interface{}); !ok || update.FolderSync != 3 {
		t.Fatalf("payload=%#v update=%#v", invoker.payload, update)
	}
}
