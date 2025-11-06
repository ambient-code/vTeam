/**
 * Mock agentic session data
 */

import type { 
  AgenticSession, 
  Message,
  UserMessage,
  AgentMessage,
  ResultMessage,
  TextBlock,
} from '@/types/api';

export const mockSessions: AgenticSession[] = [
  {
    metadata: {
      name: 'session-demo-1',
      namespace: 'demo-project',
      creationTimestamp: '2025-11-05T10:00:00Z',
      uid: 'abc123-def456-ghi789',
      labels: {
        'project': 'demo-project',
        'user': 'demo-user',
      },
    },
    spec: {
      prompt: 'Create a REST API for user authentication',
      llmSettings: {
        model: 'claude-sonnet-4.5',
        temperature: 0.7,
        maxTokens: 4096,
      },
      timeout: 3600,
      displayName: 'User Authentication API',
      project: 'demo-project',
      interactive: true,
      repos: [
        {
          input: {
            url: 'https://github.com/demo/api-services.git',
            branch: 'main',
          },
        },
      ],
      mainRepoIndex: 0,
    },
    status: {
      phase: 'Completed',
      message: 'Session completed successfully',
      startTime: '2025-11-05T10:00:00Z',
      completionTime: '2025-11-05T10:45:00Z',
      jobName: 'session-demo-1-job',
      stateDir: '/workspace/sessions/session-demo-1',
      subtype: 'code-generation',
      is_error: false,
      num_turns: 12,
      session_id: 'session-demo-1',
      total_cost_usd: 0.45,
      usage: {
        input_tokens: 5420,
        output_tokens: 3200,
      },
      result: 'Successfully created REST API with authentication endpoints',
    },
  },
  {
    metadata: {
      name: 'session-demo-2',
      namespace: 'demo-project',
      creationTimestamp: '2025-11-05T11:00:00Z',
      uid: 'xyz789-uvw456-rst123',
      labels: {
        'project': 'demo-project',
        'user': 'demo-user',
      },
    },
    spec: {
      prompt: 'Refactor the database schema to improve query performance',
      llmSettings: {
        model: 'claude-sonnet-4.5',
        temperature: 0.5,
        maxTokens: 8192,
      },
      timeout: 7200,
      displayName: 'Database Schema Refactoring',
      project: 'demo-project',
      interactive: false,
      repos: [
        {
          input: {
            url: 'https://github.com/demo/api-services.git',
            branch: 'main',
          },
        },
      ],
      mainRepoIndex: 0,
    },
    status: {
      phase: 'Running',
      message: 'Analyzing current schema...',
      startTime: '2025-11-05T11:00:00Z',
      jobName: 'session-demo-2-job',
      stateDir: '/workspace/sessions/session-demo-2',
      subtype: 'refactoring',
      is_error: false,
      num_turns: 5,
      session_id: 'session-demo-2',
      total_cost_usd: 0.18,
      usage: {
        input_tokens: 2100,
        output_tokens: 1500,
      },
      result: null,
    },
  },
  {
    metadata: {
      name: 'session-ml-1',
      namespace: 'ml-training',
      creationTimestamp: '2025-11-04T15:30:00Z',
      uid: 'mno345-pqr678-stu901',
      labels: {
        'project': 'ml-training',
        'user': 'data-scientist',
      },
    },
    spec: {
      prompt: 'Implement feature engineering pipeline for customer churn prediction',
      llmSettings: {
        model: 'claude-sonnet-4.5',
        temperature: 0.6,
        maxTokens: 4096,
      },
      timeout: 3600,
      displayName: 'Feature Engineering Pipeline',
      project: 'ml-training',
      interactive: true,
    },
    status: {
      phase: 'Completed',
      message: 'Pipeline implementation completed',
      startTime: '2025-11-04T15:30:00Z',
      completionTime: '2025-11-04T16:20:00Z',
      jobName: 'session-ml-1-job',
      stateDir: '/workspace/sessions/session-ml-1',
      subtype: 'data-science',
      is_error: false,
      num_turns: 18,
      session_id: 'session-ml-1',
      total_cost_usd: 0.62,
      usage: {
        input_tokens: 7200,
        output_tokens: 4100,
      },
      result: 'Created feature engineering pipeline with 15 derived features',
    },
  },
];

export const mockSessionMessages: Record<string, Message[]> = {
  'session-demo-1': [
    {
      type: 'user_message',
      content: 'Create a REST API for user authentication',
      timestamp: '2025-11-05T10:00:00Z',
    } as UserMessage,
    {
      type: 'agent_message',
      content: {
        type: 'text_block',
        text: 'I\'ll help you create a REST API for user authentication. Let me start by examining the existing codebase structure.',
      } as TextBlock,
      model: 'claude-sonnet-4.5',
      timestamp: '2025-11-05T10:00:05Z',
    } as AgentMessage,
    {
      type: 'agent_message',
      content: {
        type: 'text_block',
        text: 'I\'ve created the authentication API with the following endpoints:\n\n- POST /auth/register - User registration\n- POST /auth/login - User login\n- POST /auth/logout - User logout\n- POST /auth/refresh - Refresh access token\n- GET /auth/me - Get current user\n\nThe implementation includes:\n- JWT token-based authentication\n- Password hashing with bcrypt\n- Input validation\n- Error handling\n- Rate limiting',
      } as TextBlock,
      model: 'claude-sonnet-4.5',
      timestamp: '2025-11-05T10:40:00Z',
    } as AgentMessage,
    {
      type: 'result_message',
      subtype: 'success',
      duration_ms: 2700000,
      duration_api_ms: 2400000,
      is_error: false,
      num_turns: 12,
      session_id: 'session-demo-1',
      total_cost_usd: 0.45,
      usage: {
        input_tokens: 5420,
        output_tokens: 3200,
      },
      result: 'Successfully created REST API with authentication endpoints',
      timestamp: '2025-11-05T10:45:00Z',
    } as ResultMessage,
  ],
  'session-demo-2': [
    {
      type: 'user_message',
      content: 'Refactor the database schema to improve query performance',
      timestamp: '2025-11-05T11:00:00Z',
    } as UserMessage,
    {
      type: 'agent_message',
      content: {
        type: 'text_block',
        text: 'I\'ll analyze the current database schema and identify optimization opportunities. Let me start by examining the existing tables and their relationships.',
      } as TextBlock,
      model: 'claude-sonnet-4.5',
      timestamp: '2025-11-05T11:00:05Z',
    } as AgentMessage,
  ],
};

export function getMockSession(projectName: string, sessionName: string): AgenticSession | undefined {
  return mockSessions.find(
    s => s.metadata.namespace === projectName && s.metadata.name === sessionName
  );
}

export function getMockSessionsByProject(projectName: string): AgenticSession[] {
  return mockSessions.filter(s => s.metadata.namespace === projectName);
}

export function getMockSessionMessages(sessionName: string): Message[] {
  return mockSessionMessages[sessionName] || [];
}


