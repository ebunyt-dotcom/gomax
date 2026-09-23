package uploads

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
)

type uploadInvoker struct {
	payload map[string]interface{}
	result  map[string]interface{}
}

func (i *uploadInvoker) Invoke(_ context.Context, _ protocol.Opcode, payload interface{}) (map[string]interface{}, error) {
	i.payload = payload.(map[string]interface{})
	return i.result, nil
}

func TestRawUploadContentRangeVariants(t *testing.T) {
	var got []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		got = append(got, r.Header.Get("Content-Range"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewUploadServiceWithOptions(nil, Options{UserAgent: "gomax-test"})
	if err := service.postRaw(context.Background(), server.URL, "document.bin", []byte("abc")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.postRawResponse(context.Background(), server.URL, "video.mp4", []byte("abc")); err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got[0] != "0-2/3" || got[1] != "bytes 0-2/3" {
		t.Fatalf("unexpected content ranges: %#v", got)
	}
}

func TestPhotoUploadUsesCompleteUploadPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"photos":{"photo-1":{"token":"uploaded"}}}`)
	}))
	defer server.Close()
	invoker := &uploadInvoker{result: map[string]interface{}{"url": server.URL + "?photoIds=photo-1"}}
	attachment, err := NewUploadService(invoker).UploadPhoto(context.Background(), []byte("image"), "image.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if invoker.payload["type"] != 0 || invoker.payload["uploaderType"] != 0 || invoker.payload["profile"] != false || attachment.Token != "uploaded" {
		t.Fatalf("payload=%#v attachment=%#v", invoker.payload, attachment)
	}
}

func TestQuoteHeaderMatchesPythonQuote(t *testing.T) {
	if got := quoteHeader("OKMessages/1 (Android 13; Device)"); got != "OKMessages/1%20%28Android%2013%3B%20Device%29" {
		t.Fatalf("quoteHeader=%q", got)
	}
}
