package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/manuel/tinyidp/pkg/idp"
)

type event struct {
	Type         string                 `json:"type"`
	Phase        string                 `json:"phase"`
	Name         string                 `json:"name"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	Status       int                    `json:"status"`
	Bytes        int                    `json:"bytes"`
	DurationUS   int64                  `json:"duration_us"`
	Metrics      map[string]float64     `json:"metrics"`
	DB           *dbSnapshot            `json:"db"`
	AuditCount   int                    `json:"audit_count"`
	Error        string                 `json:"error"`
	PasswordWork *idp.PasswordWorkStats `json:"password_work"`
}

type dbSnapshot struct {
	OpenConnections int   `json:"open_connections"`
	InUse           int   `json:"in_use"`
	Idle            int   `json:"idle"`
	WaitCount       int64 `json:"wait_count"`
	WaitDurationUS  int64 `json:"wait_duration_us"`
	MaxIdleClosed   int64 `json:"max_idle_closed"`
	MaxLifetimeDone int64 `json:"max_lifetime_closed"`
}

type routeStats struct {
	durations []int64
	bytes     int
	statuses  map[int]int
	errors    int
}

func main() {
	input := flag.String("input", "", "runtime NDJSON input")
	output := flag.String("output", "-", "Markdown output or - for stdout")
	logLevel := flag.String("log-level", "info", "zerolog level")
	ticket := flag.String("ticket", "TINYIDP-PROD-REVIEW-001", "docmgr ticket identifier for output frontmatter")
	flag.Parse()
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --log-level: %v\n", err)
		os.Exit(2)
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	if *input == "" {
		log.Fatal().Msg("--input is required")
	}
	if err := run(*input, *output, *ticket); err != nil {
		log.Fatal().Err(err).Msg("runtime analysis failed")
	}
}

func run(input, output, ticket string) error {
	file, err := os.Open(input)
	if err != nil {
		return err
	}
	defer file.Close()

	routes := map[string]*routeStats{}
	runtimeSamples := map[string]map[string]float64{}
	databaseSamples := map[string]*dbSnapshot{}
	auditCount := 0
	var passwordWork *idp.PasswordWorkStats
	line := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line++
		var value event
		if err := json.Unmarshal(scanner.Bytes(), &value); err != nil {
			return fmt.Errorf("line %d: %w", line, err)
		}
		switch value.Type {
		case "request":
			key := value.Method + " " + value.Path + " (" + value.Name + ")"
			stats := routes[key]
			if stats == nil {
				stats = &routeStats{statuses: map[int]int{}}
				routes[key] = stats
			}
			stats.durations = append(stats.durations, value.DurationUS)
			stats.bytes += value.Bytes
			stats.statuses[value.Status]++
			if value.Error != "" {
				stats.errors++
			}
		case "runtime":
			runtimeSamples[value.Phase] = value.Metrics
		case "database":
			databaseSamples[value.Phase] = value.DB
		case "summary":
			auditCount = value.AuditCount
			passwordWork = value.PasswordWork
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	writer, closeWriter, err := markdownWriter(output)
	if err != nil {
		return err
	}
	defer closeWriter()
	return writeSummary(writer, input, ticket, routes, runtimeSamples, databaseSamples, auditCount, passwordWork)
}

func writeSummary(w io.Writer, input, ticket string, routes map[string]*routeStats, runtimeSamples map[string]map[string]float64, databaseSamples map[string]*dbSnapshot, auditCount int, passwordWork *idp.PasswordWorkStats) error {
	keys := make([]string, 0, len(routes))
	for key := range routes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	requestCount := 0
	for _, stats := range routes {
		requestCount += len(stats.durations)
	}
	_, err := fmt.Fprintf(w, `---
Title: tiny-idp runtime probe summary
Ticket: %s
Status: active
Topics:
    - oidc
    - go
    - testing
    - auth
    - research
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Measured strict production-mode login, token, refresh, read-path, Go runtime, SQLite pool, and audit behavior."
LastUpdated: %s
WhatFor: "Providing a bounded runtime regression baseline for the production review."
WhenToUse: "Use when changing authentication, Fosite, SQLite, runtime limits, or observability."
---

# tiny-idp runtime probe summary

Source: `+"`%s`"+`

Requests observed: **%d**. Audit events emitted: **%d**.

## HTTP observations

| Operation | Count | Statuses | p50 | p95 | p99 | Mean bytes | Errors |
|---|---:|---|---:|---:|---:|---:|---:|
`, ticket, time.Now().UTC().Format(time.RFC3339), input, requestCount, auditCount)
	if err != nil {
		return err
	}
	for _, key := range keys {
		stats := routes[key]
		sort.Slice(stats.durations, func(i, j int) bool { return stats.durations[i] < stats.durations[j] })
		statuses := make([]string, 0, len(stats.statuses))
		statusCodes := make([]int, 0, len(stats.statuses))
		for code := range stats.statuses {
			statusCodes = append(statusCodes, code)
		}
		sort.Ints(statusCodes)
		for _, code := range statusCodes {
			statuses = append(statuses, fmt.Sprintf("%d×%d", code, stats.statuses[code]))
		}
		meanBytes := 0
		if len(stats.durations) > 0 {
			meanBytes = stats.bytes / len(stats.durations)
		}
		if _, err := fmt.Fprintf(w, "| `%s` | %d | %s | %s | %s | %s | %d | %d |\n", key, len(stats.durations), strings.Join(statuses, ", "), formatDuration(quantile(stats.durations, 0.50)), formatDuration(quantile(stats.durations, 0.95)), formatDuration(quantile(stats.durations, 0.99)), meanBytes, stats.errors); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, "\n## Go runtime deltas\n\n| Metric | Before | After | Delta |\n|---|---:|---:|---:|"); err != nil {
		return err
	}
	before := runtimeSamples["before"]
	after := runtimeSamples["after"]
	metricNames := make([]string, 0, len(after))
	for name := range after {
		metricNames = append(metricNames, name)
	}
	sort.Strings(metricNames)
	for _, name := range metricNames {
		if _, err := fmt.Fprintf(w, "| `%s` | %.3f | %.3f | %+.3f |\n", name, before[name], after[name], after[name]-before[name]); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, "\n## SQLite pool snapshots\n\n| Phase | Open | In use | Idle | Wait count | Wait time | Max-idle closed | Max-lifetime closed |\n|---|---:|---:|---:|---:|---:|---:|---:|"); err != nil {
		return err
	}
	for _, phase := range []string{"before", "after"} {
		stats := databaseSamples[phase]
		if stats == nil {
			continue
		}
		if _, err := fmt.Fprintf(w, "| %s | %d | %d | %d | %d | %s | %d | %d |\n", phase, stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount, formatDuration(stats.WaitDurationUS), stats.MaxIdleClosed, stats.MaxLifetimeDone); err != nil {
			return err
		}
	}
	if passwordWork != nil {
		if _, err := fmt.Fprintf(w, `
## Password-work admission

| Capacity | Completed | Saturations | Rejected | Total wait | Total Argon2 duration |
|---:|---:|---:|---:|---:|---:|
| %d | %d | %d | %d | %s | %s |
`, passwordWork.Capacity, passwordWork.Completed, passwordWork.Saturations, passwordWork.Rejected, formatDuration(passwordWork.TotalWait/1_000), formatDuration(passwordWork.TotalDuration/1_000)); err != nil {
			return err
		}
	}

	_, err = fmt.Fprintln(w, "\n## Interpretation limits\n\nThis is a bounded, in-process probe using `httptest` and a temporary local SQLite database. It validates real strict-handler and persistence code paths, but it does not model reverse-proxy behavior, network TLS, multi-process SQLite contention, production disks, or sustained traffic. Treat the values as diagnostic evidence and regression baselines, not capacity numbers.")
	return err
}

func quantile(sortedValues []int64, q float64) int64 {
	if len(sortedValues) == 0 {
		return 0
	}
	index := int(float64(len(sortedValues)-1) * q)
	return sortedValues[index]
}

func formatDuration(microseconds int64) string {
	if microseconds < 1_000 {
		return fmt.Sprintf("%d µs", microseconds)
	}
	return fmt.Sprintf("%.2f ms", float64(microseconds)/1_000)
}

func markdownWriter(path string) (io.Writer, func(), error) {
	if path == "-" {
		return os.Stdout, func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return file, func() { _ = file.Close() }, nil
}
