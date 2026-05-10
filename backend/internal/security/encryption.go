package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	EncryptionPayloadVersion = 1
	EncryptionAlgorithm      = "AES-256-GCM"
	encryptionKeyBytes       = 32
	encryptionNonceBytes     = 12
)

type Encryptor struct {
	keyID string
	aead  cipher.AEAD
}

type EncryptedPayload struct {
	Version    int    `json:"version"`
	Algorithm  string `json:"algorithm"`
	KeyID      string `json:"key_id,omitempty"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func NewEncryptor(keyMaterial string, keyID string) (*Encryptor, error) {
	key, err := DecodeEncryptionKey(keyMaterial)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("initialize encryption cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialize encryption mode: %w", err)
	}

	return &Encryptor{
		keyID: strings.TrimSpace(keyID),
		aead:  aead,
	}, nil
}

func DecodeEncryptionKey(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("SNOWPANEL_ENCRYPTION_KEY cannot be empty when encryption is required")
	}

	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		decoded, err := encoding.DecodeString(trimmed)
		if err == nil && len(decoded) == encryptionKeyBytes {
			return decoded, nil
		}
	}

	if len([]byte(trimmed)) == encryptionKeyBytes {
		return []byte(trimmed), nil
	}

	return nil, errors.New("SNOWPANEL_ENCRYPTION_KEY must decode to 32 bytes for AES-256-GCM")
}

func GenerateEncryptionKey() (string, error) {
	key := make([]byte, encryptionKeyBytes)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("generate encryption key: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(key), nil
}

func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if e == nil || e.aead == nil {
		return "", errors.New("encryptor is not initialized")
	}

	nonce := make([]byte, encryptionNonceBytes)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate encryption nonce: %w", err)
	}

	ciphertext := e.aead.Seal(nil, nonce, []byte(plaintext), nil)
	payload := EncryptedPayload{
		Version:    EncryptionPayloadVersion,
		Algorithm:  EncryptionAlgorithm,
		KeyID:      e.keyID,
		Nonce:      base64.RawStdEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawStdEncoding.EncodeToString(ciphertext),
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal encrypted payload: %w", err)
	}
	return string(bytes), nil
}

func (e *Encryptor) Decrypt(rawPayload string) (string, error) {
	if e == nil || e.aead == nil {
		return "", errors.New("encryptor is not initialized")
	}

	var payload EncryptedPayload
	if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
		return "", errors.New("encrypted payload is invalid")
	}
	if payload.Version != EncryptionPayloadVersion {
		return "", errors.New("encrypted payload version is unsupported")
	}
	if payload.Algorithm != EncryptionAlgorithm {
		return "", errors.New("encrypted payload algorithm is unsupported")
	}

	nonce, err := base64.RawStdEncoding.DecodeString(payload.Nonce)
	if err != nil || len(nonce) != encryptionNonceBytes {
		return "", errors.New("encrypted payload nonce is invalid")
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return "", errors.New("encrypted payload ciphertext is invalid")
	}

	plaintext, err := e.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("encrypted payload authentication failed")
	}
	return string(plaintext), nil
}
