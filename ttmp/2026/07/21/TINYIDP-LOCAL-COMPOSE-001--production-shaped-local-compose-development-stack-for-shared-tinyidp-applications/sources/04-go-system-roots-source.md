<!-- Source: https://go.dev/src/crypto/x509/root_unix.go
Captured with defuddle on 2026-07-21 for TINYIDP-LOCAL-COMPOSE-001. -->

## Source file src/crypto/x509/root\_unix.go

```
1  // Copyright 2011 The Go Authors. All rights reserved.
  2  // Use of this source code is governed by a BSD-style
  3  // license that can be found in the LICENSE file.
  4
  5  //go:build aix || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris || wasip1
  6
  7  package x509
  8
  9  import (
 10      "io/fs"
 11      "os"
 12      "path/filepath"
 13      "strings"
 14  )
 15
 16  const (
 17      // certFileEnv is the environment variable which identifies where to locate
 18      // the SSL certificate file. If set this overrides the system default.
 19      certFileEnv = "SSL_CERT_FILE"
 20
 21      // certDirEnv is the environment variable which identifies which directory
 22      // to check for SSL certificate files. If set this overrides the system default.
 23      // It is a colon separated list of directories.
 24      // See https://www.openssl.org/docs/man1.0.2/man1/c_rehash.html.
 25      certDirEnv = "SSL_CERT_DIR"
 26  )
 27
 28  func (c *Certificate) systemVerify(opts *VerifyOptions) (chains [][]*Certificate, err error) {
 29      return nil, nil
 30  }
 31
 32  func loadSystemRoots() (*CertPool, error) {
 33      roots := NewCertPool()
 34
 35      files := certFiles
 36      if f := os.Getenv(certFileEnv); f != "" {
 37          files = []string{f}
 38      }
 39
 40      var firstErr error
 41      for _, file := range files {
 42          data, err := os.ReadFile(file)
 43          if err == nil {
 44              roots.AppendCertsFromPEM(data)
 45              break
 46          }
 47          if firstErr == nil && !os.IsNotExist(err) {
 48              firstErr = err
 49          }
 50      }
 51
 52      dirs := certDirectories
 53      if d := os.Getenv(certDirEnv); d != "" {
 54          // OpenSSL and BoringSSL both use ":" as the SSL_CERT_DIR separator.
 55          // See:
 56          //  * https://golang.org/issue/35325
 57          //  * https://www.openssl.org/docs/man1.0.2/man1/c_rehash.html
 58          dirs = strings.Split(d, ":")
 59      }
 60
 61      for _, directory := range dirs {
 62          fis, err := readUniqueDirectoryEntries(directory)
 63          if err != nil {
 64              if firstErr == nil && !os.IsNotExist(err) {
 65                  firstErr = err
 66              }
 67              continue
 68          }
 69          for _, fi := range fis {
 70              data, err := os.ReadFile(directory + "/" + fi.Name())
 71              if err == nil {
 72                  roots.AppendCertsFromPEM(data)
 73              }
 74          }
 75      }
 76
 77      if roots.len() > 0 || firstErr == nil {
 78          return roots, nil
 79      }
 80
 81      return nil, firstErr
 82  }
 83
 84  // readUniqueDirectoryEntries is like os.ReadDir but omits
 85  // symlinks that point within the directory.
 86  func readUniqueDirectoryEntries(dir string) ([]fs.DirEntry, error) {
 87      files, err := os.ReadDir(dir)
 88      if err != nil {
 89          return nil, err
 90      }
 91      uniq := files[:0]
 92      for _, f := range files {
 93          if !isSameDirSymlink(f, dir) {
 94              uniq = append(uniq, f)
 95          }
 96      }
 97      return uniq, nil
 98  }
 99
100  // isSameDirSymlink reports whether fi in dir is a symlink with a
101  // target not containing a slash.
102  func isSameDirSymlink(f fs.DirEntry, dir string) bool {
103      if f.Type()&fs.ModeSymlink == 0 {
104          return false
105      }
106      target, err := os.Readlink(filepath.Join(dir, f.Name()))
107      return err == nil && !strings.Contains(target, "/")
108  }
109
```

[View as plain text](https://go.dev/src/crypto/x509/root_unix.go?m=text)
