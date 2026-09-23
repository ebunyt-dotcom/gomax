// Package sessionapi exposes SESSION_INIT handshake helpers.
package sessionapi

import (
	"context"
	"crypto/rand"
	"encoding/binary"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

type Service struct{ invoker api.Invoker }

func NewService(invoker api.Invoker) *Service { return &Service{invoker: invoker} }

func (s *Service) Handshake(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	return s.invoker.Invoke(ctx, protocol.OpSessionInit, payload)
}

func (s *Service) MobileHandshake(ctx context.Context, mtInstanceID string, userAgent map[string]interface{}, deviceID string, clientSessionID int) (map[string]interface{}, error) {
	if clientSessionID <= 0 {
		clientSessionID = randomClientSessionID()
	}
	return s.Handshake(ctx, map[string]interface{}{
		"mt_instanceid": mtInstanceID, "userAgent": userAgent, "deviceId": deviceID, "clientSessionId": clientSessionID,
	})
}

func (s *Service) MobileHandshakeResult(ctx context.Context, mtInstanceID string, userAgent types.MobileUserAgent, deviceID string, clientSessionID int) (*types.HandshakeResponse, error) {
	payload := userAgentPayload(userAgent, false)
	res, err := s.MobileHandshake(ctx, mtInstanceID, payload, deviceID, clientSessionID)
	if err != nil {
		return nil, err
	}
	return parseHandshake(res), nil
}

func (s *Service) WebHandshake(ctx context.Context, userAgent map[string]interface{}, deviceID string) (map[string]interface{}, error) {
	return s.Handshake(ctx, map[string]interface{}{"userAgent": userAgent, "deviceId": deviceID})
}

func (s *Service) WebHandshakeResult(ctx context.Context, userAgent types.MobileUserAgent, deviceID string) (*types.HandshakeResponse, error) {
	res, err := s.WebHandshake(ctx, userAgentPayload(userAgent, true), deviceID)
	if err != nil {
		return nil, err
	}
	return parseHandshake(res), nil
}

func randomClientSessionID() int {
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return 1
	}
	return int(binary.LittleEndian.Uint64(value[:])%70) + 1
}

func userAgentPayload(userAgent types.MobileUserAgent, web bool) map[string]interface{} {
	payload := map[string]interface{}{
		"deviceType": userAgent.DeviceType, "appVersion": userAgent.AppVersion,
		"osVersion": userAgent.OSVersion, "timezone": userAgent.Timezone,
		"screen": userAgent.Screen, "locale": userAgent.Locale,
		"deviceName": userAgent.DeviceName, "deviceLocale": userAgent.DeviceLocale,
	}
	optional := map[string]interface{}{
		"pushDeviceType": userAgent.PushDeviceType, "arch": userAgent.Arch,
		"buildNumber": userAgent.BuildNumber, "release": userAgent.Release,
		"headerUserAgent": userAgent.HeaderUserAgent,
	}
	for key, value := range optional {
		switch typed := value.(type) {
		case string:
			if typed != "" {
				payload[key] = typed
			}
		case int:
			if typed != 0 {
				payload[key] = typed
			}
		}
	}
	if web {
		delete(payload, "pushDeviceType")
		delete(payload, "arch")
		delete(payload, "buildNumber")
		delete(payload, "release")
		if _, ok := payload["headerUserAgent"]; !ok {
			payload["headerUserAgent"] = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"
		}
	}
	return payload
}

func parseHandshake(raw map[string]interface{}) *types.HandshakeResponse {
	callsSeed, _ := types.Int64Value(raw["callsSeed"])
	appUpdateType, _ := types.Int64Value(raw["app-update-type"])
	return &types.HandshakeResponse{CallsSeed: callsSeed, AppUpdateType: appUpdateType}
}
