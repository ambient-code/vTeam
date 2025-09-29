package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type yamlAgent struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Content     string `yaml:"content"`
}

type agentSummary struct {
	Persona     string `json:"persona"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Description string `json:"description"`
}

func resolveAgentsDir() string {
	if v := os.Getenv("AGENTS_DIR"); v != "" {
		return v
	}
	// Default inside container image
	if _, err := os.Stat("/app/agents"); err == nil {
		return "/app/agents"
	}
	// Fallback to repo-relative path when running locally from backend directory
	return filepath.Clean("../runners/claude-code-runner/agents")
}

func readAllAgentYAMLs(dir string) ([]yamlAgent, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []yamlAgent
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") || strings.HasPrefix(name, "agent-schema") || name == "README.yaml" {
			continue
		}
		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var a yamlAgent
		if err := yaml.Unmarshal(b, &a); err != nil {
			continue
		}
		if a.Name != "" {
			out = append(out, a)
		}
	}
	return out, nil
}

func ListAgents(c *gin.Context) {
	dir := resolveAgentsDir()
	agents, err := readAllAgentYAMLs(dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read agents: %v", err)})
		return
	}
	resp := make([]agentSummary, 0, len(agents))
	for _, a := range agents {
		// Extract persona from name (e.g., "Archie (Architect)" -> "archie-architect")
		persona := extractPersonaFromName(a.Name)
		// Extract role from name (e.g., "Archie (Architect)" -> "Architect")
		role := extractRoleFromName(a.Name)

		resp = append(resp, agentSummary{
			Persona:     persona,
			Name:        a.Name,
			Role:        role,
			Description: a.Description,
		})
	}
	c.JSON(http.StatusOK, resp)
}

func GetAgentMarkdown(c *gin.Context) {
	persona := c.Param("persona")
	if persona == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "persona required"})
		return
	}
	md, err := renderAgentMarkdownContent(persona)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "persona not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to render agent markdown: %v", err)})
		return
	}
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(md))
}

// renderAgentMarkdownContent builds the markdown content for a given agent persona
// by reading its YAML definition from the configured agents directory.
func renderAgentMarkdownContent(persona string) (string, error) {
	if strings.TrimSpace(persona) == "" {
		return "", fmt.Errorf("persona required")
	}
	dir := resolveAgentsDir()
	agents, err := readAllAgentYAMLs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read agents: %w", err)
	}
	var found *yamlAgent
	for i := range agents {
		agentPersona := extractPersonaFromName(agents[i].Name)
		if strings.EqualFold(agentPersona, persona) {
			found = &agents[i]
			break
		}
	}
	if found == nil {
		return "", fmt.Errorf("persona not found")
	}
	var sb strings.Builder

	// --- YAML Front Matter ---
	displayName := found.Name
	description := found.Description

	// Extract tools from description or use default
	tools := "Read, Write, Edit, Bash, Glob, Grep"
	if strings.Contains(strings.ToLower(description), "websearch") {
		tools += ", WebSearch"
	}
	if strings.Contains(strings.ToLower(description), "webfetch") {
		tools += ", WebFetch"
	}

	fmt.Fprintf(&sb, "---\n")
	fmt.Fprintf(&sb, "name: %s\n", displayName)
	fmt.Fprintf(&sb, "description: %s\n", description)
	fmt.Fprintf(&sb, "tools: %s\n", tools)
	fmt.Fprintf(&sb, "---\n\n")

	// --- Agent Content ---
	fmt.Fprintf(&sb, "%s\n", found.Content)
	return sb.String(), nil
}

// extractPersonaFromName extracts persona from name format like "Archie Architect" -> "archie-architect"
func extractPersonaFromName(name string) string {
	// Extract first name and role from format like "Archie Architect"
	parts := strings.Fields(name)
	if len(parts) >= 2 {
		firstName := strings.ToLower(parts[0])
		role := strings.ToLower(strings.Join(parts[1:], "_"))
		return firstName + "-" + role
	}
	// Fallback: just convert name to lowercase with dashes
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

// extractRoleFromName extracts role from name format like "Archie Architect" -> "Architect"
func extractRoleFromName(name string) string {
	parts := strings.Fields(name)
	if len(parts) >= 2 {
		return strings.Join(parts[1:], " ")
	}
	return ""
}

// titleCaseFromSnakeOrUpper converts strings like "ENGINEERING_MANAGER" or "engineering manager"
// into "Engineering Manager".
func titleCaseFromSnakeOrUpper(s string) string {
	s = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), "_", " ")
	parts := strings.Fields(s)
	for i := range parts {
		p := parts[i]
		if len(p) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}