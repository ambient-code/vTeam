'use client';

import { useCallback, useMemo, useState } from 'react';
import { Users, User as UserIcon, Plus, RefreshCw, Loader2, Trash2, Info } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { DestructiveConfirmationDialog } from '@/components/confirmation-dialog';

import { useProjectPermissions, useAddProjectPermission, useRemoveProjectPermission } from '@/services/queries';
import { successToast, errorToast } from '@/hooks/use-toast';
import type { PermissionRole, SubjectType } from '@/types/project';
import { ROLE_DEFINITIONS } from '@/lib/role-colors';

type GrantPermissionForm = {
  subjectType: SubjectType;
  subjectName: string;
  role: PermissionRole;
};

type SharingSectionProps = {
  projectName: string;
};

export function SharingSection({ projectName }: SharingSectionProps) {
  const { data: permissions = [], isLoading, refetch } = useProjectPermissions(projectName);
  const addPermissionMutation = useAddProjectPermission();
  const removePermissionMutation = useRemoveProjectPermission();

  const [showGrantDialog, setShowGrantDialog] = useState(false);
  const [grantForm, setGrantForm] = useState<GrantPermissionForm>({
    subjectType: 'group',
    subjectName: '',
    role: 'view',
  });
  const [grantError, setGrantError] = useState<string | null>(null);
  const userRole: PermissionRole | undefined = undefined;

  const [showRevokeDialog, setShowRevokeDialog] = useState(false);
  const [toRevoke, setToRevoke] = useState<{ subjectType: SubjectType; subjectName: string; role: PermissionRole } | null>(null);

  const isAdmin = userRole === 'admin' || userRole === undefined;

  const handleGrant = useCallback(() => {
    if (!grantForm.subjectName.trim()) {
      setGrantError(`${grantForm.subjectType === 'group' ? 'Group' : 'User'} name is required`);
      return;
    }

    const key = `${grantForm.subjectType}:${grantForm.subjectName}`.toLowerCase();
    if (permissions.some((i) => `${i.subjectType}:${i.subjectName}`.toLowerCase() === key)) {
      setGrantError('This subject already has access to the workspace');
      return;
    }

    setGrantError(null);
    addPermissionMutation.mutate(
      {
        projectName,
        permission: {
          subjectType: grantForm.subjectType,
          subjectName: grantForm.subjectName,
          role: grantForm.role,
        },
      },
      {
        onSuccess: () => {
          successToast(`Permission granted to ${grantForm.subjectName} successfully`);
          setShowGrantDialog(false);
          setGrantForm({ subjectType: 'group', subjectName: '', role: 'view' });
        },
        onError: (error) => {
          const message = error instanceof Error ? error.message : 'Failed to grant permission';
          setGrantError(message);
          errorToast(message);
        },
      }
    );
  }, [grantForm, permissions, projectName, addPermissionMutation]);

  const handleRevoke = useCallback(() => {
    if (!toRevoke) return;

    removePermissionMutation.mutate(
      {
        projectName,
        subjectType: toRevoke.subjectType,
        subjectName: toRevoke.subjectName,
      },
      {
        onSuccess: () => {
          successToast(`Permission revoked from ${toRevoke.subjectName} successfully`);
          setShowRevokeDialog(false);
          setToRevoke(null);
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to revoke permission');
        },
      }
    );
  }, [toRevoke, projectName, removePermissionMutation]);

  const emptyState = useMemo(
    () => (
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
    ),
    [isAdmin]
  );

  return (
    <>
      {!isAdmin && (
        <Card className="mb-6 border-status-info-border bg-status-info dark:border-status-info-border dark:bg-status-info">
          <CardContent className="pt-6 flex items-center gap-2">
            <Info className="w-4 h-4 text-status-info-foreground" />
            <p className="text-status-info-foreground">
              You have {userRole || 'view'} access. Only admins can grant or revoke permissions.
            </p>
          </CardContent>
        </Card>
      )}

      <Card className="flex-1">
        <CardHeader>
          <div className="flex items-start justify-between">
            <div>
              <CardTitle>
                Sharing
              </CardTitle>
              <CardDescription>Users and groups with access to this workspace and their roles</CardDescription>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => refetch()} disabled={isLoading}>
                <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
              {isAdmin && (
                <Button onClick={() => setShowGrantDialog(true)}>
                  <Plus className="w-4 h-4 mr-2" />
                  Grant Permission
                </Button>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {permissions.length > 0 ? (
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
                {permissions.map((p) => {
                  const roleConfig = ROLE_DEFINITIONS[p.role];
                  const RoleIcon = roleConfig.icon;
                  const isRevokingThis =
                    removePermissionMutation.isPending &&
                    removePermissionMutation.variables?.subjectName === p.subjectName &&
                    removePermissionMutation.variables?.subjectType === p.subjectType;

                  return (
                    <TableRow key={`${p.subjectType}:${p.subjectName}:${p.role}`}>
                      <TableCell className="font-medium">{p.subjectName}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                          {p.subjectType === 'group' ? (
                            <Users className="w-3 h-3" />
                          ) : (
                            <UserIcon className="w-3 h-3" />
                          )}
                          {p.subjectType === 'group' ? 'Group' : 'User'}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge className={roleConfig.color} style={{ cursor: 'default' }}>
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
                              setShowRevokeDialog(true);
                            }}
                            disabled={isRevokingThis}
                          >
                            {isRevokingThis ? (
                              <Loader2 className="w-4 h-4 animate-spin" />
                            ) : (
                              <Trash2 className="w-4 h-4" />
                            )}
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
            <DialogDescription>Add a user or group to this workspace with a role</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Subject Type</Label>
              <Tabs
                value={grantForm.subjectType}
                onValueChange={(value) => {
                  if (addPermissionMutation.isPending) return;
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
              <Label htmlFor="subjectName">
                {grantForm.subjectType === 'group' ? 'Group' : 'User'} Name
              </Label>
              <Input
                id="subjectName"
                placeholder={`Enter ${grantForm.subjectType} name`}
                value={grantForm.subjectName}
                onChange={(e) => setGrantForm((prev) => ({ ...prev, subjectName: e.target.value }))}
                disabled={addPermissionMutation.isPending}
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
                        disabled={addPermissionMutation.isPending}
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
            {grantError && <div className="text-sm text-red-600 dark:text-red-400 bg-red-50 p-2 rounded">{grantError}</div>}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowGrantDialog(false)}
              disabled={addPermissionMutation.isPending}
            >
              Cancel
            </Button>
            <Button onClick={handleGrant} disabled={addPermissionMutation.isPending}>
              {addPermissionMutation.isPending ? (
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
      <DestructiveConfirmationDialog
        open={showRevokeDialog}
        onOpenChange={setShowRevokeDialog}
        onConfirm={handleRevoke}
        title="Revoke Permission"
        description={`Are you sure you want to revoke access for "${toRevoke?.subjectName}" (${toRevoke?.subjectType})? They will immediately lose access to this workspace.`}
        confirmText="Revoke"
        loading={removePermissionMutation.isPending}
      />
    </>
  );
}

