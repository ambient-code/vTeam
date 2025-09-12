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
  const loadSecretValues = async (name: string) => {
    if (!name) return;
    try {
      setSecretsLoading(true);
      const apiUrl = getApiUrl();
      const secRes = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets`);
      if (secRes.ok) {
        const data = await secRes.json();
        const items = Object.entries<string>(data.data || {}).map(([k, v]) => ({ key: k, value: v }));
        setSecrets(items);
      } else {
        setSecrets([]);
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
      for (const { key, value } of secrets) {
        if (!key) continue;
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