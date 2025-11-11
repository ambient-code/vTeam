"use client";

import { useState, useCallback } from "react";
import { useGitPull, useGitPush } from "@/services/queries/use-workspace";
import { successToast, errorToast } from "@/hooks/use-toast";
import type { GitStatus } from "../lib/types";

type UseGitOperationsProps = {
  projectName: string;
  sessionName: string;
  directoryPath: string;
  remoteBranch?: string;
};

export function useGitOperations({
  projectName,
  sessionName,
  directoryPath,
  remoteBranch = "main",
}: UseGitOperationsProps) {
  const [synchronizing, setSynchronizing] = useState(false);
  const [committing, setCommitting] = useState(false);
  const [gitStatus, setGitStatus] = useState<GitStatus | null>(null);
  
  const gitPullMutation = useGitPull();
  const gitPushMutation = useGitPush();

  // Fetch git status for the current directory
  const fetchGitStatus = useCallback(async () => {
    if (!projectName || !sessionName) return;
    
    try {
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/git/status?path=${encodeURIComponent(directoryPath)}`
      );
      
      if (response.ok) {
        const data = await response.json();
        setGitStatus(data);
      }
    } catch (error) {
      console.error("Failed to fetch git status:", error);
    }
  }, [projectName, sessionName, directoryPath]);

  // Configure remote for the directory
  const configureRemote = useCallback(async (remoteUrl: string, branch: string) => {
    try {
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/git/configure-remote`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            path: directoryPath,
            remoteUrl: remoteUrl.trim(),
            branch: branch.trim() || "main",
          }),
        }
      );
      
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Failed to configure remote");
      }
      
      successToast("Remote configured successfully");
      await fetchGitStatus();
      
      return true;
    } catch (error) {
      errorToast(error instanceof Error ? error.message : "Failed to configure remote");
      return false;
    }
  }, [projectName, sessionName, directoryPath, fetchGitStatus]);

  // Pull changes from remote
  const handleGitPull = useCallback((onSuccess?: () => void) => {
    gitPullMutation.mutate(
      {
        projectName,
        sessionName,
        path: directoryPath,
        branch: remoteBranch,
      },
      {
        onSuccess: () => {
          successToast("Changes pulled successfully");
          fetchGitStatus();
          onSuccess?.();
        },
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to pull changes"),
      }
    );
  }, [gitPullMutation, projectName, sessionName, directoryPath, remoteBranch, fetchGitStatus]);

  // Push changes to remote
  const handleGitPush = useCallback((onSuccess?: () => void) => {
    const timestamp = new Date().toISOString();
    const message = `Workflow progress - ${timestamp}`;
    
    gitPushMutation.mutate(
      {
        projectName,
        sessionName,
        path: directoryPath,
        branch: remoteBranch,
        message,
      },
      {
        onSuccess: () => {
          successToast("Changes pushed successfully");
          fetchGitStatus();
          onSuccess?.();
        },
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to push changes"),
      }
    );
  }, [gitPushMutation, projectName, sessionName, directoryPath, remoteBranch, fetchGitStatus]);

  // Synchronize: pull then push
  const handleGitSynchronize = useCallback(async (onSuccess?: () => void) => {
    try {
      setSynchronizing(true);
      
      // Pull first
      await gitPullMutation.mutateAsync({
        projectName,
        sessionName,
        path: directoryPath,
        branch: remoteBranch,
      });
      
      // Then push
      const timestamp = new Date().toISOString();
      const message = `Workflow progress - ${timestamp}`;
      
      await gitPushMutation.mutateAsync({
        projectName,
        sessionName,
        path: directoryPath,
        branch: remoteBranch,
        message,
      });
      
      successToast("Changes synchronized successfully");
      fetchGitStatus();
      onSuccess?.();
    } catch (error) {
      errorToast(error instanceof Error ? error.message : "Failed to synchronize");
    } finally {
      setSynchronizing(false);
    }
  }, [gitPullMutation, gitPushMutation, projectName, sessionName, directoryPath, remoteBranch, fetchGitStatus]);

  // Commit changes without pushing
  const handleCommit = useCallback(async (commitMessage: string) => {
    if (!commitMessage.trim()) {
      errorToast("Commit message is required");
      return false;
    }

    setCommitting(true);
    try {
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/git/synchronize`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            path: directoryPath,
            message: commitMessage.trim(),
            branch: remoteBranch,
          }),
        }
      );

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to commit');
      }

      successToast('Changes committed successfully');
      fetchGitStatus();
      return true;
    } catch (error) {
      errorToast(error instanceof Error ? error.message : 'Failed to commit');
      return false;
    } finally {
      setCommitting(false);
    }
  }, [projectName, sessionName, directoryPath, remoteBranch, fetchGitStatus]);

  return {
    gitStatus,
    synchronizing,
    committing,
    fetchGitStatus,
    configureRemote,
    handleGitPull,
    handleGitPush,
    handleGitSynchronize,
    handleCommit,
    isPulling: gitPullMutation.isPending,
    isPushing: gitPushMutation.isPending,
  };
}

