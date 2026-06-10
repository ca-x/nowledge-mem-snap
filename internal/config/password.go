package config

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(encoded, password string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("unsupported password hash")
	}
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false, fmt.Errorf("invalid password hash params")
	}
	memory, err := parseParam(params[0], "m")
	if err != nil {
		return false, err
	}
	timeCost, err := parseParam(params[1], "t")
	if err != nil {
		return false, err
	}
	threads, err := parseParam(params[2], "p")
	if err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, uint32(timeCost), uint32(memory), uint8(threads), uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func parseParam(raw, name string) (int, error) {
	prefix := name + "="
	if !strings.HasPrefix(raw, prefix) {
		return 0, fmt.Errorf("missing password hash param %s", name)
	}
	value, err := strconv.Atoi(strings.TrimPrefix(raw, prefix))
	if err != nil {
		return 0, err
	}
	return value, nil
}
