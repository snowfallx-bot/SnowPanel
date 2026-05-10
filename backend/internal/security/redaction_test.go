package security

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactJSONRedactsNestedSensitiveKeys(t *testing.T) {
	raw := `{
		"username":"admin",
		"password":"plain",
		"nested":{"api_key":"abc","ok":"visible"},
		"headers":[{"Authorization":"Bearer token-value"},{"cookie":"session=abc"}]
	}`

	redacted := RedactJSON(raw)
	if strings.Contains(redacted, "plain") ||
		strings.Contains(redacted, "abc") ||
		strings.Contains(redacted, "token-value") ||
		strings.Contains(redacted, "session=abc") {
		t.Fatalf("redacted JSON leaked sensitive value: %s", redacted)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(redacted), &decoded); err != nil {
		t.Fatalf("expected redacted output to remain valid JSON: %v", err)
	}
	if decoded["username"] != "admin" {
		t.Fatalf("expected non-sensitive field to remain visible: %s", redacted)
	}
}

func TestRedactTextRedactsAssignments(t *testing.T) {
	raw := `connect failed password=plain token:"abc" service=nginx`

	redacted := RedactText(raw)
	if strings.Contains(redacted, "plain") || strings.Contains(redacted, "abc") {
		t.Fatalf("redacted text leaked sensitive value: %s", redacted)
	}
	if !strings.Contains(redacted, "service=nginx") {
		t.Fatalf("expected non-sensitive text to remain visible: %s", redacted)
	}
}

func TestIsSensitiveKey(t *testing.T) {
	for _, key := range []string{"password", "passwd", "refresh_token", "client_secret", "api_key", "credential", "authorization", "set-cookie"} {
		if !IsSensitiveKey(key) {
			t.Fatalf("expected %s to be sensitive", key)
		}
	}
	if IsSensitiveKey("service_name") {
		t.Fatal("did not expect service_name to be sensitive")
	}
}
