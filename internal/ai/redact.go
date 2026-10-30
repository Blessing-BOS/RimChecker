package ai

import (
	"regexp"
	"strings"
)

var redactions = []*regexp.Regexp{
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`),
	regexp.MustCompile(`(?i)\b(?:api[_-]?key|access[_-]?token|refresh[_-]?token|authorization|password|passwd|secret|token)\b["']?\s*[:=]\s*["']?(?:Bearer\s+)?[^\s,;"'}]+`),
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{8,}\b`),
	regexp.MustCompile(`(?i)[A-Z]:[\\/]+(?:Users|Documents and Settings)[\\/]+[^\\/\r\n]+`),
	regexp.MustCompile(`/(?:home|Users)/[^/\r\n]+`),
	regexp.MustCompile(`(?i)\b(?:steam(?:id|_id)?|user[_-]?id|account(?:[_-]?id)?|username|email)\b["']?\s*[:=]\s*["']?[^\s,;"'}]+`),
	regexp.MustCompile(`\b7656119[0-9]{10}\b`),
	regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`),
}

// Redact is best-effort. Known secrets are removed before pattern matching and
// before any truncation so a partial credential is not exposed by clipping.
func Redact(text string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "[REDACTED]")
		}
	}
	for _, pattern := range redactions {
		text = pattern.ReplaceAllString(text, "[REDACTED]")
	}
	return text
}
