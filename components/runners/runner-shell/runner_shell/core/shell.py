"""
Core shell for managing runner lifecycle and message flow.
"""

import asyncio
import json
from typing import Dict, Any
from datetime import datetime

from .protocol import Message, MessageType, PartialInfo
from .transport_ws import WebSocketTransport
from .context import RunnerContext


class RunnerShell:
    """Core shell that orchestrates runner execution."""

    def __init__(
        self,
        session_id: str,
        workspace_path: str,
        websocket_url: str,
        adapter: Any,
    ):
        self.session_id = session_id
        self.workspace_path = workspace_path
        self.adapter = adapter

        # Initialize components
        self.transport = WebSocketTransport(websocket_url)
        self.sink = None
        self.context = RunnerContext(
            session_id=session_id,
            workspace_path=workspace_path,
        )

        self.running = False
        self.message_seq = 0

    async def start(self):
        """Start the runner shell."""
        self.running = True

        # Connect transport
        await self.transport.connect()
        # Forward incoming WS messages to adapter
        self.transport.set_receive_handler(self.handle_incoming_message)

        # Send session started as a system message
        await self._send_message(
            MessageType.SYSTEM_MESSAGE,
            "session.started"
        )

        try:
            # Initialize adapter with context
            await self.adapter.initialize(self.context)

            # Run adapter main loop
            result = await self.adapter.run()

            # Send completion as a system message
            await self._send_message(
                MessageType.SYSTEM_MESSAGE,
                "session.completed"
            )

        except Exception as e:
            # Send error as a system message
            await self._send_message(
                MessageType.SYSTEM_MESSAGE,
                "session.failed"
            )
            raise
        finally:
            await self.stop()

    async def stop(self):
        """Stop the runner shell."""
        self.running = False
        await self.transport.disconnect()
        # No-op; backend handles persistence

    async def _send_message(self, msg_type: MessageType, payload: Dict[str, Any], partial: PartialInfo | None = None):
        """Send a message through transport and persist to sink."""
        self.message_seq += 1

        message = Message(
            seq=self.message_seq,
            type=msg_type,
            timestamp=datetime.utcnow().isoformat(),
            payload=payload,
            partial=partial,
        )

        # Send via transport
        await self.transport.send(message.dict())

        # No-op persistence; messages are persisted by backend

    async def handle_incoming_message(self, message: Dict[str, Any]):
        """Handle messages from backend."""
        # Forward to adapter if it has a handler
        if hasattr(self.adapter, 'handle_message'):
            await self.adapter.handle_message(message)