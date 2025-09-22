package resources

// Resource names used throughout the RFE controller
const (
	// Secrets (RFE-controller namespace specific)
	DefaultRunnerSecretsName = "ambient-runner-secrets"

	// Secret keys - expected contents of rfe-controller-secrets
	AnthropicAPIKeySecretKey = "ANTHROPIC_API_KEY"
	GitHubTokenSecretKey     = "GITHUB_TOKEN"
	GitTokenSecretKey        = "GIT_TOKEN"        // Alternative to GITHUB_TOKEN for other git providers
	GitSSHKeySecretKey       = "GIT_SSH_KEY"      // Optional: for SSH-based git access

	// ConfigMaps (RFE-controller namespace specific)
	GitConfigMapName    = "rfe-controller-git-config"
	RunnerConfigMapName = "rfe-controller-runner-config"

	// PersistentVolumeClaims (RFE-controller namespace specific)
	WorkspacePVCName = "rfe-controller-workspace"

	// Services and Deployments (RFE-controller namespace specific)
	ContentServiceName = "rfe-controller-content"

	// Labels and selectors (shared/central)
	AppLabelKey               = "app"
	ManagedLabelKey          = "ambient-code.io/managed"
	ManagedLabelValue        = "true"
	SecretTypeLabelKey       = "ambient-code.io/secret-type"
	RunnerSecretsLabelValue  = "runner-secrets"
	ConfigTypeLabelKey       = "ambient-code.io/config-type"
	GitConfigLabelValue      = "git"
	RunnerConfigLabelValue   = "runner"
	CopiedFromLabelKey       = "ambient-code.io/copied-from"
	CopiedAtAnnotationKey    = "ambient-code.io/copied-at"
)