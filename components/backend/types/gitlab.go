package types

import "time"

// GitLabConnection represents a user's connection to GitLab (GitLab.com or self-hosted)
type GitLabConnection struct {
	UserID       string    `json:"userId"`       // vTeam user identifier
	GitLabUserID string    `json:"gitlabUserId"` // GitLab user ID (from /user API)
	InstanceURL  string    `json:"instanceUrl"`  // e.g., "https://gitlab.com" or "https://gitlab.company.com"
	Username     string    `json:"username"`     // GitLab username
	UpdatedAt    time.Time `json:"updatedAt"`    // Last connection update
}

// GitLabRepository extends GitRepository for GitLab-specific attributes
// Internal parsed representation (not persisted to CRD)
type ParsedGitLabRepo struct {
	Host      string // "gitlab.com" or "gitlab.example.com"
	Owner     string // Repository owner/namespace
	Repo      string // Repository name
	APIURL    string // Constructed API base URL (e.g., "https://gitlab.com/api/v4")
	ProjectID string // URL-encoded project path (owner%2Frepo) for API calls
}

// GitLabAPIError represents structured error type for GitLab API failures
type GitLabAPIError struct {
	StatusCode  int                    `json:"statusCode"`  // HTTP status code
	Message     string                 `json:"message"`     // User-friendly error message
	Remediation string                 `json:"remediation"` // Actionable guidance for user
	RawError    string                 `json:"rawError"`    // Original error from GitLab API
	RequestID   string                 `json:"requestId"`   // GitLab request ID for debugging
	Metadata    map[string]interface{} `json:"metadata"`    // Additional context
}

// Error implements the error interface
func (e *GitLabAPIError) Error() string {
	if e.Remediation != "" {
		return e.Message + ". " + e.Remediation
	}
	return e.Message
}

// GitLabBranch represents a Git branch in a GitLab repository
type GitLabBranch struct {
	Name      string       `json:"name"`
	Commit    GitLabCommit `json:"commit"`
	Protected bool         `json:"protected"`
	Default   bool         `json:"default"`
}

// GitLabCommit represents commit information
type GitLabCommit struct {
	ID            string    `json:"id"`          // SHA
	ShortID       string    `json:"short_id"`    // Short SHA
	Title         string    `json:"title"`       // Commit title
	Message       string    `json:"message"`     // Full commit message
	AuthorName    string    `json:"author_name"` // Author name
	AuthorEmail   string    `json:"author_email"`
	CommittedDate time.Time `json:"committed_date"`
}

// GitLabTreeEntry represents a file or directory entry in a GitLab repository tree
type GitLabTreeEntry struct {
	ID   string `json:"id"`   // Object SHA
	Name string `json:"name"` // File/directory name
	Type string `json:"type"` // "blob" or "tree"
	Path string `json:"path"` // Full path from repository root
	Mode string `json:"mode"` // File mode (e.g., "100644")
}
