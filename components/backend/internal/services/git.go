package services

import (
	"fmt"
	"strings"

	"ambient-code-backend/internal/types"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// LoadGitConfigFromConfigMapForProject loads Git configuration from a ConfigMap in the project namespace
func LoadGitConfigFromConfigMapForProject(c *gin.Context, reqK8s *kubernetes.Clientset, project string) (*types.GitConfig, error) {
	configMap, err := reqK8s.CoreV1().ConfigMaps(project).Get(c.Request.Context(), "git-config", v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get git-config ConfigMap: %v", err)
	}

	gitConfig := &types.GitConfig{}

	if name := configMap.Data["git-user-name"]; name != "" {
		if gitConfig.User == nil {
			gitConfig.User = &types.GitUser{}
		}
		gitConfig.User.Name = name
	}
	if email := configMap.Data["git-user-email"]; email != "" {
		if gitConfig.User == nil {
			gitConfig.User = &types.GitUser{}
		}
		gitConfig.User.Email = email
	}

	if sshKeySecret := configMap.Data["git-ssh-key-secret"]; sshKeySecret != "" {
		if gitConfig.Authentication == nil {
			gitConfig.Authentication = &types.GitAuthentication{}
		}
		gitConfig.Authentication.SSHKeySecret = &sshKeySecret
	}
	if tokenSecret := configMap.Data["git-token-secret"]; tokenSecret != "" {
		if gitConfig.Authentication == nil {
			gitConfig.Authentication = &types.GitAuthentication{}
		}
		gitConfig.Authentication.TokenSecret = &tokenSecret
	}

	if reposList := configMap.Data["git-repositories"]; reposList != "" {
		lines := strings.Split(strings.TrimSpace(reposList), "\n")
		var repos []types.GitRepository
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				repos = append(repos, types.GitRepository{URL: line, Branch: types.StringPtr("main")})
			}
		}
		if len(repos) > 0 {
			gitConfig.Repositories = repos
		}
	}

	return gitConfig, nil
}

// MergeGitConfigs merges user-provided GitConfig with ConfigMap defaults
func MergeGitConfigs(userConfig, defaultConfig *types.GitConfig) *types.GitConfig {
	if userConfig == nil && defaultConfig == nil {
		return nil
	}
	if userConfig == nil {
		return defaultConfig
	}
	if defaultConfig == nil {
		return userConfig
	}

	merged := &types.GitConfig{}
	if userConfig.User != nil {
		merged.User = userConfig.User
	} else if defaultConfig.User != nil {
		merged.User = defaultConfig.User
	}
	if userConfig.Authentication != nil {
		merged.Authentication = userConfig.Authentication
	} else if defaultConfig.Authentication != nil {
		merged.Authentication = defaultConfig.Authentication
	}

	if len(userConfig.Repositories) > 0 || len(defaultConfig.Repositories) > 0 {
		merged.Repositories = make([]types.GitRepository, 0, len(userConfig.Repositories)+len(defaultConfig.Repositories))
		merged.Repositories = append(merged.Repositories, userConfig.Repositories...)
		for _, def := range defaultConfig.Repositories {
			conflict := false
			for _, usr := range userConfig.Repositories {
				if usr.URL == def.URL {
					conflict = true
					break
				}
			}
			if !conflict {
				merged.Repositories = append(merged.Repositories, def)
			}
		}
	}
	return merged
}