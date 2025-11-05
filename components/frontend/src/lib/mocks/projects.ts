/**
 * Mock project data
 */

import type { Project, PermissionAssignment } from '@/types/api';

export const mockProjects: Project[] = [
  {
    name: 'demo-project',
    displayName: 'Demo Project',
    description: 'A demonstration project for testing',
    labels: {
      'app': 'vteam',
      'environment': 'dev',
    },
    annotations: {
      'created-by': 'demo-user',
    },
    creationTimestamp: '2025-01-15T10:00:00Z',
    status: 'active',
    isOpenShift: true,
  },
  {
    name: 'ml-training',
    displayName: 'ML Training Pipeline',
    description: 'Machine learning model training and evaluation',
    labels: {
      'app': 'vteam',
      'team': 'data-science',
    },
    annotations: {},
    creationTimestamp: '2025-01-10T08:30:00Z',
    status: 'active',
    isOpenShift: true,
  },
  {
    name: 'web-app',
    displayName: 'Web Application',
    description: 'Full-stack web application development',
    labels: {
      'app': 'vteam',
      'team': 'frontend',
    },
    annotations: {},
    creationTimestamp: '2025-01-05T14:20:00Z',
    status: 'active',
    isOpenShift: true,
  },
  {
    name: 'api-services',
    displayName: 'API Services',
    description: 'Backend API and microservices',
    labels: {
      'app': 'vteam',
      'team': 'backend',
    },
    annotations: {},
    creationTimestamp: '2025-01-01T09:00:00Z',
    status: 'active',
    isOpenShift: true,
  },
];

export const mockPermissions: Record<string, PermissionAssignment[]> = {
  'demo-project': [
    {
      subjectType: 'user',
      subjectName: 'demo-user',
      role: 'admin',
      permissions: ['read', 'write', 'delete', 'manage'],
      grantedAt: '2025-01-15T10:00:00Z',
      grantedBy: 'system',
    },
    {
      subjectType: 'group',
      subjectName: 'developers',
      role: 'edit',
      permissions: ['read', 'write'],
      memberCount: 5,
      grantedAt: '2025-01-15T10:05:00Z',
      grantedBy: 'demo-user',
    },
    {
      subjectType: 'group',
      subjectName: 'viewers',
      role: 'view',
      permissions: ['read'],
      memberCount: 12,
      grantedAt: '2025-01-15T10:10:00Z',
      grantedBy: 'demo-user',
    },
  ],
  'ml-training': [
    {
      subjectType: 'group',
      subjectName: 'data-science',
      role: 'admin',
      permissions: ['read', 'write', 'delete', 'manage'],
      memberCount: 3,
      grantedAt: '2025-01-10T08:30:00Z',
      grantedBy: 'system',
    },
  ],
  'web-app': [
    {
      subjectType: 'group',
      subjectName: 'frontend',
      role: 'admin',
      permissions: ['read', 'write', 'delete', 'manage'],
      memberCount: 7,
      grantedAt: '2025-01-05T14:20:00Z',
      grantedBy: 'system',
    },
  ],
  'api-services': [
    {
      subjectType: 'group',
      subjectName: 'backend',
      role: 'admin',
      permissions: ['read', 'write', 'delete', 'manage'],
      memberCount: 4,
      grantedAt: '2025-01-01T09:00:00Z',
      grantedBy: 'system',
    },
  ],
};

export function getMockProject(name: string): Project | undefined {
  return mockProjects.find(p => p.name === name);
}

export function getMockProjectPermissions(projectName: string): PermissionAssignment[] {
  return mockPermissions[projectName] || [];
}

