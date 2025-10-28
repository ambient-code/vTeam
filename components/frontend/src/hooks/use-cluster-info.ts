/**
 * Cluster information hook
 * Detects cluster type (OpenShift vs vanilla Kubernetes) based on project data
 */

import { useProjects } from '@/services/queries';

export type ClusterInfo = {
  isOpenShift: boolean | null; // null = not yet detected, true = OpenShift, false = vanilla k8s
  isLoading: boolean;
  isDetected: boolean;
};

/**
 * Detects whether the cluster is OpenShift or vanilla Kubernetes
 * Uses the isOpenShift flag from any project in the list
 * Returns null if cluster type hasn't been detected yet (no projects exist)
 */
export function useClusterInfo(): ClusterInfo {
  const { data: projects = [], isLoading } = useProjects();

  // If we have at least one project, we can determine cluster type
  const isDetected = projects.length > 0;
  const isOpenShift = isDetected ? (projects[0]?.isOpenShift ?? false) : null;

  return {
    isOpenShift,
    isLoading,
    isDetected,
  };
}

