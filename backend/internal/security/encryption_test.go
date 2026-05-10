package security

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	encryptor, err := NewEncryptor(key, "primary")
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	encrypted, err := encryptor.Encrypt("secret-setting-value")
	if err != nil {
		t.Fatalf("expected encrypt success, got %v", err)
	}
	if strings.Contains(encrypted, "secret-setting-value") {
		t.Fatalf("encrypted payload leaked plaintext: %s", encrypted)
	}

	decrypted, err := encryptor.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("expected decrypt success, got %v", err)
	}
	if decrypted != "secret-setting-value" {
		t.Fatalf("unexpected decrypted value: %s", decrypted)
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	keyA, _ := GenerateEncryptionKey()
	keyB, _ := GenerateEncryptionKey()
	encryptorA, err := NewEncryptor(keyA, "primary")
	if err != nil {
		t.Fatalf("failed to create first encryptor: %v", err)
	}
	encryptorB, err := NewEncryptor(keyB, "secondary")
	if err != nil {
		t.Fatalf("failed to create second encryptor: %v", err)
	}

	encrypted, err := encryptorA.Encrypt("secret-setting-value")
	if err != nil {
		t.Fatalf("expected encrypt success, got %v", err)
	}

	if _, err := encryptorB.Decrypt(encrypted); err == nil {
		t.Fatal("expected decrypt with wrong key to fail")
	}
}

func TestNewEncryptorRejectsMissingOrInvalidKey(t *testing.T) {
	for _, key := range []string{"", "short", "not-base64-and-not-32-bytes"} {
		if _, err := NewEncryptor(key, "primary"); err == nil {
			t.Fatalf("expected key %q to be rejected", key)
		}
	}
}

func TestEncryptedPayloadShape(t *testing.T) {
	key, _ := GenerateEncryptionKey()
	encryptor, err := NewEncryptor(key, "primary")
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	encrypted, err := encryptor.Encrypt("secret")
	if err != nil {
		t.Fatalf("expected encrypt success, got %v", err)
	}

	var payload EncryptedPayload
	if err := json.Unmarshal([]byte(encrypted), &payload); err != nil {
		t.Fatalf("expected encrypted payload JSON: %v", err)
	}
	if payload.Version != EncryptionPayloadVersion || payload.Algorithm != EncryptionAlgorithm {
		t.Fatalf("unexpected payload metadata: %+v", payload)
	}
	if payload.Nonce == "" || payload.Ciphertext == "" {
		t.Fatalf("expected nonce and ciphertext: %+v", payload)
	}
}
