package security

import (
	"encoding/json"
	"regexp"
	"strings"
)

const RedactedValue = "[REDACTED]"

var sensitiveKeyParts = []string{
	"password",
	"passwd",
	"token",
	"secret",
	"key",
	"credential",
	"authorization",
	"cookie",
	"set-cookie",
}

var sensitiveAssignmentPattern = regexp.MustCompile(`(?i)\b(password|passwd|token|secret|key|credential|authorization|cookie|set-cookie)\b\s*[:=]\s*("[^"]*"|'[^']*'|[^\s,}]+)`)

func RedactJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}

	var value interface{}
	if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
		return RedactText(raw)
	}

	redacted := redactValue(value)
	bytes, err := json.Marshal(redacted)
	if err != nil {
		return RedactText(raw)
	}
	return string(bytes)
}

func RedactText(raw string) string {
	if raw == "" {
		return raw
	}
	return sensitiveAssignmentPattern.ReplaceAllString(raw, `${1}: `+RedactedValue)
}

func redactValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, nested := range typed {
			if IsSensitiveKey(key) {
				typed[key] = RedactedValue
				continue
			}
			typed[key] = redactValue(nested)
		}
		return typed
	case []interface{}:
		for index, nested := range typed {
			typed[index] = redactValue(nested)
		}
		return typed
	default:
		return typed
	}
}

func IsSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}
	for _, part := range sensitiveKeyParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}
