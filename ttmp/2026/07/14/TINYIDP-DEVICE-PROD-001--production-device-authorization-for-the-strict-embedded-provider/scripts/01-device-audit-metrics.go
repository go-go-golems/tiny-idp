// Command 01-device-audit-metrics converts tiny-idp's durable JSONL audit
// records into a deliberately small Prometheus metric surface for the device
// authorization grant. It reads a bounded local file and never emits subjects,
// client IDs, user codes, device codes, or token material as labels.
//
// Example:
//
//	go run ./ttmp/2026/07/14/TINYIDP-DEVICE-PROD-001--production-device-authorization-for-the-strict-embedded-provider/scripts/01-device-audit-metrics.go \
//	  -audit /var/log/tinyidp/audit.jsonl -since 15m
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

const maxAuditLineBytes = 1 << 20

type auditEvent struct {
	Time   time.Time `json:"time"`
	Name   string    `json:"name"`
	Result string    `json:"result"`
	Reason string    `json:"reason"`
	Fields struct {
		GrantType string `json:"grant_type"`
	} `json:"fields"`
}

type counterKey struct {
	Name    string
	Stage   string
	Outcome string
	Reason  string
}

func main() {
	auditPath := flag.String("audit", "", "path to tiny-idp JSONL audit file")
	sinceRaw := flag.String("since", "0", "include records no older than this duration; 0 includes all")
	flag.Parse()
	if *auditPath == "" {
		fmt.Fprintln(os.Stderr, "-audit is required")
		os.Exit(2)
	}
	since, err := time.ParseDuration(*sinceRaw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse -since: %v\n", err)
		os.Exit(2)
	}
	if since < 0 {
		fmt.Fprintln(os.Stderr, "-since must not be negative")
		os.Exit(2)
	}

	file, err := os.Open(*auditPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open audit: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	cutoff := time.Time{}
	if since > 0 {
		cutoff = time.Now().UTC().Add(-since)
	}
	counters, rejected, err := collect(file, cutoff)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read audit: %v\n", err)
		os.Exit(1)
	}
	writePrometheus(os.Stdout, counters, rejected)
}

func collect(r io.Reader, cutoff time.Time) (map[counterKey]uint64, uint64, error) {
	counters := make(map[counterKey]uint64)
	var rejected uint64
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4<<10), maxAuditLineBytes)
	for scanner.Scan() {
		var event auditEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			rejected++
			continue
		}
		if !cutoff.IsZero() && event.Time.Before(cutoff) {
			continue
		}
		if key, ok := deviceMetricKey(event); ok {
			counters[key]++
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, rejected, err
	}
	return counters, rejected, nil
}

func deviceMetricKey(event auditEvent) (counterKey, bool) {
	switch event.Name {
	case "device.authorization.created":
		return counterKey{Name: "tinyidp_device_authorizations_total", Stage: "authorization", Outcome: "created"}, true
	case "device.authorization.rejected":
		return counterKey{Name: "tinyidp_device_rejections_total", Stage: "authorization", Outcome: "rejected", Reason: boundedReason(event.Reason)}, true
	case "device.verification.approved":
		return counterKey{Name: "tinyidp_device_verifications_total", Stage: "verification", Outcome: "approved"}, true
	case "device.verification.denied":
		return counterKey{Name: "tinyidp_device_verifications_total", Stage: "verification", Outcome: "denied"}, true
	case "device.verification.rejected", "device.verification.unavailable", "device.verification.render_failed":
		return counterKey{Name: "tinyidp_device_rejections_total", Stage: "verification", Outcome: "rejected", Reason: boundedReason(event.Reason)}, true
	case "token.request.accepted", "token.request.rejected":
		if event.Fields.GrantType != "urn:ietf:params:oauth:grant-type:device_code" {
			return counterKey{}, false
		}
		outcome := "accepted"
		if event.Name == "token.request.rejected" {
			outcome = "rejected"
		}
		return counterKey{Name: "tinyidp_device_token_requests_total", Stage: "token", Outcome: outcome, Reason: boundedReason(event.Reason)}, true
	default:
		return counterKey{}, false
	}
}

func boundedReason(reason string) string {
	// This allow-list prevents a malformed or future error string from becoming
	// an unbounded telemetry label. Extend it only with reviewed protocol states.
	switch reason {
	case "", "authorization_pending", "slow_down", "access_denied", "expired_token", "invalid_grant",
		"invalid_scope", "invalid_audience", "device_grant_not_allowed", "client_authentication_failed",
		"rate_limited", "invalid_csrf", "missing_login", "authentication_unavailable", "renderer_failed",
		"state_transition_failed", "rate_limited_or_invalid_client", "code_collision_limit":
		if reason == "" {
			return "none"
		}
		return reason
	default:
		return "other"
	}
}

func writePrometheus(w io.Writer, counters map[counterKey]uint64, invalidLines uint64) {
	fmt.Fprintln(w, "# HELP tinyidp_device_audit_invalid_lines_total Invalid or oversized audit records skipped by the device audit metrics exporter.")
	fmt.Fprintln(w, "# TYPE tinyidp_device_audit_invalid_lines_total counter")
	fmt.Fprintf(w, "tinyidp_device_audit_invalid_lines_total %d\n", invalidLines)

	keys := make([]counterKey, 0, len(counters))
	for key := range counters {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return metricLine(keys[i]) < metricLine(keys[j])
	})
	for _, key := range keys {
		fmt.Fprintf(w, "%s %d\n", metricLine(key), counters[key])
	}
}

func metricLine(key counterKey) string {
	labels := []string{fmt.Sprintf("stage=%q", key.Stage), fmt.Sprintf("outcome=%q", key.Outcome)}
	if key.Reason != "" {
		labels = append(labels, fmt.Sprintf("reason=%q", key.Reason))
	}
	return key.Name + "{" + strings.Join(labels, ",") + "}"
}
