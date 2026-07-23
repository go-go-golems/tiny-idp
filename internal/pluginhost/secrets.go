package pluginhost

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

type FileSecretResolver struct {
	MaxBytes int64
}

func (r FileSecretResolver) Read(ctx context.Context, path string, minimumBytes int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(path) == "" || minimumBytes <= 0 {
		return nil, errors.New("secret path and positive minimum size are required")
	}
	limit := r.MaxBytes
	if limit == 0 {
		limit = 64 << 10
	}
	if limit < int64(minimumBytes) {
		return nil, errors.New("secret response bound is below the required minimum")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat secret file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > limit {
		return nil, errors.New("secret file must be bounded, regular, and owner-only (0600 or 0400)")
	}
	value, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read secret file: %w", err)
	}
	value = bytes.TrimSuffix(value, []byte("\n"))
	if len(value) < minimumBytes {
		zeroSecretBytes(value)
		return nil, fmt.Errorf("secret file must contain at least %d bytes", minimumBytes)
	}
	return value, nil
}

func zeroSecretBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
