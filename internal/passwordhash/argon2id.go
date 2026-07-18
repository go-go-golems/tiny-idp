package passwordhash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	AlgorithmArgon2id = "argon2id-v1"
	argon2Version     = 19
)

var (
	ErrInvalidHash      = errors.New("invalid password hash")
	ErrPasswordMismatch = errors.New("password mismatch")
)

type Params struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultParams() Params {
	return Params{MemoryKiB: 64 * 1024, Iterations: 3, Parallelism: 2, SaltLength: 16, KeyLength: 32}
}

func TestParams() Params {
	return Params{MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLength: 8, KeyLength: 16}
}

type Hasher struct {
	Params Params
	Rand   io.Reader
}

func New(params Params) Hasher {
	if params.MemoryKiB == 0 {
		params = DefaultParams()
	}
	return Hasher{Params: params, Rand: rand.Reader}
}

func (h Hasher) HashPassword(password []byte) ([]byte, error) {
	params := h.Params
	if params.MemoryKiB == 0 {
		params = DefaultParams()
	}
	if params.SaltLength == 0 || params.KeyLength == 0 || params.Iterations == 0 || params.Parallelism == 0 {
		return nil, fmt.Errorf("invalid argon2id params")
	}
	r := h.Rand
	if r == nil {
		r = rand.Reader
	}
	salt := make([]byte, params.SaltLength)
	if _, err := io.ReadFull(r, salt); err != nil {
		return nil, err
	}
	key := argon2.IDKey(password, salt, params.Iterations, params.MemoryKiB, params.Parallelism, params.KeyLength)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version,
		params.MemoryKiB,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return []byte(encoded), nil
}

func (h Hasher) VerifyPassword(password, encoded []byte) (bool, error) {
	parsed, err := Parse(encoded)
	if err != nil {
		return false, err
	}
	key := argon2.IDKey(password, parsed.Salt, parsed.Params.Iterations, parsed.Params.MemoryKiB, parsed.Params.Parallelism, parsed.Params.KeyLength)
	if subtle.ConstantTimeCompare(key, parsed.Key) != 1 {
		return false, ErrPasswordMismatch
	}
	want := h.Params
	if want.MemoryKiB == 0 {
		want = DefaultParams()
	}
	return parsed.Params != want, nil
}

type ParsedHash struct {
	Params Params
	Salt   []byte
	Key    []byte
}

func Parse(encoded []byte) (ParsedHash, error) {
	parts := strings.Split(string(encoded), "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return ParsedHash{}, ErrInvalidHash
	}
	if parts[2] != "v=19" {
		return ParsedHash{}, ErrInvalidHash
	}
	params, err := parseParams(parts[3])
	if err != nil {
		return ParsedHash{}, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return ParsedHash{}, ErrInvalidHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return ParsedHash{}, ErrInvalidHash
	}
	saltLength, ok := checkedLength(uint64(len(salt)))
	if !ok {
		return ParsedHash{}, ErrInvalidHash
	}
	keyLength, ok := checkedLength(uint64(len(key)))
	if !ok {
		return ParsedHash{}, ErrInvalidHash
	}
	params.SaltLength = saltLength
	params.KeyLength = keyLength
	return ParsedHash{Params: params, Salt: salt, Key: key}, nil
}

func checkedLength(length uint64) (uint32, bool) {
	if length == 0 || length > math.MaxUint32 {
		return 0, false
	}
	return uint32(length), true
}

func parseParams(raw string) (Params, error) {
	out := Params{}
	for _, part := range strings.Split(raw, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return Params{}, ErrInvalidHash
		}
		value, err := strconv.ParseUint(kv[1], 10, 32)
		if err != nil {
			return Params{}, ErrInvalidHash
		}
		switch kv[0] {
		case "m":
			out.MemoryKiB = uint32(value)
		case "t":
			out.Iterations = uint32(value)
		case "p":
			if value > 255 {
				return Params{}, ErrInvalidHash
			}
			out.Parallelism = uint8(value)
		default:
			return Params{}, ErrInvalidHash
		}
	}
	if out.MemoryKiB == 0 || out.Iterations == 0 || out.Parallelism == 0 {
		return Params{}, ErrInvalidHash
	}
	return out, nil
}
