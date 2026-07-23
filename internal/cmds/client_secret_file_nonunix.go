//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package cmds

import "github.com/pkg/errors"

func readClientSecretFile(string) ([]byte, error) {
	return nil, errors.New("client secret file input is supported only on POSIX platforms")
}
