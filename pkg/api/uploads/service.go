package uploads

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// UploadService handles binary media and document uploads to Max servers.
type UploadService struct {
	invoker    api.Invoker
	httpClient *http.Client
}

// NewUploadService creates a new UploadService instance.
func NewUploadService(invoker api.Invoker) *UploadService {
	return &UploadService{
		invoker:    invoker,
		httpClient: &http.Client{},
	}
}

// UploadPhoto uploads an image and returns a Photo Attachment.
func (s *UploadService) UploadPhoto(ctx context.Context, data []byte, fileName string) (*types.Attachment, error) {
	payload := map[string]interface{}{
		"count": 1,
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpPhotoUpload, payload)
	if err != nil {
		return nil, fmt.Errorf("request photo upload url failed: %w", err)
	}

	uploadURL, _ := res["url"].(string)
	token, _ := res["token"].(string)

	if uploadURL != "" {
		if err := s.httpPost(ctx, uploadURL, "photo", fileName, data); err != nil {
			return nil, fmt.Errorf("photo upload http failed: %w", err)
		}
	}

	return &types.Attachment{
		Type:     types.AttachmentPhoto,
		Token:    token,
		FileName: fileName,
		FileSize: int64(len(data)),
	}, nil
}

// UploadVideo requests a video slot and uploads video data.
func (s *UploadService) UploadVideo(ctx context.Context, data []byte, fileName string, duration int) (*types.Attachment, error) {
	payload := map[string]interface{}{
		"count": 1,
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpVideoUpload, payload)
	if err != nil {
		return nil, fmt.Errorf("request video upload url failed: %w", err)
	}

	uploadURL, _ := res["url"].(string)
	token, _ := res["token"].(string)

	if uploadURL != "" {
		if err := s.httpPost(ctx, uploadURL, "video", fileName, data); err != nil {
			return nil, fmt.Errorf("video upload http failed: %w", err)
		}
	}

	return &types.Attachment{
		Type:     types.AttachmentVideo,
		Token:    token,
		FileName: fileName,
		FileSize: int64(len(data)),
		Duration: duration,
	}, nil
}

// UploadFile uploads an arbitrary document or file.
func (s *UploadService) UploadFile(ctx context.Context, data []byte, fileName string) (*types.Attachment, error) {
	payload := map[string]interface{}{
		"count": 1,
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpFileUpload, payload)
	if err != nil {
		return nil, fmt.Errorf("request file upload url failed: %w", err)
	}

	uploadURL, _ := res["url"].(string)
	token, _ := res["token"].(string)

	if uploadURL != "" {
		if err := s.httpPost(ctx, uploadURL, "file", fileName, data); err != nil {
			return nil, fmt.Errorf("file upload http failed: %w", err)
		}
	}

	return &types.Attachment{
		Type:     types.AttachmentFile,
		Token:    token,
		FileName: fileName,
		FileSize: int64(len(data)),
	}, nil
}

// UploadVoice uploads voice audio recording.
func (s *UploadService) UploadVoice(ctx context.Context, data []byte, duration int) (*types.Attachment, error) {
	att, err := s.UploadFile(ctx, data, "voice.ogg")
	if err != nil {
		return nil, err
	}
	att.Type = types.AttachmentVoice
	att.Duration = duration
	return att, nil
}

func (s *UploadService) httpPost(ctx context.Context, targetURL, fieldName, fileName string, data []byte) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http upload returned status %s", resp.Status)
	}
	return nil
}
