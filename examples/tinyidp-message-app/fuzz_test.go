package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func FuzzNormalizeReturnTo(f *testing.F) {
	for _, seed := range []string{"", "/", "/messages", "https://example.test/", "//example.test", "/%2fadmin", "/../admin"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		normalized, err := normalizeReturnTo(raw)
		if err != nil {
			return
		}
		if !strings.HasPrefix(normalized, "/") || strings.HasPrefix(normalized, "//") || strings.Contains(normalized, "\\") {
			t.Fatalf("accepted unsafe return path %q from %q", normalized, raw)
		}
	})
}

func FuzzDecodeCreateMessageRequest(f *testing.F) {
	for _, seed := range []string{`{"body":"hello"}`, `{"body":"<script>alert(1)</script>"}`, `{}`, `[]`, `{"body":"x"}{"body":"y"}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		request := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(raw))
		request.Header.Set("Content-Type", "application/json")
		_, _ = decodeCreateMessageRequest(httptest.NewRecorder(), request)
	})
}
