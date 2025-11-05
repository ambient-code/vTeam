package gitlab

import (
	"fmt"
	"log"
	"regexp"
	"strings"
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
func RedactURL(gitURL string) string {
	// Remove credentials from URLs like https://oauth2:token@gitlab.com/...
	if strings.Contains(gitURL, "@") {
		parts := strings.Split(gitURL, "@")
		if len(parts) == 2 {
			// Keep the protocol and domain, redact credentials
			protocolParts := strings.Split(parts[0], "://")
			if len(protocolParts) == 2 {
				return fmt.Sprintf("%s://%s@%s", protocolParts[0], TokenRedactionPlaceholder, parts[1])
			}
		}
	}

	return gitURL
}

// SanitizeErrorMessage removes sensitive information from error messages
func SanitizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	message := err.Error()
	return RedactToken(message)
}
