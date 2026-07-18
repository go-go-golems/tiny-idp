package main

import (
	"strings"
	"testing"
	"time"
)

func TestCollectUsesOnlyDeviceEventsAndBoundedLabels(t *testing.T) {
	input := strings.Join([]string{
		`{"time":"2026-07-16T12:00:00Z","name":"device.authorization.created","result":"accepted","client_id":"client-that-must-not-be-a-label"}`,
		`{"time":"2026-07-16T12:00:01Z","name":"device.authorization.rejected","reason":"unreviewed-dynamic-reason"}`,
		`{"time":"2026-07-16T12:00:02Z","name":"device.verification.approved","subject":"must-not-be-a-label"}`,
		`{"time":"2026-07-16T12:00:03Z","name":"token.request.accepted","fields":{"grant_type":"urn:ietf:params:oauth:grant-type:device_code"}}`,
		`{"time":"2026-07-16T12:00:04Z","name":"token.request.accepted","fields":{"grant_type":"authorization_code"}}`,
		`not json`,
	}, "\n")
	counters, invalid, err := collect(strings.NewReader(input), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if invalid != 1 {
		t.Fatalf("invalid lines=%d, want 1", invalid)
	}
	for key, want := range map[counterKey]uint64{
		{Name: "tinyidp_device_authorizations_total", Stage: "authorization", Outcome: "created"}:               1,
		{Name: "tinyidp_device_rejections_total", Stage: "authorization", Outcome: "rejected", Reason: "other"}: 1,
		{Name: "tinyidp_device_verifications_total", Stage: "verification", Outcome: "approved"}:                1,
		{Name: "tinyidp_device_token_requests_total", Stage: "token", Outcome: "accepted", Reason: "none"}:      1,
	} {
		if got := counters[key]; got != want {
			t.Fatalf("counter %#v=%d, want %d; all=%#v", key, got, want, counters)
		}
	}
	if len(counters) != 4 {
		t.Fatalf("counter count=%d, want 4: %#v", len(counters), counters)
	}
}

func TestCollectRejectsOversizedAuditLine(t *testing.T) {
	oversized := `{"name":"` + strings.Repeat("x", maxAuditLineBytes) + `"}`
	if _, _, err := collect(strings.NewReader(oversized), time.Time{}); err == nil {
		t.Fatal("oversized audit line was accepted")
	}
}
