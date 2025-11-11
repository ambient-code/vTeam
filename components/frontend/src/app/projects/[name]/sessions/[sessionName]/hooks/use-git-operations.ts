"use client";

import { useState, useCallback } from "react";
import { 
  useGitPull, 
  useGitPush, 
  useGitStatus,
  useConfigureGitRemote,
  useSynchronizeGit 
} from "@/services/queries/use-workspace";
import { successToast, errorToast } from "@/hooks/use-toast";

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
  
  const gitPullMutation = useGitPull();
  const gitPushMutation = useGitPush();
  const configureRemoteMutation = useConfigureGitRemote();
  const synchronizeGitMutation = useSynchronizeGit();
  
  // Use React Query for git status
  const { data: gitStatus, refetch: fetchGitStatus } = useGitStatus(
    projectName,
    sessionName,
    directoryPath,
    { enabled: !!projectName && !!sessionName && !!directoryPath }
  );

  // Configure remote for the directory
  const configureRemote = useCallback(async (remoteUrl: string, branch: string) => {
    try {
      await configureRemoteMutation.mutateAsync({
        projectName,
        sessionName,
        path: directoryPath,
        remoteUrl: remoteUrl.trim(),
        branch: branch.trim() || "main",
      });
      
      successToast("Remote configured successfully");
      await fetchGitStatus();
      
      return true;
    } catch (error) {
      console.error("Failed to configure remote:", error);
      errorToast(error instanceof Error ? error.message : "Failed to configure remote");
      return false;
    }
  }, [projectName, sessionName, directoryPath, configureRemoteMutation, fetchGitStatus]);

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

    try {
      await synchronizeGitMutation.mutateAsync({
        projectName,
        sessionName,
        path: directoryPath,
        message: commitMessage.trim(),
        branch: remoteBranch,
      });

      successToast('Changes committed successfully');
      fetchGitStatus();
      return true;
    } catch (error) {
      console.error("Failed to commit:", error);
      errorToast(error instanceof Error ? error.message : 'Failed to commit');
      return false;
    }
  }, [projectName, sessionName, directoryPath, remoteBranch, synchronizeGitMutation, fetchGitStatus]);

  return {
    gitStatus,
    synchronizing,
    committing: synchronizeGitMutation.isPending,
    fetchGitStatus,
    configureRemote,
    handleGitPull,
    handleGitPush,
    handleGitSynchronize,
    handleCommit,
    isPulling: gitPullMutation.isPending,
    isPushing: gitPushMutation.isPending,
    isConfiguringRemote: configureRemoteMutation.isPending,
  };
}

