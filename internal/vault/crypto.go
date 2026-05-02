package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	encryptedPrefix = "aes-gcm:"
	keySizeBytes    = 32
)

// KeyProvider loads the daemon-local vault encryption key.
type KeyProvider interface {
	Key() ([]byte, error)
}

type fileKeyProvider struct {
	path      string
	lookupEnv func(string) (string, bool)
}

// NewFileKeyProvider returns a non-interactive key provider backed by AGH_VAULT_KEY or a 0600 key file.
func NewFileKeyProvider(homeDir string, lookupEnv func(string) (string, bool)) KeyProvider {
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	return fileKeyProvider{
		path:      filepath.Join(strings.TrimSpace(homeDir), "vault.key"),
		lookupEnv: lookupEnv,
	}
}

func (p fileKeyProvider) Key() ([]byte, error) {
	if value, ok := p.lookupEnv("AGH_VAULT_KEY"); ok && strings.TrimSpace(value) != "" {
		key, err := decodeKey(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("vault: decode AGH_VAULT_KEY: %w", err)
		}
		return key, nil
	}
	if strings.TrimSpace(p.path) == "" {
		return nil, errors.New("vault: key path is required")
	}
	payload, err := os.ReadFile(p.path)
	if err == nil {
		key, decodeErr := decodeKey(strings.TrimSpace(string(payload)))
		if decodeErr != nil {
			return nil, fmt.Errorf("vault: decode key file %q: %w", p.path, decodeErr)
		}
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("vault: read key file %q: %w", p.path, err)
	}
	key := make([]byte, keySizeBytes)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("vault: generate key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(p.path), 0o700); err != nil {
		return nil, fmt.Errorf("vault: create key directory %q: %w", filepath.Dir(p.path), err)
	}
	encoded := base64.StdEncoding.EncodeToString(key)
	if err := os.WriteFile(p.path, []byte(encoded+"\n"), 0o600); err != nil {
		return nil, fmt.Errorf("vault: write key file %q: %w", p.path, err)
	}
	return key, nil
}

func decodeKey(value string) ([]byte, error) {
	if strings.TrimSpace(value) == "" {
		return nil, errors.New("key is required")
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil && len(decoded) == keySizeBytes {
		return decoded, nil
	}
	if decoded, err := hex.DecodeString(value); err == nil && len(decoded) == keySizeBytes {
		return decoded, nil
	}
	if len(value) == keySizeBytes {
		return []byte(value), nil
	}
	return nil, fmt.Errorf("key must be %d bytes as raw text, hex, or base64", keySizeBytes)
}

func encryptValue(key []byte, plaintext string) (string, error) {
	if len(key) != keySizeBytes {
		return "", fmt.Errorf("vault: key must be %d bytes", keySizeBytes)
	}
	if strings.TrimSpace(plaintext) == "" {
		return "", ErrMissingSecret
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("vault: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("vault: create gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("vault: generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	payload := make([]byte, 0, len(nonce)+len(ciphertext))
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)
	return encryptedPrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func decryptValue(key []byte, encrypted string) (string, error) {
	if len(key) != keySizeBytes {
		return "", fmt.Errorf("vault: key must be %d bytes", keySizeBytes)
	}
	if !strings.HasPrefix(encrypted, encryptedPrefix) {
		return "", errors.New("vault: encrypted value is missing aes-gcm prefix")
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encrypted, encryptedPrefix))
	if err != nil {
		return "", fmt.Errorf("vault: decode encrypted value: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("vault: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("vault: create gcm: %w", err)
	}
	if len(payload) <= gcm.NonceSize() {
		return "", errors.New("vault: encrypted value is truncated")
	}
	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("vault: decrypt value: %w", err)
	}
	return string(plaintext), nil
}
