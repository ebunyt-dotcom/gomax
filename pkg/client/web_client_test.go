package client

import "testing"

func TestDefaultWebUserAgentDoesNotMutateCustomUserAgent(t *testing.T) {
	custom := map[string]interface{}{
		"deviceType":      "WEB",
		"locale":          "en",
		"deviceLocale":    "en-US",
		"osVersion":       "Linux",
		"deviceName":      "Firefox",
		"headerUserAgent": "custom-agent",
		"appVersion":      "custom-version",
		"screen":          "100x100 1.0x",
		"timezone":        "UTC",
		"buildNumber":     123,
	}
	cfg := &Config{UserAgent: custom}

	got := defaultWebUserAgent(cfg)
	if got["headerUserAgent"] != "custom-agent" {
		t.Fatalf("custom header user agent was not preserved: %#v", got)
	}
	if got["deviceName"] != "Firefox" || got["appVersion"] != "custom-version" {
		t.Fatalf("custom web user agent fields were not preserved: %#v", got)
	}
	if _, ok := got["buildNumber"]; ok {
		t.Fatalf("mobile-only buildNumber leaked into web payload: %#v", got)
	}
	if custom["buildNumber"] != 123 {
		t.Fatalf("caller-provided user agent was mutated: %#v", custom)
	}
}
