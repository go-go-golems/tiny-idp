//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package cmds

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

const maxClientSecretFileBytes = 4096

func readClientSecretFile(path string) ([]byte, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		if err == unix.ELOOP {
			return nil, errors.New("client secret file must be regular and not a symlink")
		}
		return nil, errors.Wrap(err, "open client secret file")
	}
	file := os.NewFile(uintptr(fd), path)
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "inspect client secret file")
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("client secret file must be regular and not a symlink")
	}
	if info.Size() > maxClientSecretFileBytes {
		return nil, errors.New("client secret file is too large")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxClientSecretFileBytes+1))
	if err != nil {
		return nil, errors.Wrap(err, "read client secret file")
	}
	if len(data) > maxClientSecretFileBytes {
		return nil, errors.New("client secret file is too large")
	}
	return data, nil
}
