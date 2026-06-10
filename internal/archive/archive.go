package archive

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"golang.org/x/crypto/scrypt"
)

const (
	PlainExtension     = ".zip"
	EncryptedExtension = ".zip.aes.json"
	envelopeFormat     = "nowledge-mem-snap/aes-gcm/v1"
)

type Metadata struct {
	TaskKey   string    `json:"task_key"`
	TaskName  string    `json:"task_name"`
	CreatedAt time.Time `json:"created_at"`
	SizeBytes int       `json:"size_bytes"`
	ItemCount int       `json:"item_count"`
}

type Artifact struct {
	Name        string
	ContentType string
	Data        []byte
	Encrypted   bool
	SizeBytes   int64
}

type encryptedEnvelope struct {
	Format     string    `json:"format"`
	KDF        string    `json:"kdf"`
	N          int       `json:"n"`
	R          int       `json:"r"`
	P          int       `json:"p"`
	Salt       string    `json:"salt"`
	Nonce      string    `json:"nonce"`
	CreatedAt  time.Time `json:"created_at"`
	Metadata   Metadata  `json:"metadata"`
	Ciphertext string    `json:"ciphertext"`
}

type BuildOptions struct {
	Prefix             string
	TaskKey            string
	TaskName           string
	ItemCount          int
	CreatedAt          time.Time
	Encrypt            bool
	EncryptionPassword string
}

func Build(zipData []byte, opts BuildOptions) (Artifact, error) {
	if opts.CreatedAt.IsZero() {
		opts.CreatedAt = time.Now().UTC()
	}
	name := ObjectName(opts.Prefix, opts.TaskKey, opts.CreatedAt, PlainExtension)
	artifact := Artifact{
		Name:        name,
		ContentType: "application/zip",
		Data:        zipData,
		Encrypted:   false,
		SizeBytes:   int64(len(zipData)),
	}
	if !opts.Encrypt {
		return artifact, nil
	}
	if strings.TrimSpace(opts.EncryptionPassword) == "" {
		return Artifact{}, fmt.Errorf("encryption is enabled but password is empty")
	}
	metadata := Metadata{
		TaskKey:   opts.TaskKey,
		TaskName:  opts.TaskName,
		CreatedAt: opts.CreatedAt,
		SizeBytes: len(zipData),
		ItemCount: opts.ItemCount,
	}
	encrypted, err := Encrypt(zipData, strings.TrimSpace(opts.EncryptionPassword), metadata)
	if err != nil {
		return Artifact{}, err
	}
	artifact.Name = ObjectName(opts.Prefix, opts.TaskKey, opts.CreatedAt, EncryptedExtension)
	artifact.ContentType = "application/json"
	artifact.Data = encrypted
	artifact.Encrypted = true
	artifact.SizeBytes = int64(len(encrypted))
	return artifact, nil
}

func Encrypt(plain []byte, password string, metadata Metadata) ([]byte, error) {
	salt := make([]byte, 16)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	const n = 32768
	const r = 8
	const p = 1
	key, err := scrypt.Key([]byte(password), salt, n, r, p, 32)
	if err != nil {
		return nil, fmt.Errorf("derive encryption key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plain, aad(metadata))
	env := encryptedEnvelope{
		Format:     envelopeFormat,
		KDF:        "scrypt",
		N:          n,
		R:          r,
		P:          p,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		CreatedAt:  time.Now().UTC(),
		Metadata:   metadata,
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(env); err != nil {
		return nil, fmt.Errorf("encode encrypted envelope: %w", err)
	}
	return buf.Bytes(), nil
}

func ObjectName(prefix, taskKey string, at time.Time, extension string) string {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	timestamp := at.UTC().Format("20060102T150405Z")
	date := at.UTC().Format("2006-01-02")
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "nowledge-mem/{task}/{timestamp}"
	}
	replacer := strings.NewReplacer(
		"{task}", cleanSegment(taskKey),
		"{timestamp}", timestamp,
		"{date}", date,
	)
	name := replacer.Replace(prefix)
	if strings.HasSuffix(name, "/") {
		name += "nowledge-mem-" + cleanSegment(taskKey) + "-" + timestamp
	}
	if !strings.HasSuffix(name, PlainExtension) && !strings.HasSuffix(name, EncryptedExtension) {
		name += extension
	}
	name = cleanObjectPath(name)
	if strings.HasPrefix(name, "/") {
		name = strings.TrimLeft(name, "/")
	}
	return name
}

func RetentionDirectory(prefix, taskKey string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "nowledge-mem/{task}/{timestamp}"
	}
	prefix = strings.ReplaceAll(prefix, "{task}", cleanSegment(taskKey))
	firstDynamic := -1
	for _, token := range []string{"{timestamp}", "{date}"} {
		if idx := strings.Index(prefix, token); idx >= 0 && (firstDynamic == -1 || idx < firstDynamic) {
			firstDynamic = idx
		}
	}
	if firstDynamic >= 0 {
		before := prefix[:firstDynamic]
		if strings.HasSuffix(before, "/") || strings.HasSuffix(before, "\\") {
			return retentionScope(before)
		}
		return retentionScope(path.Dir(before))
	}
	if strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, "\\") {
		return retentionScope(prefix)
	}
	return retentionScope(path.Dir(prefix))
}

func retentionScope(raw string) string {
	scope := cleanObjectPath(raw)
	if scope == "" {
		return "."
	}
	return scope
}

func aad(metadata Metadata) []byte {
	data, _ := json.Marshal(struct {
		Format   string   `json:"format"`
		Metadata Metadata `json:"metadata"`
	}{Format: envelopeFormat, Metadata: metadata})
	return data
}

func cleanObjectPath(raw string) string {
	parts := strings.Split(strings.ReplaceAll(raw, "\\", "/"), "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" || part == "." || part == ".." {
			continue
		}
		part = cleanSegment(part)
		if part == "" || part == "." || part == ".." {
			continue
		}
		cleaned = append(cleaned, part)
	}
	return path.Join(cleaned...)
}

func cleanSegment(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.', r == '{', r == '}':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), ".-_/")
}
