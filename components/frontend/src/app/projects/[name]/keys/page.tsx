"use client";

import { useEffect, useMemo, useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { getApiUrl } from "@/lib/config";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { AlertTriangle, Copy, KeyRound, Loader2, Plus, RefreshCw, Trash2, Eye, Edit, Shield } from "lucide-react";
import { ProjectSubpageHeader } from "@/components/project-subpage-header";

type ProjectKey = {
  id: string;
  name: string;
  description?: string;
  createdAt?: string;
  lastUsedAt?: string;
  role?: "view" | "edit" | "admin";
};

type ListResponse = { items: ProjectKey[] };

export default function ProjectKeysPage({ params }: { params: Promise<{ name: string }> }) {
  const [projectName, setProjectName] = useState<string>("");
  const [keys, setKeys] = useState<ProjectKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyDesc, setNewKeyDesc] = useState("");
  const [newKeyRole, setNewKeyRole] = useState<"view" | "edit" | "admin">("edit");
  const [oneTimeKey, setOneTimeKey] = useState<string | null>(null);
  const [oneTimeKeyName, setOneTimeKeyName] = useState<string>("");

  const apiUrl = useMemo(() => getApiUrl(), []);

  const ROLE_DEFINITIONS = useMemo(() => ({
    view: {
      label: "View",
      description: "Can see sessions and duplicate to their own project",
      color: "bg-blue-100 text-blue-800",
      icon: Eye,
    },
    edit: {
      label: "Edit",
      description: "Can create sessions in the project",
      color: "bg-green-100 text-green-800",
      icon: Edit,
    },
    admin: {
      label: "Admin",
      description: "Full project management access",
      color: "bg-purple-100 text-purple-800",
      icon: Shield,
    },
  } as const), []);

  const fetchKeys = async () => {
    if (!projectName) return;
    try {
      setError(null);
      const res = await fetch(`${apiUrl}/projects/${projectName}/keys`);
      if (!res.ok) {
        const data = await res.json().catch(() => ({ error: "Failed to fetch keys" }));
        throw new Error(data.error || "Failed to fetch keys");
      }
      const data: ListResponse = await res.json();
      setKeys(data.items || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
      setKeys([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    params.then(({ name }) => setProjectName(name));
  }, [params]);

  useEffect(() => {
    if (projectName) {
      fetchKeys();
      const i = setInterval(fetchKeys, 30000);
      return () => clearInterval(i);
    }
  }, [projectName]);

  const handleCreate = async () => {
    if (!newKeyName.trim()) return;
    try {
      setCreating(true);
      setError(null);
      setOneTimeKey(null);
      const res = await fetch(`${apiUrl}/projects/${projectName}/keys`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: newKeyName.trim(), description: newKeyDesc.trim() || undefined, role: newKeyRole }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({ error: "Failed to create key" }));
        throw new Error(data.error || "Failed to create key");
      }
      const data: { id: string; name: string; key: string; description?: string } = await res.json();
      setOneTimeKey(data.key);
      setOneTimeKeyName(data.name);
      setNewKeyName("");
      setNewKeyDesc("");
      setShowCreate(false);
      await fetchKeys();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create key");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (keyId: string) => {
    try {
      setDeleting(keyId);
      setError(null);
      const res = await fetch(`${apiUrl}/projects/${projectName}/keys/${keyId}`, { method: "DELETE" });
      if (!res.ok && res.status !== 204) {
        const data = await res.json().catch(() => ({ error: "Failed to delete key" }));
        throw new Error(data.error || "Failed to delete key");
      }
      await fetchKeys();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to delete key");
    } finally {
      setDeleting(null);
    }
  };

  const copy = async (text: string) => {
    try { await navigator.clipboard.writeText(text); } catch {}
  };

  if (!projectName || (loading && keys.length === 0)) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="animate-spin h-8 w-8" />
          <span className="ml-2">Loading access keys...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6">
      <ProjectSubpageHeader
        title={<><KeyRound className="w-6 h-6" />Access Keys</>}
        description={<>Create and manage API keys for non-user access</>}
        actions={
          <>
            <Button onClick={() => setShowCreate(true)}>
              <Plus className="w-4 h-4 mr-2" />
              Create Key
            </Button>
            <Button variant="outline" onClick={fetchKeys} disabled={loading}>
              <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
              Refresh
            </Button>
          </>
        }
      />

      {error && (
        <Card className="mb-6 border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-red-600" />
              <p className="text-red-700">{error}</p>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <KeyRound className="w-5 h-5" />
            Access Keys ({keys.length})
          </CardTitle>
          <CardDescription>API keys scoped to this project</CardDescription>
        </CardHeader>
        <CardContent>
          {keys.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Description</TableHead>
              <TableHead>Created</TableHead>
                  <TableHead>Last Used</TableHead>
              <TableHead>Role</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {keys.map(k => (
                  <TableRow key={k.id}>
                    <TableCell className="font-medium">{k.name}</TableCell>
                    <TableCell>{k.description || <span className="text-muted-foreground italic">No description</span>}</TableCell>
                    <TableCell>
                      {k.createdAt ? formatDistanceToNow(new Date(k.createdAt), { addSuffix: true }) : <span className="text-muted-foreground">Unknown</span>}
                    </TableCell>
                    <TableCell>
                      {k.lastUsedAt ? formatDistanceToNow(new Date(k.lastUsedAt), { addSuffix: true }) : <span className="text-muted-foreground">Never</span>}
                    </TableCell>
                    <TableCell>
                      {k.role ? (
                        (() => {
                          const role = k.role as keyof typeof ROLE_DEFINITIONS;
                          const cfg = ROLE_DEFINITIONS[role];
                          const Icon = cfg.icon;
                          return (
                            <Badge className={cfg.color} style={{ cursor: "default" }}>
                              <Icon className="w-3 h-3 mr-1" />
                              {cfg.label}
                            </Badge>
                          );
                        })()
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Button variant="ghost" size="sm" onClick={() => handleDelete(k.id)} disabled={deleting === k.id}>
                        {deleting === k.id ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className="text-center py-12">
              <KeyRound className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">No access keys</h3>
              <p className="text-muted-foreground mb-4">Create an API key to enable non-user access</p>
              <Button onClick={() => setShowCreate(true)}>
                <Plus className="w-4 h-4 mr-2" />
                Create Your First Key
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Key Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Create Access Key</DialogTitle>
            <DialogDescription>Provide a name and optional description</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="key-name">Name *</Label>
              <Input id="key-name" value={newKeyName} onChange={(e) => setNewKeyName(e.target.value)} placeholder="my-ci-key" maxLength={64} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="key-desc">Description</Label>
              <Input id="key-desc" value={newKeyDesc} onChange={(e) => setNewKeyDesc(e.target.value)} placeholder="Used by CI pipelines" maxLength={200} />
            </div>
          <div className="space-y-2">
            <Label>Role</Label>
            <div className="space-y-3">
              {(["view","edit","admin"] as const).map((roleKey) => {
                const cfg = ROLE_DEFINITIONS[roleKey];
                const Icon = cfg.icon;
                const id = `key-role-${roleKey}`;
                return (
                  <div key={roleKey} className="flex items-start gap-3">
                    <input
                      type="radio"
                      name="key-role"
                      id={id}
                      className="mt-1 h-4 w-4"
                      value={roleKey}
                      checked={newKeyRole === roleKey}
                      onChange={() => setNewKeyRole(roleKey)}
                      disabled={creating}
                    />
                    <Label htmlFor={id} className="flex-1 cursor-pointer">
                      <div className="flex items-center gap-2">
                        <Icon className="w-4 h-4" />
                        <span className="font-medium">{cfg.label}</span>
                      </div>
                      <div className="text-sm text-muted-foreground ml-6">{cfg.description}</div>
                    </Label>
                  </div>
                );
              })}
            </div>
          </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)} disabled={creating}>Cancel</Button>
            <Button onClick={handleCreate} disabled={creating || !newKeyName.trim()}>
              {creating ? (<><Loader2 className="w-4 h-4 mr-2 animate-spin" />Creating...</>) : "Create Key"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* One-time Key Viewer */}
      <Dialog open={oneTimeKey !== null} onOpenChange={(open) => !open && setOneTimeKey(null)}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Copy Your New Access Key</DialogTitle>
            <DialogDescription>
              This is the only time the full key will be shown. Store it securely. Key name: <b>{oneTimeKeyName}</b>
            </DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-2">
            <code className="text-sm bg-muted px-2 py-2 rounded break-all w-full">{oneTimeKey || ""}</code>
            <Button variant="ghost" size="sm" onClick={() => oneTimeKey && copy(oneTimeKey)}>
              <Copy className="w-4 h-4" />
            </Button>
          </div>
          <DialogFooter>
            <Button onClick={() => setOneTimeKey(null)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}


