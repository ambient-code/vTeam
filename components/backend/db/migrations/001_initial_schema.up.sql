-- 001_initial_schema.up.sql

-- Create extensions if needed
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create workflow_sessions table
CREATE TABLE IF NOT EXISTS workflow_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id TEXT NOT NULL,
    workflow_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    input_data JSONB,
    output_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Create indices on workflow_sessions table
CREATE INDEX IF NOT EXISTS workflow_sessions_project_id_idx ON workflow_sessions (project_id);
CREATE INDEX IF NOT EXISTS workflow_sessions_workflow_name_idx ON workflow_sessions (workflow_name);
CREATE INDEX IF NOT EXISTS workflow_sessions_status_idx ON workflow_sessions (status);
CREATE INDEX IF NOT EXISTS workflow_sessions_created_at_idx ON workflow_sessions (created_at);

-- Create session_messages table
CREATE TABLE IF NOT EXISTS session_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES workflow_sessions(id) ON DELETE CASCADE,
    message_type TEXT NOT NULL,
    content JSONB NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indices on session_messages table
CREATE INDEX IF NOT EXISTS session_messages_session_id_idx ON session_messages (session_id);
CREATE INDEX IF NOT EXISTS session_messages_message_type_idx ON session_messages (message_type);
CREATE INDEX IF NOT EXISTS session_messages_timestamp_idx ON session_messages (timestamp);