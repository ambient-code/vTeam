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
import { Plus, Trash2, Eye, EyeOff } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export default function ProjectSettingsPage({ params }: { params: Promise<{ name: string }> }) {
  const [projectName, setProjectName] = useState<string>("");
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [saving, setSaving] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [formData, setFormData] = useState({ displayName: "", description: "" });
  const [secretName, setSecretName] = useState<string>("");
  const [secrets, setSecrets] = useState<Array<{ key: string; value: string }>>([]);
  const [secretsLoading, setSecretsLoading] = useState<boolean>(true);
  const [secretsSaving, setSecretsSaving] = useState<boolean>(false);
  const [configSaving, setConfigSaving] = useState<boolean>(false);
  const [warnNoSecret, setWarnNoSecret] = useState<boolean>(false);
  const [secretList, setSecretList] = useState<Array<{ name: string }>>([]);
  const [mode, setMode] = useState<"existing" | "new">("existing");
  const [showValues, setShowValues] = useState<Record<number, boolean>>({});
  const [anthropicApiKey, setAnthropicApiKey] = useState<string>("");
  const [showAnthropicKey, setShowAnthropicKey] = useState<boolean>(false);
  const [gitUserName, setGitUserName] = useState<string>("");
  const [gitUserEmail, setGitUserEmail] = useState<string>("");
  const [gitToken, setGitToken] = useState<string>("");
  const [showGitToken, setShowGitToken] = useState<boolean>(false);
  const [jiraUrl, setJiraUrl] = useState<string>("");
  const [jiraProject, setJiraProject] = useState<string>("");
  const [jiraEmail, setJiraEmail] = useState<string>("");
  const [jiraToken, setJiraToken] = useState<string>("");
  const [showJiraToken, setShowJiraToken] = useState<boolean>(false);
  const FIXED_KEYS = ["ANTHROPIC_API_KEY","GIT_USER_NAME","GIT_USER_EMAIL","GIT_TOKEN","JIRA_URL","JIRA_PROJECT","JIRA_EMAIL","JIRA_API_TOKEN"] as const;
  const loadSecretValues = async (name: string) => {
    if (!name) return;
    try {
      setSecretsLoading(true);
      const apiUrl = getApiUrl();
      const secRes = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets`);
      if (secRes.ok) {
        const data = await secRes.json();
        const items = Object.entries<string>(data.data || {}).map(([k, v]) => ({ key: k, value: v }));
        const byKey: Record<string, string> = Object.fromEntries(items.map(it => [it.key, it.value]));
        setAnthropicApiKey(byKey["ANTHROPIC_API_KEY"] || "");
        setGitUserName(byKey["GIT_USER_NAME"] || "");
        setGitUserEmail(byKey["GIT_USER_EMAIL"] || "");
        setGitToken(byKey["GIT_TOKEN"] || "");
        setJiraUrl(byKey["JIRA_URL"] || "");
        setJiraProject(byKey["JIRA_PROJECT"] || "");
        setJiraEmail(byKey["JIRA_EMAIL"] || "");
        setJiraToken(byKey["JIRA_API_TOKEN"] || "");
        setSecrets(items.filter(it => !FIXED_KEYS.includes(it.key as typeof FIXED_KEYS[number])));
      } else {
        setSecrets([]);
        setAnthropicApiKey("");
        setGitUserName("");
        setGitUserEmail("");
        setGitToken("");
        setJiraUrl("");
        setJiraProject("");
        setJiraEmail("");
        setJiraToken("");
      }
    } finally {
      setSecretsLoading(false);
    }
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
    const fetchRunnerSecrets = async () => {
      if (!projectName) return;
      try {
        setSecretsLoading(true);
        const apiUrl = getApiUrl();
        // Load list of secrets for dropdown
        const listRes = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/secrets`);
        if (listRes.ok) {
          const list = await listRes.json();
          setSecretList((list.items || []).map((i: { name: string }) => ({ name: i.name })));
        }
        const cfgRes = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets/config`);
        if (cfgRes.ok) {
          const cfg = await cfgRes.json();
          const hasExisting = (secretList.length > 0);
          if (cfg.secretName) {
            setSecretName(cfg.secretName);
            setWarnNoSecret(false);
            setMode("existing");
          } else {
            setSecretName("ambient-runner-secrets");
            setWarnNoSecret(false);
            setMode(hasExisting ? "existing" : "new");
          }
          if (cfg.secretName) {
            await loadSecretValues(cfg.secretName);
          } else {
            setSecrets([]);
          }
        }
      } catch {
        // noop
      } finally {
        setSecretsLoading(false);
      }
    };
    if (projectName) void fetchRunnerSecrets();
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
      setWarnNoSecret(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save secret config");
    } finally {
      setConfigSaving(false);
    }
  };

  const handleSaveSecrets = async () => {
    if (!projectName) return;
    // Always persist config first (auto default name when creating new)
    await handleSaveConfig();
    setSecretsSaving(true);
    try {
      const apiUrl = getApiUrl();
      const data: Record<string, string> = {};
      // Persist Anthropic API key (required)
      if (anthropicApiKey) data["ANTHROPIC_API_KEY"] = anthropicApiKey;
      // Persist Git convenience fields into the secret under fixed keys
      if (gitUserName) data["GIT_USER_NAME"] = gitUserName;
      if (gitUserEmail) data["GIT_USER_EMAIL"] = gitUserEmail;
      if (gitToken) data["GIT_TOKEN"] = gitToken;
      // Persist Jira convenience fields into the secret under fixed keys
      if (jiraUrl) data["JIRA_URL"] = jiraUrl;
      if (jiraProject) data["JIRA_PROJECT"] = jiraProject;
      if (jiraEmail) data["JIRA_EMAIL"] = jiraEmail;
      if (jiraToken) data["JIRA_API_TOKEN"] = jiraToken;
      // Add remaining dynamic keys (excluding fixed keys)
      for (const { key, value } of secrets) {
        if (!key) continue;
        if (FIXED_KEYS.includes(key as typeof FIXED_KEYS[number])) continue;
        data[key] = value ?? "";
      }
      const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ data }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: "Unknown error" }));
        throw new Error(err.message || err.error || "Failed to save secrets");
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save secrets");
    } finally {
      setSecretsSaving(false);
    }
  };

  const addSecretRow = () => {
    setSecrets((prev) => [...prev, { key: "", value: "" }]);
  };

  const removeSecretRow = (idx: number) => {
    setSecrets((prev) => prev.filter((_, i) => i !== idx));
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
          <CardTitle>Runner Secrets</CardTitle>
          <CardDescription>
            Configure the Secret and manage key/value pairs used by project runners.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <div className="p-2 rounded border border-red-200 bg-red-50 text-sm text-red-700">{error}</div>
          )}
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-3">
              <div>
                <Label>Runner Secret</Label>
                <div className="text-sm text-muted-foreground">Using: {secretName || "ambient-runner-secrets"}</div>
              </div>
            </div>
            <Tabs value={mode} onValueChange={(v) => setMode(v as typeof mode)}>
              <TabsList>
                <TabsTrigger value="existing">Use existing</TabsTrigger>
                <TabsTrigger value="new">Create new</TabsTrigger>
              </TabsList>
              <TabsContent value="existing">
                <div className="flex gap-2 items-center pt-2">
                  {secretList.length > 0 && (
                    <Select
                      value={secretName}
                      onValueChange={(val) => {
                        setSecretName(val);
                        void loadSecretValues(val);
                      }}
                    >
                      <SelectTrigger className="w-80">
                        <SelectValue placeholder="Select a secret..." />
                      </SelectTrigger>
                      <SelectContent>
                        {secretList.map((s) => (
                          <SelectItem key={s.name} value={s.name}>{s.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                </div>
                {secretList.length === 0 ? (
                  <div className="mt-2 text-sm text-amber-600">No runner secrets found in this project. Use the &quot;Create new&quot; tab to create one.</div>
                ) : (!secretName ? (
                  <div className="mt-2 text-sm text-muted-foreground">No secret selected. You can still add key/value pairs below and Save; they will be written to the default secret name.</div>
                ) : null)}
              </TabsContent>
              <TabsContent value="new">
                <div className="flex gap-2 items-center pt-2">
                  <Input
                    id="secretName"
                    value={secretName}
                    onChange={(e) => setSecretName(e.target.value)}
                    placeholder="ambient-runner-secrets"
                    maxLength={253}
                  />
                </div>
              </TabsContent>
            </Tabs>
          </div>

          {(mode === "new" || (mode === "existing" && !!secretName)) && (
            <div className="pt-2 space-y-2">
              <div className="flex items-center justify-between">
                <Label>Key/Value Pairs</Label>
                <Button variant="outline" onClick={addSecretRow} disabled={secretsLoading}>
                  <Plus className="w-4 h-4 mr-2" /> Add Row
                </Button>
              </div>
              {secretsLoading ? (
                <div className="text-sm text-muted-foreground">Loading secrets...</div>
              ) : (
                <div className="space-y-2">
                  {secrets.length === 0 && (
                    <div className="text-sm text-muted-foreground">No keys configured.</div>
                  )}
                  {secrets.map((item, idx) => (
                    <div key={idx} className="flex gap-2 items-center">
                      <Input
                        value={item.key}
                        onChange={(e) =>
                          setSecrets((prev) => prev.map((it, i) => (i === idx ? { ...it, key: e.target.value } : it)))
                        }
                        placeholder="KEY"
                        className="w-1/3"
                      />
                      <div className="flex-1 flex items-center gap-2">
                        <Input
                          type={showValues[idx] ? "text" : "password"}
                          value={item.value}
                          onChange={(e) =>
                            setSecrets((prev) => prev.map((it, i) => (i === idx ? { ...it, value: e.target.value } : it)))
                          }
                          placeholder="value"
                          className="flex-1"
                        />
                        <Button
                          type="button"
                          variant="ghost"
                          onClick={() => setShowValues((prev) => ({ ...prev, [idx]: !prev[idx] }))}
                          aria-label={showValues[idx] ? "Hide value" : "Show value"}
                        >
                          {showValues[idx] ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                        </Button>
                      </div>
                      <Button variant="ghost" onClick={() => removeSecretRow(idx)} aria-label="Remove row">
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
              <div className="pt-4 space-y-3 border-t">
                <div className="pt-3">
                  <Label className="text-base font-semibold">Anthropic API Key (Required)</Label>
                  <div className="text-xs text-muted-foreground mb-3">Your Anthropic API key for Claude Code runner</div>
                  <div className="flex items-center gap-2">
                    <Input
                      id="anthropicApiKey"
                      type={showAnthropicKey ? "text" : "password"}
                      placeholder="sk-ant-..."
                      value={anthropicApiKey}
                      onChange={(e) => setAnthropicApiKey(e.target.value)}
                      className="flex-1"
                    />
                    <Button type="button" variant="ghost" onClick={() => setShowAnthropicKey((v) => !v)} aria-label={showAnthropicKey ? "Hide key" : "Show key"}>
                      {showAnthropicKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </Button>
                  </div>
                </div>
              </div>
              <div className="pt-4 space-y-3 border-t">
                <div className="pt-3">
                  <Label className="text-base font-semibold">Git Integration (Optional)</Label>
                  <div className="text-xs text-muted-foreground mb-3">Configure Git credentials for repository operations (clone, commit, push)</div>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label htmlFor="gitUserName">Git User Name</Label>
                    <Input id="gitUserName" placeholder="Your Name" value={gitUserName} onChange={(e) => setGitUserName(e.target.value)} />
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor="gitUserEmail">Git User Email</Label>
                    <Input id="gitUserEmail" placeholder="you@example.com" value={gitUserEmail} onChange={(e) => setGitUserEmail(e.target.value)} />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="gitToken">GitHub API Token</Label>
                  <div className="flex items-center gap-2">
                    <Input
                      id="gitToken"
                      type={showGitToken ? "text" : "password"}
                      placeholder="ghp_... or glpat-..."
                      value={gitToken}
                      onChange={(e) => setGitToken(e.target.value)}
                      className="flex-1"
                    />
                    <Button type="button" variant="ghost" onClick={() => setShowGitToken((v) => !v)} aria-label={showGitToken ? "Hide token" : "Show token"}>
                      {showGitToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </Button>
                  </div>
                  <div className="text-xs text-muted-foreground">GitHub personal access token or fine-grained token for git operations and API access</div>
                </div>
                <div className="text-xs text-muted-foreground">Git credentials will be saved with keys: GIT_USER_NAME, GIT_USER_EMAIL, GIT_TOKEN</div>
              </div>
              <div className="pt-4 space-y-3 border-t">
                <div className="pt-3">
                  <Label className="text-base font-semibold">Jira Integration (Optional)</Label>
                  <div className="text-xs text-muted-foreground mb-3">Configure Jira integration for issue management</div>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label htmlFor="jiraUrl">Jira Base URL</Label>
                    <Input id="jiraUrl" placeholder="https://your-domain.atlassian.net" value={jiraUrl} onChange={(e) => setJiraUrl(e.target.value)} />
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor="jiraProject">Jira Project Key</Label>
                    <Input id="jiraProject" placeholder="ABC" value={jiraProject} onChange={(e) => setJiraProject(e.target.value)} />
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor="jiraEmail">Jira Email/Username</Label>
                    <Input id="jiraEmail" placeholder="you@example.com" value={jiraEmail} onChange={(e) => setJiraEmail(e.target.value)} />
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor="jiraToken">Jira API Token</Label>
                    <div className="flex items-center gap-2">
                      <Input id="jiraToken" type={showJiraToken ? "text" : "password"} placeholder="token" value={jiraToken} onChange={(e) => setJiraToken(e.target.value)} />
                      <Button type="button" variant="ghost" onClick={() => setShowJiraToken((v) => !v)} aria-label={showJiraToken ? "Hide token" : "Show token"}>
                        {showJiraToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </Button>
                    </div>
                  </div>
                </div>
                <div className="text-xs text-muted-foreground">Jira credentials will be saved with keys: JIRA_URL, JIRA_PROJECT, JIRA_EMAIL, JIRA_API_TOKEN</div>
              </div>
            </div>
          )}

          <div className="pt-2">
              <Button onClick={async () => {
                await handleSaveSecrets();
                setWarnNoSecret(false);
              }} disabled={secretsSaving || secretsLoading || (mode === "existing" && (secretList.length === 0 || !secretName))}>
                {secretsSaving ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Saving Secrets
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4 mr-2" />
                    Save Secrets
                  </>
                )}
              </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}