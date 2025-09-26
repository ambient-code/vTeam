"""
Protocol definitions for runner-backend communication.
"""

from enum import Enum
from typing import Dict, Any, Optional, List
from pydantic import BaseModel, Field


class MessageType(str, Enum):
    """Unified message types for runner communication."""

    SYSTEM_MESSAGE = "system.message"
    AGENT_MESSAGE = "agent.message"
    USER_MESSAGE = "user.message"
    MESSAGE_PARTIAL = "message.partial"
    AGENT_RUNNING = "agent.running"
    WAITING_FOR_INPUT = "agent.waiting"


class SessionStatus(str, Enum):
    """Session status values."""

    QUEUED = "queued"
    RUNNING = "running"
    SUCCEEDED = "succeeded"
    FAILED = "failed"


class Message(BaseModel):
    """Standard message format."""

    seq: int = Field(description="Monotonic sequence number")
    type: MessageType
    timestamp: str
    payload: Any
    partial: Optional["PartialInfo"] = None


class PartialInfo(BaseModel):
    """Information for partial/fragmented messages."""

    id: str = Field(description="Unique ID for this partial set")
    index: int = Field(description="0-based index of this fragment")
    total: int = Field(description="Total number of fragments")
    data: str = Field(description="Fragment data")


class PRIntent(BaseModel):
    """PR creation intent."""

    repo_url: str
    source_branch: str
    target_branch: str
    title: str
    description: str
    changes_summary: List[str]