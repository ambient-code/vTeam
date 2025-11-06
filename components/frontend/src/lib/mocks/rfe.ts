/**
 * Mock RFE workflow data
 */

import type { 
  RFEWorkflow, 
  AgentPersona, 
  ArtifactFile,
} from '@/types/api';

export const mockAgentPersonas: AgentPersona[] = [
  {
    persona: 'olivia',
    name: 'Olivia',
    role: 'Product Owner',
    description: 'Defines product vision and requirements',
  },
  {
    persona: 'archie',
    name: 'Archie',
    role: 'Architect',
    description: 'Designs system architecture and technical approach',
  },
  {
    persona: 'stella',
    name: 'Stella',
    role: 'Staff Engineer',
    description: 'Implements technical solutions and code',
  },
  {
    persona: 'ryan',
    name: 'Ryan',
    role: 'UX Researcher',
    description: 'Conducts user research and testing',
  },
  {
    persona: 'steve',
    name: 'Steve',
    role: 'UX Designer',
    description: 'Creates user interface designs',
  },
];

export const mockRFEWorkflows: RFEWorkflow[] = [
  {
    id: 'rfe-001',
    title: 'Implement Dark Mode Support',
    description: 'Add dark mode toggle and theme support across the application',
    branchName: 'feature/dark-mode',
    currentPhase: 'implement',
    status: 'active',
    umbrellaRepo: {
      url: 'https://github.com/demo/web-app.git',
      branch: 'main',
    },
    supportingRepos: [
      {
        url: 'https://github.com/demo/design-system.git',
        branch: 'main',
      },
    ],
    workspacePath: '/workspace/rfe-001',
    parentOutcome: 'Improve user experience with theme customization',
    agentSessions: [
      {
        id: 'rfe-001-ideate-olivia',
        agentPersona: 'olivia',
        phase: 'ideate',
        status: 'completed',
        startedAt: '2025-11-01T09:00:00Z',
        completedAt: '2025-11-01T09:30:00Z',
        result: 'Identified user needs for dark mode and theme preferences',
        cost: 0.15,
      },
      {
        id: 'rfe-001-specify-archie',
        agentPersona: 'archie',
        phase: 'specify',
        status: 'completed',
        startedAt: '2025-11-01T10:00:00Z',
        completedAt: '2025-11-01T11:00:00Z',
        result: 'Designed theme architecture using CSS variables and context API',
        cost: 0.28,
      },
      {
        id: 'rfe-001-implement-stella',
        agentPersona: 'stella',
        phase: 'implement',
        status: 'running',
        startedAt: '2025-11-05T08:00:00Z',
        cost: 0.42,
      },
    ],
    artifacts: [
      {
        path: 'docs/dark-mode-requirements.md',
        name: 'Dark Mode Requirements',
        content: '# Dark Mode Requirements\n\n## User Stories\n- As a user, I want to toggle between light and dark themes\n- As a user, I want the app to remember my theme preference\n- As a user, I want the theme to match my system preferences',
        lastModified: '2025-11-01T09:30:00Z',
        size: 1024,
        agent: 'olivia',
        phase: 'ideate',
      },
      {
        path: 'docs/theme-architecture.md',
        name: 'Theme Architecture',
        content: '# Theme Architecture\n\n## Design\n- Use CSS variables for theme colors\n- Implement ThemeProvider context\n- Add theme toggle component',
        lastModified: '2025-11-01T11:00:00Z',
        size: 2048,
        agent: 'archie',
        phase: 'specify',
      },
    ],
    createdAt: '2025-11-01T09:00:00Z',
    updatedAt: '2025-11-05T08:00:00Z',
    phaseResults: {
      'ideate': {
        phase: 'ideate',
        status: 'completed',
        agents: ['olivia'],
        artifacts: ['docs/dark-mode-requirements.md'],
        summary: 'Identified user needs and defined requirements for dark mode feature',
        startedAt: '2025-11-01T09:00:00Z',
        completedAt: '2025-11-01T09:30:00Z',
      },
      'specify': {
        phase: 'specify',
        status: 'completed',
        agents: ['archie'],
        artifacts: ['docs/theme-architecture.md'],
        summary: 'Designed technical architecture for theme system',
        startedAt: '2025-11-01T10:00:00Z',
        completedAt: '2025-11-01T11:00:00Z',
      },
    },
    jiraLinks: [
      {
        path: 'VTEAM-123',
        jiraKey: 'VTEAM-123',
      },
    ],
  },
  {
    id: 'rfe-002',
    title: 'Real-time Collaboration Features',
    description: 'Add real-time collaboration capabilities using WebSockets',
    branchName: 'feature/collaboration',
    currentPhase: 'specify',
    status: 'active',
    umbrellaRepo: {
      url: 'https://github.com/demo/web-app.git',
      branch: 'main',
    },
    supportingRepos: [
      {
        url: 'https://github.com/demo/api-services.git',
        branch: 'main',
      },
    ],
    workspacePath: '/workspace/rfe-002',
    parentOutcome: 'Enable teams to work together in real-time',
    agentSessions: [
      {
        id: 'rfe-002-ideate-olivia',
        agentPersona: 'olivia',
        phase: 'ideate',
        status: 'completed',
        startedAt: '2025-11-03T14:00:00Z',
        completedAt: '2025-11-03T14:45:00Z',
        result: 'Defined collaboration scenarios and user workflows',
        cost: 0.18,
      },
      {
        id: 'rfe-002-specify-archie',
        agentPersona: 'archie',
        phase: 'specify',
        status: 'running',
        startedAt: '2025-11-05T09:00:00Z',
        cost: 0.22,
      },
    ],
    artifacts: [
      {
        path: 'docs/collaboration-requirements.md',
        name: 'Collaboration Requirements',
        content: '# Real-time Collaboration Requirements\n\n## Features\n- Live cursor positions\n- Shared editing\n- Presence indicators\n- Chat functionality',
        lastModified: '2025-11-03T14:45:00Z',
        size: 1536,
        agent: 'olivia',
        phase: 'ideate',
      },
    ],
    createdAt: '2025-11-03T14:00:00Z',
    updatedAt: '2025-11-05T09:00:00Z',
    phaseResults: {
      'ideate': {
        phase: 'ideate',
        status: 'completed',
        agents: ['olivia'],
        artifacts: ['docs/collaboration-requirements.md'],
        summary: 'Defined real-time collaboration requirements and user workflows',
        startedAt: '2025-11-03T14:00:00Z',
        completedAt: '2025-11-03T14:45:00Z',
      },
    },
    jiraLinks: [
      {
        path: 'VTEAM-124',
        jiraKey: 'VTEAM-124',
      },
    ],
  },
  {
    id: 'rfe-003',
    title: 'Performance Optimization',
    description: 'Optimize application performance and reduce load times',
    branchName: 'feature/performance',
    currentPhase: 'completed',
    status: 'completed',
    umbrellaRepo: {
      url: 'https://github.com/demo/web-app.git',
      branch: 'main',
    },
    workspacePath: '/workspace/rfe-003',
    parentOutcome: 'Improve application responsiveness and user experience',
    agentSessions: [
      {
        id: 'rfe-003-ideate-olivia',
        agentPersona: 'olivia',
        phase: 'ideate',
        status: 'completed',
        startedAt: '2025-10-28T10:00:00Z',
        completedAt: '2025-10-28T10:30:00Z',
        result: 'Identified performance bottlenecks and optimization opportunities',
        cost: 0.12,
      },
      {
        id: 'rfe-003-implement-stella',
        agentPersona: 'stella',
        phase: 'implement',
        status: 'completed',
        startedAt: '2025-10-29T09:00:00Z',
        completedAt: '2025-10-29T15:00:00Z',
        result: 'Implemented code splitting, lazy loading, and caching strategies',
        cost: 0.58,
      },
    ],
    artifacts: [
      {
        path: 'docs/performance-report.md',
        name: 'Performance Analysis Report',
        content: '# Performance Analysis\n\n## Improvements\n- Reduced bundle size by 40%\n- Improved initial load time by 60%\n- Implemented lazy loading for routes',
        lastModified: '2025-10-29T15:00:00Z',
        size: 3072,
        agent: 'stella',
        phase: 'implement',
      },
    ],
    createdAt: '2025-10-28T10:00:00Z',
    updatedAt: '2025-10-29T15:00:00Z',
    phaseResults: {
      'ideate': {
        phase: 'ideate',
        status: 'completed',
        agents: ['olivia'],
        artifacts: [],
        summary: 'Identified key performance bottlenecks',
        startedAt: '2025-10-28T10:00:00Z',
        completedAt: '2025-10-28T10:30:00Z',
      },
      'implement': {
        phase: 'implement',
        status: 'completed',
        agents: ['stella'],
        artifacts: ['docs/performance-report.md'],
        summary: 'Implemented performance optimizations',
        startedAt: '2025-10-29T09:00:00Z',
        completedAt: '2025-10-29T15:00:00Z',
      },
    },
  },
];

export function getMockRFEWorkflow(projectName: string, workflowId: string): RFEWorkflow | undefined {
  return mockRFEWorkflows.find(w => w.id === workflowId);
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export function getMockRFEWorkflowsByProject(projectName: string): RFEWorkflow[] {
  // In a real implementation, this would filter by project
  // For now, we return all workflows regardless of project
  return mockRFEWorkflows;
}

export function getMockArtifacts(projectName: string, workflowId: string): ArtifactFile[] {
  const workflow = getMockRFEWorkflow(projectName, workflowId);
  return workflow?.artifacts || [];
}

