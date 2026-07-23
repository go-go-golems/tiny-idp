//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package cmds

import (
	"strings"
	"testing"
)

func TestResolveClientSecretFileFailsClosedOnNonPOSIX(t *testing.T) {
	_, err := resolveClientSecret("", "operator-managed-secret", false)
	if err == nil || !strings.Contains(err.Error(), "supported only on POSIX") {
		t.Fatalf("non-POSIX secret-file error = %v", err)
	}
}
