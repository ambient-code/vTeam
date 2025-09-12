"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { PermissionAssignment, PermissionRole, SubjectType } from "@/types/project";
import { Eye, Edit, Shield, Users, User as UserIcon, Plus, RefreshCw, Loader2, Trash2, Info } from "lucide-react";
import { getApiUrl } from "@/lib/config";
import { ProjectSubpageHeader } from "@/components/project-subpage-header";

const ROLE_DEFINITIONS = {
  view: {
    label: "View",
    description: "Can see sessions and duplicate to their own project",
    permissions: ["sessions:read", "sessions:duplicate"] as const,
    color: "bg-blue-100 text-blue-800",
    icon: Eye,
  },
  edit: {
    label: "Edit",
    description: "Can create sessions in the project",
    permissions: ["sessions:read", "sessions:create", "sessions:duplicate"] as const,
    color: "bg-green-100 text-green-800",
    icon: Edit,
  },
  admin: {
    label: "Admin",
    description: "Full project management access",
    permissions: ["*"] as const,
    color: "bg-purple-100 text-purple-800",
    icon: Shield,
  },
} as const;

type GrantPermissionForm = {
  subjectType: SubjectType;
  subjectName: string;
  role: PermissionRole;
};

type ProjectDetails = {
  name: string;
  displayName: string;
  userRole: PermissionRole;
};

export default function PermissionsPage({ params }: { params: Promise<{ name: string }> }) {
  const router = useRouter();
  const [projectName, setProjectName] = useState<string>("");
  const [items, setItems] = useState<PermissionAssignment[]>([]);
  const [project, setProject] = useState<ProjectDetails | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [showGrantDialog, setShowGrantDialog] = useState(false);
  const [grantForm, setGrantForm] = useState<GrantPermissionForm>({ subjectType: "group", subjectName: "", role: "view" });
  const [granting, setGranting] = useState(false);
  const [grantError, setGrantError] = useState<string | null>(null);

  const [showRevokeDialog, setShowRevokeDialog] = useState(false);
  const [toRevoke, setToRevoke] = useState<PermissionAssignment | null>(null);
  const [revoking, setRevoking] = useState(false);
  const [revokeError, setRevokeError] = useState<string | null>(null);

  const isAdmin = project?.userRole === "admin";

  useEffect(() => {
    params.then(({ name }) => setProjectName(name));
  }, [params]);

  useEffect(() => {
    if (projectName) {
      void fetchProject();
      void fetchPermissions();
    }
  }, [projectName]);

  async function fetchProject() {
    try {
      const apiUrl = getApiUrl();
      const resp = await fetch(`${apiUrl}/projects/${projectName}`);
      const data = await resp.json().catch(() => ({}));

      // Access role
      let userRole: PermissionRole = "view";
      try {
        const accessResp = await fetch(`${apiUrl}/projects/${projectName}/access`);
        if (accessResp.ok) {
          const accessData = await accessResp.json();
          if (accessData?.userRole === "admin" || accessData?.userRole === "edit" || accessData?.userRole === "view") {
            userRole = accessData.userRole;
          }
        }
      } catch {}

      setProject({ name: projectName, displayName: data.displayName || data.name || projectName, userRole });
    } catch (e) {
      // non-fatal
    }
  }

  async function fetchPermissions() {
    setLoading(true);
    setError(null);
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/projects/${projectName}/permissions`);
      if (!response.ok) {
        throw new Error("Failed to fetch permissions");
      }
      const raw = await response.json();
      const data: PermissionAssignment[] = Array.isArray(raw) ? raw : raw.items ?? [];
      setItems(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }

  function handleRefresh() {
    void fetchPermissions();
  }

  async function handleGrant() {
    if (!grantForm.subjectName.trim()) {
      setGrantError(`${grantForm.subjectType === "group" ? "Group" : "User"} name is required`);
      return;
    }
    const key = `${grantForm.subjectType}:${grantForm.subjectName}`.toLowerCase();
    if (items.some(i => `${i.subjectType}:${i.subjectName}`.toLowerCase() === key)) {
      setGrantError("This subject already has access to the project");
      return;
    }

    setGranting(true);
    setGrantError(null);
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/projects/${projectName}/permissions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(grantForm),
      });
      if (!response.ok) {
        const errData = await response.json().catch(() => ({ error: "Unknown error" }));
        throw new Error(errData.message || errData.error || "Failed to grant permission");
      }
      await fetchPermissions();
      setShowGrantDialog(false);
      setGrantForm({ subjectType: "group", subjectName: "", role: "view" });
    } catch (err) {
      setGrantError(err instanceof Error ? err.message : "Failed to grant permission");
    } finally {
      setGranting(false);
    }
  }

  async function handleRevoke() {
    if (!toRevoke) return;
    setRevoking(true);
    setRevokeError(null);
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/projects/${projectName}/permissions/${toRevoke.subjectType}/${encodeURIComponent(toRevoke.subjectName)}`, { method: "DELETE" });
      if (!response.ok) {
        const errData = await response.json().catch(() => ({ error: "Unknown error" }));
        throw new Error(errData.message || errData.error || "Failed to revoke permission");
      }
      await fetchPermissions();
      setShowRevokeDialog(false);
      setToRevoke(null);
    } catch (err) {
      setRevokeError(err instanceof Error ? err.message : "Failed to revoke permission");
    } finally {
      setRevoking(false);
    }
  }

  const emptyState = useMemo(() => (
    <div className="text-center py-8">
      <Users className="w-8 h-8 text-muted-foreground mx-auto mb-2" />
      <p className="text-sm text-muted-foreground mb-4">No users or groups have access yet</p>
      {isAdmin && (
        <Button onClick={() => setShowGrantDialog(true)} size="sm">
          <Plus className="w-4 h-4 mr-2" />
          Grant First Permission
        </Button>
      )}
    </div>
  ), [isAdmin]);

  if (!projectName || (loading && items.length === 0)) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="animate-spin h-8 w-8" />
          <span className="ml-2">Loading permissions...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6">
      <ProjectSubpageHeader
        title={<>Permissions</>}
        description={<>Manage user and group access for {project?.displayName || projectName}</>}
        actions={
          <>
            <Button variant="outline" onClick={handleRefresh} disabled={loading}>
              <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
              Refresh
            </Button>
            {isAdmin && (
              <Button onClick={() => setShowGrantDialog(true)}>
                <Plus className="w-4 h-4 mr-2" />
                Grant Permission
              </Button>
            )}
          </>
        }
      />

      {error && (
        <Card className="mb-6 border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <p className="text-red-700">Error: {error}</p>
          </CardContent>
        </Card>
      )}

      {!isAdmin && (
        <Card className="mb-6 border-blue-200 bg-blue-50">
          <CardContent className="pt-6 flex items-center gap-2">
            <Info className="w-4 h-4 text-blue-600" />
            <p className="text-blue-700">You have {project?.userRole} access. Only admins can grant or revoke permissions.</p>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Users className="w-5 h-5" />
            Permissions
          </CardTitle>
          <CardDescription>Users and groups with access to this project and their roles</CardDescription>
        </CardHeader>
        <CardContent>
          {items.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Subject</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Role</TableHead>
                  {isAdmin && <TableHead className="text-right">Actions</TableHead>}
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((p) => {
                  const roleConfig = ROLE_DEFINITIONS[p.role];
                  const RoleIcon = roleConfig.icon;
                  return (
                    <TableRow key={`${p.subjectType}:${p.subjectName}:${p.role}`}>
                      <TableCell className="font-medium">{p.subjectName}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                          {p.subjectType === "group" ? <Users className="w-3 h-3" /> : <UserIcon className="w-3 h-3" />}
                          {p.subjectType === "group" ? "Group" : "User"}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge className={roleConfig.color} style={{ cursor: "default" }}>
                          <RoleIcon className="w-3 h-3 mr-1" />
                          {roleConfig.label}
                        </Badge>
                      </TableCell>
                      
                      {isAdmin && (
                        <TableCell className="text-right">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                              setToRevoke(p);
                              setRevokeError(null);
                              setShowRevokeDialog(true);
                            }}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </TableCell>
                      )}
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          ) : (
            emptyState
          )}
        </CardContent>
      </Card>

      {/* Grant Permission Dialog */}
      <Dialog open={showGrantDialog} onOpenChange={setShowGrantDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Grant Permission</DialogTitle>
            <DialogDescription>Add a user or group to this project with a role</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Subject Type</Label>
              <Tabs
                value={grantForm.subjectType}
                onValueChange={(value) => {
                  if (granting) return;
                  setGrantForm((prev) => ({ ...prev, subjectType: value as SubjectType }));
                }}
              >
                <TabsList className="grid grid-cols-2 w-full">
                  <TabsTrigger value="group">Group</TabsTrigger>
                  <TabsTrigger value="user">User</TabsTrigger>
                </TabsList>
              </Tabs>
            </div>
            <div className="space-y-2">
              <Label htmlFor="subjectName">{grantForm.subjectType === "group" ? "Group" : "User"} Name</Label>
              <Input
                id="subjectName"
                placeholder={`Enter ${grantForm.subjectType} name`}
                value={grantForm.subjectName}
                onChange={(e) => setGrantForm((prev) => ({ ...prev, subjectName: e.target.value }))}
                disabled={granting}
              />
            </div>
            <div className="space-y-2">
              <Label>Role</Label>
              <div className="space-y-3">
                {Object.entries(ROLE_DEFINITIONS).map(([roleKey, roleConfig]) => {
                  const RoleIcon = roleConfig.icon;
                  const id = `role-${roleKey}`;
                  return (
                    <div key={roleKey} className="flex items-start gap-3">
                      <input
                        type="radio"
                        name="grant-role"
                        id={id}
                        className="mt-1 h-4 w-4"
                        value={roleKey}
                        checked={grantForm.role === (roleKey as PermissionRole)}
                        onChange={() => setGrantForm((prev) => ({ ...prev, role: roleKey as PermissionRole }))}
                        disabled={granting}
                      />
                      <Label htmlFor={id} className="flex-1 cursor-pointer">
                        <div className="flex items-center gap-2">
                          <RoleIcon className="w-4 h-4" />
                          <span className="font-medium">{roleConfig.label}</span>
                        </div>
                        <div className="text-sm text-muted-foreground ml-6">{roleConfig.description}</div>
                      </Label>
                    </div>
                  );
                })}
              </div>
            </div>
            {grantError && <div className="text-sm text-red-600 bg-red-50 p-2 rounded">{grantError}</div>}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowGrantDialog(false)} disabled={granting}>
              Cancel
            </Button>
            <Button onClick={handleGrant} disabled={granting}>
              {granting ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Granting...
                </>
              ) : (
                <>
                  <Plus className="w-4 h-4 mr-2" />
                  Grant Permission
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Revoke Permission Dialog */}
      <Dialog open={showRevokeDialog} onOpenChange={setShowRevokeDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke Permission</DialogTitle>
            <DialogDescription>
              Are you sure you want to revoke access for &quot;{toRevoke?.subjectName}&quot; ({toRevoke?.subjectType})?
            </DialogDescription>
          </DialogHeader>
          {revokeError && <div className="text-sm text-red-600 bg-red-50 p-2 rounded">{revokeError}</div>}
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowRevokeDialog(false)} disabled={revoking}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleRevoke} disabled={revoking}>
              {revoking ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Revoking...
                </>
              ) : (
                <>
                  <Trash2 className="w-4 h-4 mr-2" />
                  Revoke
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}


