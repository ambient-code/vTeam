"use client";

import { useEffect, useState } from "react";
import { ProjectSubpageHeader } from "@/components/project-subpage-header";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { RefreshCw, Save, Loader2 } from "lucide-react";
import { getApiUrl } from "@/lib/config";
import type { Project } from "@/types/project";
import { CheckCircle, XCircle, AlertCircle, ChevronDown, ChevronRight, Plus, Trash2 } from "lucide-react";

export default function ProjectSettingsPage({ params }: { params: Promise<{ name: string }> }) {
  const [projectName, setProjectName] = useState<string>("");
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [saving, setSaving] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [formData, setFormData] = useState({ displayName: "", description: "" });
  const [secretName, setSecretName] = useState<string>("");
  const [configSaving, setConfigSaving] = useState<boolean>(false);
  const [validationLoading, setValidationLoading] = useState<boolean>(false);
  const [validationResult, setValidationResult] = useState<{ valid: boolean; message: string } | null>(null);
  const [syncTriggering, setSyncTriggering] = useState<boolean>(false);
  const [showCreateSecret, setShowCreateSecret] = useState<boolean>(false);
  const [secretData, setSecretData] = useState<{ key: string; value: string }[]>([
    { key: "ANTHROPIC_API_KEY", value: "" },
    { key: "GITHUB_TOKEN", value: "" },
    { key: "GIT_TOKEN", value: "" },
    { key: "GIT_SSH_KEY", value: "" }
  ]);
  const [creatingSecret, setCreatingSecret] = useState<boolean>(false);
  const validateSourceSecret = async () => {
    if (!projectName) return;
    try {
      setValidationLoading(true);
      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets/validate`);
      if (response.ok) {
        const result = await response.json();
        setValidationResult(result);
      } else {
        setValidationResult({ valid: false, message: "Failed to validate secret" });
      }
    } catch (e) {
      setValidationResult({ valid: false, message: "Error validating secret" });
    } finally {
      setValidationLoading(false);
    }
  };

  const triggerSecretSync = async () => {
    if (!projectName) return;
    try {
      setSyncTriggering(true);
      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets/trigger-sync`, {
        method: "PUT",
      });
      if (response.ok) {
        const result = await response.json();
        // Trigger validation after sync to check status
        await validateSourceSecret();
      } else {
        throw new Error("Failed to trigger sync");
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to trigger secret sync");
    } finally {
      setSyncTriggering(false);
    }
  };

  const createSourceSecret = async () => {
    if (!projectName || !secretName.trim()) return;

    // Validate that ANTHROPIC_API_KEY is provided
    const anthropicKey = secretData.find(item => item.key === "ANTHROPIC_API_KEY")?.value?.trim();
    if (!anthropicKey) {
      setError("ANTHROPIC_API_KEY is required");
      return;
    }

    try {
      setCreatingSecret(true);
      setError(null);
      const apiUrl = getApiUrl();

      // Convert array to object, filtering out empty values
      const data: Record<string, string> = {};
      secretData.forEach(item => {
        if (item.key.trim() && item.value.trim()) {
          data[item.key.trim()] = item.value.trim();
        }
      });

      const response = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets/create-source`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          secretName: secretName.trim(),
          data: data
        }),
      });

      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: "Unknown error" }));
        throw new Error(err.message || err.error || "Failed to create source secret");
      }

      // Clear the form and hide the section
      setSecretData([
        { key: "ANTHROPIC_API_KEY", value: "" },
        { key: "GITHUB_TOKEN", value: "" },
        { key: "GIT_TOKEN", value: "" },
        { key: "GIT_SSH_KEY", value: "" }
      ]);
      setShowCreateSecret(false);

      // Re-validate to show the new status
      await validateSourceSecret();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create source secret");
    } finally {
      setCreatingSecret(false);
    }
  };

  const addSecretField = () => {
    setSecretData([...secretData, { key: "", value: "" }]);
  };

  const removeSecretField = (index: number) => {
    if (secretData.length > 1) {
      setSecretData(secretData.filter((_, i) => i !== index));
    }
  };

  const updateSecretField = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...secretData];
    updated[index][field] = value;
    setSecretData(updated);
  };

  useEffect(() => {
    params.then(({ name }) => setProjectName(name));
  }, [params]);

  useEffect(() => {
    const fetchProject = async () => {
      if (!projectName) return;
      try {
        const apiUrl = getApiUrl();
        const response = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}`);
        if (!response.ok) throw new Error("Failed to fetch project");
        const data: Project = await response.json();
        setProject(data);
        setFormData({ displayName: data.displayName || "", description: data.description || "" });
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to fetch project");
      } finally {
        setLoading(false);
      }
    };
    if (projectName) void fetchProject();
  }, [projectName]);

  useEffect(() => {
    const fetchRunnerSecretsConfig = async () => {
      if (!projectName) return;
      try {
        setValidationLoading(true);
        const apiUrl = getApiUrl();

        // Load current configuration
        const cfgRes = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets/config`);
        if (cfgRes.ok) {
          const cfg = await cfgRes.json();
          if (cfg.secretName) {
            setSecretName(cfg.secretName);
          } else {
            setSecretName("ambient-runner-secrets"); // Default operator secret name
          }
        } else {
          setSecretName("ambient-runner-secrets"); // Default operator secret name
        }

        // Validate the source secret automatically
        await validateSourceSecret();
      } catch {
        // noop
      } finally {
        setValidationLoading(false);
      }
    };
    if (projectName) void fetchRunnerSecretsConfig();
  }, [projectName]);

  const handleRefresh = () => {
    setLoading(true);
    setError(null);
    // re-run effect
    const apiUrl = getApiUrl();
    fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}`)
      .then((r) => r.json())
      .then((data: Project) => {
        setProject(data);
        setFormData({ displayName: data.displayName || "", description: data.description || "" });
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  const handleSave = async () => {
    if (!project) return;
    setSaving(true);
    setError(null);
    try {
      const apiUrl = getApiUrl();
      const payload = {
        name: project.name,
        displayName: formData.displayName.trim(),
        description: formData.description.trim() || undefined,
        annotations: project.annotations || {},
      } as Partial<Project> & { name: string };

      const response = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: "Unknown error" }));
        throw new Error(err.message || err.error || "Failed to update project");
      }
      const updated = await response.json();
      setProject(updated);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to update project");
    } finally {
      setSaving(false);
    }
  };

  const handleSaveConfig = async () => {
    if (!projectName) return;
    setConfigSaving(true);
    try {
      const apiUrl = getApiUrl();
      const name = (secretName.trim() || "ambient-runner-secrets");
      const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets/config`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ secretName: name }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: "Unknown error" }));
        throw new Error(err.message || err.error || "Failed to save secret config");
      }
      setSecretName(name);
      // Re-validate after configuration change
      await validateSourceSecret();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save secret config");
    } finally {
      setConfigSaving(false);
    }
  };


  return (
    <div className="container mx-auto p-6 max-w-4xl">
      <ProjectSubpageHeader
        title={<>Project Settings</>}
        description={<>{projectName}</>}
        actions={
          <Button variant="outline" onClick={handleRefresh} disabled={loading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        }
      />

      <Card>
        <CardHeader>
          <CardTitle>Edit Project</CardTitle>
          <CardDescription>Rename display name or update description</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <div className="p-2 rounded border border-red-200 bg-red-50 text-sm text-red-700">{error}</div>
          )}
          <div className="space-y-2">
            <Label htmlFor="displayName">Display Name</Label>
            <Input
              id="displayName"
              value={formData.displayName}
              onChange={(e) => setFormData((prev) => ({ ...prev, displayName: e.target.value }))}
              placeholder="My Awesome Project"
              maxLength={100}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={formData.description}
              onChange={(e) => setFormData((prev) => ({ ...prev, description: e.target.value }))}
              placeholder="Describe the purpose and goals of this project..."
              maxLength={500}
              rows={3}
            />
          </div>
          <div className="flex gap-3 pt-2">
            <Button onClick={handleSave} disabled={saving || loading || !project}>
              {saving ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Saving...
                </>
              ) : (
                <>
                  <Save className="w-4 h-4 mr-2" />
                  Save Changes
                </>
              )}
            </Button>
            <Button variant="outline" onClick={handleRefresh} disabled={saving || loading}>
              <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
              Reset
            </Button>
          </div>
        </CardContent>
      </Card>

      <div className="h-6" />

      <Card>
        <CardHeader>
          <CardTitle>Runner Secrets Configuration</CardTitle>
          <CardDescription>
            Configure which operator secret is used for this project. The operator copies secrets from its namespace to this project.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <div className="p-2 rounded border border-red-200 bg-red-50 text-sm text-red-700">{error}</div>
          )}

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="sourceSecretName">Source Secret Name</Label>
              <Input
                id="sourceSecretName"
                value={secretName}
                onChange={(e) => setSecretName(e.target.value)}
                placeholder="ambient-runner-secrets"
                maxLength={253}
                className="max-w-md"
              />
              <p className="text-sm text-muted-foreground">
                Name of the secret in the operator namespace containing API keys and git tokens.
              </p>
            </div>

            {/* Expected Keys Documentation */}
            <div className="bg-gray-50 p-4 rounded-lg">
              <h4 className="font-medium text-sm mb-2">Expected Secret Keys</h4>
              <ul className="text-sm space-y-1 text-muted-foreground">
                <li><strong>ANTHROPIC_API_KEY</strong>: API key for Claude/Anthropic services (required)</li>
                <li><strong>GITHUB_TOKEN</strong>: GitHub personal access token for repository operations (optional)</li>
                <li><strong>GIT_TOKEN</strong>: Git token for non-GitHub providers (optional)</li>
                <li><strong>GIT_SSH_KEY</strong>: SSH private key for git operations (optional)</li>
              </ul>
              <p className="text-xs text-muted-foreground mt-2">
                Git authentication is only needed for private repositories and push operations.
              </p>
            </div>

            {/* Validation Status */}
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Label>Source Secret Status</Label>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={validateSourceSecret}
                  disabled={validationLoading}
                >
                  {validationLoading ? (
                    <Loader2 className="w-4 h-4 mr-1 animate-spin" />
                  ) : (
                    <RefreshCw className="w-4 h-4 mr-1" />
                  )}
                  Check
                </Button>
              </div>

              {validationResult && (
                <div className={`flex items-center gap-2 p-3 rounded-lg ${
                  validationResult.valid
                    ? "bg-green-50 text-green-700 border border-green-200"
                    : "bg-red-50 text-red-700 border border-red-200"
                }`}>
                  {validationResult.valid ? (
                    <CheckCircle className="w-4 h-4" />
                  ) : (
                    <XCircle className="w-4 h-4" />
                  )}
                  <span className="text-sm">{validationResult.message}</span>
                </div>
              )}
            </div>

            {/* Action Buttons */}
            <div className="flex gap-3 pt-2">
              <Button onClick={handleSaveConfig} disabled={configSaving || !secretName.trim()}>
                {configSaving ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Saving...
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4 mr-2" />
                    Save Configuration
                  </>
                )}
              </Button>

              <Button variant="outline" onClick={triggerSecretSync} disabled={syncTriggering}>
                {syncTriggering ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Syncing...
                  </>
                ) : (
                  <>
                    <RefreshCw className="w-4 h-4 mr-2" />
                    Trigger Sync
                  </>
                )}
              </Button>
            </div>

            {/* Advanced: Create Source Secret (Not Recommended) */}
            <div className="border-t pt-4 mt-6">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowCreateSecret(!showCreateSecret)}
                className="text-orange-600 hover:text-orange-700"
              >
                {showCreateSecret ? (
                  <ChevronDown className="w-4 h-4 mr-2" />
                ) : (
                  <ChevronRight className="w-4 h-4 mr-2" />
                )}
                Advanced: Create Source Secret (Not Recommended)
              </Button>

              {showCreateSecret && (
                <div className="mt-4 space-y-4 bg-orange-50 p-4 rounded-lg border border-orange-200">
                  <div className="flex items-center gap-2 text-orange-700">
                    <AlertCircle className="w-4 h-4" />
                    <span className="text-sm font-medium">Warning: Not Recommended</span>
                  </div>
                  <p className="text-sm text-orange-600">
                    Creating secrets via UI is not recommended. Use operator-managed secrets for better security.
                    This option is provided for testing or environments where operator secrets are not available.
                  </p>

                  <div className="space-y-3">
                    <div className="space-y-2">
                      <Label>Secret Key-Value Pairs</Label>
                      {secretData.map((item, index) => (
                        <div key={index} className="flex gap-2 items-center">
                          <Input
                            placeholder="Key (e.g., ANTHROPIC_API_KEY)"
                            value={item.key}
                            onChange={(e) => updateSecretField(index, 'key', e.target.value)}
                            className="flex-1"
                          />
                          <Input
                            type="password"
                            placeholder="Value"
                            value={item.value}
                            onChange={(e) => updateSecretField(index, 'value', e.target.value)}
                            className="flex-1"
                          />
                          {secretData.length > 1 && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => removeSecretField(index)}
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          )}
                        </div>
                      ))}
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={addSecretField}
                        className="w-fit"
                      >
                        <Plus className="w-4 h-4 mr-2" />
                        Add Key
                      </Button>
                    </div>

                    <div className="flex gap-2">
                      <Button
                        onClick={createSourceSecret}
                        disabled={creatingSecret || !secretName.trim()}
                        className="bg-orange-600 hover:bg-orange-700"
                      >
                        {creatingSecret ? (
                          <>
                            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                            Creating...
                          </>
                        ) : (
                          <>
                            <Save className="w-4 h-4 mr-2" />
                            Create Source Secret
                          </>
                        )}
                      </Button>
                      <Button
                        variant="outline"
                        onClick={() => {
                          setShowCreateSecret(false);
                          setSecretData([
                            { key: "ANTHROPIC_API_KEY", value: "" },
                            { key: "GITHUB_TOKEN", value: "" },
                            { key: "GIT_TOKEN", value: "" },
                            { key: "GIT_SSH_KEY", value: "" }
                          ]);
                        }}
                        disabled={creatingSecret}
                      >
                        Cancel
                      </Button>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}