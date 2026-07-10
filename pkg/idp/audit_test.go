package idp_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/manuel/tinyidp/pkg/idp"
)

func TestFileAuditSinkDurabilityAndFailureHealth(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit", "events.jsonl")
	sink, err := idp.NewFileAuditSink(path)
	if err != nil {
		t.Fatal(err)
	}
	event := idp.New("test.accepted")
	if err := sink.Emit(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	health := sink.AuditHealth(context.Background())
	if !health.Ready || health.Delivered != 1 || health.Dropped != 0 || health.Policy != "synchronous-fsync" {
		t.Fatalf("unexpected audit health: %#v", health)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var stored idp.Event
	if err := json.NewDecoder(bufio.NewReader(file)).Decode(&stored); err != nil {
		t.Fatal(err)
	}
	if stored.Name != event.Name {
		t.Fatalf("stored event = %q", stored.Name)
	}
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
	if err := sink.Emit(context.Background(), idp.New("after.close")); err == nil {
		t.Fatal("expected closed sink failure")
	}
	health = sink.AuditHealth(context.Background())
	if health.Ready || health.Failed != 1 || health.Reason != "audit_closed" {
		t.Fatalf("unexpected failed audit health: %#v", health)
	}
}

func TestFileAuditSinkRejectsCanceledContextWithoutWriting(t *testing.T) {
	sink, err := idp.NewFileAuditSink(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sink.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sink.Emit(ctx, idp.New("canceled")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Emit error = %v", err)
	}
	if health := sink.AuditHealth(context.Background()); health.Delivered != 0 {
		t.Fatalf("unexpected delivery: %#v", health)
	}
}
