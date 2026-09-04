package uploads

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

const uploadReadyTimeout = 60 * time.Second

// UploadService implements PyMax's two-phase upload protocol.
// Photos use multipart/form-data and return their token in the HTTP JSON
// response. Videos, voice notes and files use a raw HTTP stream and complete
// only after Max emits NOTIF_ATTACH with the corresponding ID.
type UploadService struct {
	invoker    api.Invoker
	httpClient *http.Client

	mu           sync.Mutex
	videoWaiters map[int64]chan struct{}
	fileWaiters  map[int64]chan struct{}
	voiceWaiters map[int64]chan struct{}
}

// NewUploadService creates a new UploadService instance.
func NewUploadService(invoker api.Invoker) *UploadService {
	return &UploadService{
		invoker:      invoker,
		httpClient:   &http.Client{},
		videoWaiters: make(map[int64]chan struct{}),
		fileWaiters:  make(map[int64]chan struct{}),
		voiceWaiters: make(map[int64]chan struct{}),
	}
}

// UploadPhoto uploads an image and returns a photo attachment.
func (s *UploadService) UploadPhoto(ctx context.Context, data []byte, fileName string) (*types.Attachment, error) {
	return s.UploadPhotoWithOptions(ctx, data, fileName, false)
}

// UploadPhotoWithOptions uploads an image, optionally as a profile avatar.
func (s *UploadService) UploadPhotoWithOptions(ctx context.Context, data []byte, fileName string, profile bool) (*types.Attachment, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpPhotoUpload, map[string]interface{}{
		"count": 1, "profile": profile,
	})
	if err != nil {
		return nil, fmt.Errorf("request photo upload url failed: %w", err)
	}
	uploadURL := stringValue(res["url"])
	if uploadURL == "" {
		return nil, fmt.Errorf("photo upload response did not contain url")
	}

	photoID, err := queryValue(uploadURL, "photoIds")
	if err != nil {
		return nil, fmt.Errorf("parse photo upload id: %w", err)
	}
	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".bin"
	}
	contentType := mime.TypeByExtension(strings.ToLower(ext))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	result, err := s.postMultipart(ctx, uploadURL, "image"+ext, contentType, data)
	if err != nil {
		return nil, fmt.Errorf("photo upload http failed: %w", err)
	}

	photos, _ := result["photos"].(map[string]interface{})
	photo, _ := photos[photoID].(map[string]interface{})
	token := stringValue(photo["token"])
	if token == "" {
		return nil, fmt.Errorf("photo upload response did not contain token for photo id %s", photoID)
	}
	return &types.Attachment{
		Type: types.AttachmentPhoto, ID: photoID, Token: token,
		FileName: fileName, FileSize: int64(len(data)),
	}, nil
}

// UploadVideo requests a video slot, uploads the bytes and waits for the
// server's processing notification.
func (s *UploadService) UploadVideo(ctx context.Context, data []byte, fileName string, duration int) (*types.Attachment, error) {
	return s.uploadVideoLike(ctx, data, fileName, duration, false)
}

// UploadVoice uploads an audio/voice note through VIDEO_UPLOAD, as required
// by PyMax's protocol (voice is not a FILE_UPLOAD operation).
func (s *UploadService) UploadVoice(ctx context.Context, data []byte, duration int) (*types.Attachment, error) {
	return s.uploadVideoLike(ctx, data, "voice.ogg", duration, true)
}

func (s *UploadService) uploadVideoLike(ctx context.Context, data []byte, fileName string, duration int, voice bool) (*types.Attachment, error) {
	payload := map[string]interface{}{"count": 1}
	if voice {
		payload["type"] = 2
		payload["uploaderType"] = 1
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpVideoUpload, payload)
	if err != nil {
		return nil, fmt.Errorf("request video upload url failed: %w", err)
	}
	info, err := firstUploadInfo(res)
	if err != nil {
		return nil, fmt.Errorf("parse video upload response: %w", err)
	}
	id, err := int64Value(info["videoId"])
	if err != nil {
		return nil, fmt.Errorf("parse video id: %w", err)
	}
	uploadURL := stringValue(info["url"])
	token := stringValue(info["token"])
	if uploadURL == "" || token == "" {
		return nil, fmt.Errorf("video upload response is missing url or token")
	}

	waiter := s.registerWaiter(id, voice)
	defer s.removeWaiter(id, voice)
	if err := s.postRaw(ctx, uploadURL, fileName, data); err != nil {
		return nil, fmt.Errorf("video upload http failed: %w", err)
	}
	if err := waitReady(ctx, waiter); err != nil {
		return nil, fmt.Errorf("video processing failed: %w", err)
	}

	attachmentType := types.AttachmentVideo
	if voice {
		attachmentType = types.AttachmentVoice
	}
	return &types.Attachment{
		Type: attachmentType, ID: strconv.FormatInt(id, 10), Token: token,
		FileName: fileName, FileSize: int64(len(data)), Duration: duration,
	}, nil
}

// UploadFile uploads a document and waits for FILE_READY.
func (s *UploadService) UploadFile(ctx context.Context, data []byte, fileName string) (*types.Attachment, error) {
	res, err := s.invoker.Invoke(ctx, protocol.OpFileUpload, map[string]interface{}{"count": 1})
	if err != nil {
		return nil, fmt.Errorf("request file upload url failed: %w", err)
	}
	info, err := firstUploadInfo(res)
	if err != nil {
		return nil, fmt.Errorf("parse file upload response: %w", err)
	}
	id, err := int64Value(info["fileId"])
	if err != nil {
		return nil, fmt.Errorf("parse file id: %w", err)
	}
	uploadURL := stringValue(info["url"])
	if uploadURL == "" {
		return nil, fmt.Errorf("file upload response is missing url")
	}

	waiter := s.registerFileWaiter(id)
	defer s.removeFileWaiter(id)
	if err := s.postRaw(ctx, uploadURL, fileName, data); err != nil {
		return nil, fmt.Errorf("file upload http failed: %w", err)
	}
	if err := waitReady(ctx, waiter); err != nil {
		return nil, fmt.Errorf("file processing failed: %w", err)
	}

	return &types.Attachment{
		Type: types.AttachmentFile, ID: strconv.FormatInt(id, 10),
		Token: stringValue(info["token"]), FileName: fileName, FileSize: int64(len(data)),
	}, nil
}

// NotifyReady resolves an upload waiter from a NOTIF_ATTACH payload. It is
// called by Client/WebClient event dispatchers and is safe for concurrent use.
func (s *UploadService) NotifyReady(payload map[string]interface{}) {
	if id, err := int64Value(payload["videoId"]); err == nil {
		s.resolveWaiter(id, false)
	}
	if id, err := int64Value(payload["audioId"]); err == nil {
		s.resolveWaiter(id, true)
	}
	if id, err := int64Value(payload["fileId"]); err == nil {
		s.resolveFileWaiter(id)
	}
}

// NotifyVideoReady, NotifyVoiceReady and NotifyFileReady are explicit hooks
// for applications that route raw protocol events themselves.
func (s *UploadService) NotifyVideoReady(id int64) { s.resolveWaiter(id, false) }
func (s *UploadService) NotifyVoiceReady(id int64) { s.resolveWaiter(id, true) }
func (s *UploadService) NotifyFileReady(id int64)  { s.resolveFileWaiter(id) }

func (s *UploadService) registerWaiter(id int64, voice bool) chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan struct{})
	if voice {
		s.voiceWaiters[id] = ch
	} else {
		s.videoWaiters[id] = ch
	}
	return ch
}

func (s *UploadService) removeWaiter(id int64, voice bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if voice {
		delete(s.voiceWaiters, id)
	} else {
		delete(s.videoWaiters, id)
	}
}

func (s *UploadService) registerFileWaiter(id int64) chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan struct{})
	s.fileWaiters[id] = ch
	return ch
}

func (s *UploadService) removeFileWaiter(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.fileWaiters, id)
}

func (s *UploadService) resolveWaiter(id int64, voice bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	waiters := s.videoWaiters
	if voice {
		waiters = s.voiceWaiters
	}
	if ch, ok := waiters[id]; ok {
		delete(waiters, id)
		close(ch)
	}
}

func (s *UploadService) resolveFileWaiter(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.fileWaiters[id]; ok {
		delete(s.fileWaiters, id)
		close(ch)
	}
}

func waitReady(ctx context.Context, waiter chan struct{}) error {
	readyCtx, cancel := context.WithTimeout(ctx, uploadReadyTimeout)
	defer cancel()
	select {
	case <-waiter:
		return nil
	case <-readyCtx.Done():
		return readyCtx.Err()
	}
}

func (s *UploadService) postMultipart(ctx context.Context, targetURL, fileName, contentType string, data []byte) (map[string]interface{}, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	partHeaders := make(textproto.MIMEHeader)
	partHeaders.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
	partHeaders.Set("Content-Type", contentType)
	part, err := writer.CreatePart(partHeaders)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Preserve the source MIME type for upload gateways that inspect it.
	req.Header.Set("X-File-Content-Type", contentType)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upload returned status %s", resp.Status)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode upload response: %w", err)
	}
	return result, nil
}

func (s *UploadService) postRaw(ctx context.Context, targetURL, fileName string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	end := len(data) - 1
	req.Header.Set("Content-Disposition", "attachment; filename="+url.QueryEscape(fileName))
	req.Header.Set("Content-Length", strconv.Itoa(len(data)))
	req.Header.Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", end, len(data)))
	req.Header.Set("Connection", "keep-alive")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload returned status %s", resp.Status)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func firstUploadInfo(res map[string]interface{}) (map[string]interface{}, error) {
	list, ok := res["info"].([]interface{})
	if !ok || len(list) == 0 {
		return nil, fmt.Errorf("response info is empty")
	}
	info, ok := list[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("response info has invalid format")
	}
	return info, nil
}

func queryValue(rawURL, key string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	value := parsed.Query().Get(key)
	if value == "" {
		return "", fmt.Errorf("query parameter %s is missing", key)
	}
	return value, nil
}

func stringValue(value interface{}) string {
	v, _ := value.(string)
	return v
}

func int64Value(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case json.Number:
		return v.Int64()
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("invalid integer %T", value)
	}
}

// DecodeThumbhash decodes the optional base64 thumbhash returned for a video note.
func DecodeThumbhash(value string) ([]byte, error) {
	value += strings.Repeat("=", (-len(value))%4)
	return base64.StdEncoding.DecodeString(value)
}
