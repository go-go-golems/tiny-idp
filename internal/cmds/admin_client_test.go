//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package cmds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestResolveClientSecretFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("operator-managed-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secret, err := resolveClientSecret("", path, false)
	if err != nil || secret != "operator-managed-secret" {
		t.Fatalf("secret length=%d err=%v", len(secret), err)
	}
	if _, err := resolveClientSecret("inline", path, false); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("mutual exclusion error = %v", err)
	}
	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveClientSecret("", empty, false); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty error = %v", err)
	}
	symlink := filepath.Join(t.TempDir(), "symlink")
	if err := os.Symlink(path, symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveClientSecret("", symlink, false); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink error = %v", err)
	}
	tooLongSecret := filepath.Join(t.TempDir(), "too-long-secret")
	if err := os.WriteFile(tooLongSecret, make([]byte, 73), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveClientSecret("", tooLongSecret, false); err == nil || !strings.Contains(err.Error(), "72 bytes") {
		t.Fatalf("long secret error = %v", err)
	}
	large := filepath.Join(t.TempDir(), "large")
	if err := os.WriteFile(large, make([]byte, 4097), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveClientSecret("", large, false); err == nil || !strings.Contains(err.Error(), "large") {
		t.Fatalf("large file error = %v", err)
	}
}

func TestResolveClientSecretFileRejectsFIFOWithoutBlocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.fifo")
	if err := unix.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := resolveClientSecret("", path, false)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "regular") {
			t.Fatalf("FIFO error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO secret file blocked instead of being rejected")
	}
}
