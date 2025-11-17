package gitlab

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
)

// TokenRedactionPlaceholder is used to replace sensitive tokens in logs
const TokenRedactionPlaceholder = "[REDACTED]"

// RedactToken removes sensitive token information from a string
func RedactToken(s string) string {
	// GitLab PAT format: glpat-xxxxxxxxxxxxx
	gitlabPATPattern := regexp.MustCompile(`glpat-[a-zA-Z0-9_-]+`)
	s = gitlabPATPattern.ReplaceAllString(s, TokenRedactionPlaceholder)

	// GitLab CI token format: gitlab-ci-token
	gitlabCIPattern := regexp.MustCompile(`gitlab-ci-token:\s*[a-zA-Z0-9_-]+`)
	s = gitlabCIPattern.ReplaceAllString(s, "gitlab-ci-token: "+TokenRedactionPlaceholder)

	// Bearer tokens in Authorization headers
	bearerPattern := regexp.MustCompile(`Bearer\s+[a-zA-Z0-9_-]+`)
	s = bearerPattern.ReplaceAllString(s, "Bearer "+TokenRedactionPlaceholder)

	// OAuth2 tokens in URLs: oauth2:TOKEN@
	oauthURLPattern := regexp.MustCompile(`oauth2:[^@]+@`)
	s = oauthURLPattern.ReplaceAllString(s, "oauth2:"+TokenRedactionPlaceholder+"@")

	// Generic token pattern in URLs
	tokenURLPattern := regexp.MustCompile(`://[^:]+:[^@]+@`)
	s = tokenURLPattern.ReplaceAllString(s, "://"+TokenRedactionPlaceholder+":"+TokenRedactionPlaceholder+"@")

	return s
}

// LogInfo logs an informational message with token redaction
func LogInfo(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	redacted := RedactToken(message)
	log.Printf("[GitLab] INFO: %s", redacted)
}

// LogWarning logs a warning message with token redaction
func LogWarning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	redacted := RedactToken(message)
	log.Printf("[GitLab] WARNING: %s", redacted)
}

// LogError logs an error message with token redaction
func LogError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	redacted := RedactToken(message)
	log.Printf("[GitLab] ERROR: %s", redacted)
}

// RedactURL removes sensitive information from a Git URL
// Handles both GitLab (oauth2:token@) and GitHub (x-access-token:token@) formats
func RedactURL(gitURL string) string {
	// Parse the URL properly instead of string splitting
	parsedURL, err := url.Parse(gitURL)
	if err != nil {
		// If parsing fails, fall back to regex-based redaction
		return RedactToken(gitURL)
	}

	// Check if URL contains user info (credentials)
	if parsedURL.User != nil {
		// Redact the entire userinfo part (handles oauth2:token, x-access-token:token, etc.)
		parsedURL.User = url.User(TokenRedactionPlaceholder)
	}

	return parsedURL.String()
}

// SanitizeErrorMessage removes sensitive information from error messages
func SanitizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	message := err.Error()
	return RedactToken(message)
}
