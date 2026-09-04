// Package bots contains the small bot/web-app API surface exposed by PyMax.
package bots

import (
	"context"
	"fmt"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// BotsService handles initialization of a bot web app.
type BotsService struct{ invoker api.Invoker }

// NewBotsService creates a bot web-app service backed by invoker.
func NewBotsService(invoker api.Invoker) *BotsService { return &BotsService{invoker: invoker} }

// GetInitData requests the URL and query ID needed to open a bot web app.
func (s *BotsService) GetInitData(ctx context.Context, botID, chatID int64, startParam string) (*types.InitData, error) {
	payload := map[string]interface{}{"botId": botID}
	if chatID != 0 {
		payload["chatId"] = chatID
	}
	if startParam != "" {
		payload["startParam"] = startParam
	}
	res, err := s.invoker.Invoke(ctx, protocol.OpWebAppInitData, payload)
	if err != nil {
		return nil, fmt.Errorf("get bot init data failed: %w", err)
	}
	return &types.InitData{QueryID: stringValue(res["queryId"]), URL: stringValue(res["url"])}, nil
}

func stringValue(value interface{}) string { v, _ := value.(string); return v }
