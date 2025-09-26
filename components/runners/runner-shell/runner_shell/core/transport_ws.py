"""
WebSocket transport for bidirectional communication with backend.
"""

import asyncio
import json
import logging
import os
from typing import Optional, Dict, Any, Callable

import websockets
from websockets.client import WebSocketClientProtocol


logger = logging.getLogger(__name__)


class WebSocketTransport:
    """WebSocket transport implementation."""

    def __init__(self, url: str, reconnect_interval: int = 5):
        self.url = url
        self.reconnect_interval = reconnect_interval
        self.websocket: Optional[WebSocketClientProtocol] = None
        self.running = False
        self.receive_handler: Optional[Callable] = None
        self._recv_task: Optional[asyncio.Task] = None

    async def connect(self):
        """Connect to WebSocket endpoint."""
        try:
            # Forward Authorization header if BOT_TOKEN (runner SA token) is present
            headers: Dict[str, str] = {}
            token = (os.getenv("BOT_TOKEN") or os.getenv("RUNNER_TOKEN") or "").strip()
            if token:
                headers["Authorization"] = f"Bearer {token}"

            # Some websockets versions use `extra_headers`, others use `additional_headers`.
            # Pass headers as list of tuples for broad compatibility.
            header_items = [(k, v) for k, v in headers.items()]
            try:
                self.websocket = await websockets.connect(self.url, extra_headers=header_items)
            except TypeError:
                # Fallback for newer versions
                self.websocket = await websockets.connect(self.url, additional_headers=header_items)
            self.running = True
            logger.info(f"Connected to WebSocket: {self.url}")

            # Start receive loop only once
            if self._recv_task is None or self._recv_task.done():
                self._recv_task = asyncio.create_task(self._receive_loop())

        except websockets.exceptions.InvalidStatusCode as e:
            status = getattr(e, "status_code", None)
            logger.error(
                f"Failed to connect to WebSocket: HTTP {status if status is not None else 'unknown'}"
            )
            # Surface a clearer hint when auth is likely missing
            if status == 401:
                has_token = bool((os.getenv("BOT_TOKEN") or os.getenv("RUNNER_TOKEN") or "").strip())
                if not has_token:
                    logger.error(
                        "No BOT_TOKEN present; backend project routes require Authorization."
                    )
            raise
        except Exception as e:
            logger.error(f"Failed to connect to WebSocket: {e}")
            raise

    async def disconnect(self):
        """Disconnect from WebSocket."""
        self.running = False
        if self.websocket:
            await self.websocket.close()
            self.websocket = None
        # Cancel receive loop if running
        if self._recv_task and not self._recv_task.done():
            self._recv_task.cancel()
            try:
                await self._recv_task
            except Exception:
                pass
            finally:
                self._recv_task = None

    async def send(self, message: Dict[str, Any]):
        """Send message through WebSocket."""
        if not self.websocket:
            raise RuntimeError("WebSocket not connected")

        try:
            data = json.dumps(message)
            await self.websocket.send(data)
            logger.debug(f"Sent message: {message.get('type')}")

        except Exception as e:
            logger.error(f"Failed to send message: {e}")
            raise

    async def _receive_loop(self):
        """Receive messages from WebSocket."""
        while self.running:
            try:
                if not self.websocket:
                    await asyncio.sleep(self.reconnect_interval)
                    continue

                message = await self.websocket.recv()
                data = json.loads(message)
                logger.debug(f"Received message: {data.get('type')}")

                if self.receive_handler:
                    await self.receive_handler(data)

            except websockets.exceptions.ConnectionClosed:
                logger.warning("WebSocket connection closed")
                await self._reconnect()

            except Exception as e:
                logger.error(f"Error in receive loop: {e}")

    async def _reconnect(self):
        """Attempt to reconnect to WebSocket."""
        if not self.running:
            return

        logger.info("Attempting to reconnect...")
        self.websocket = None

        while self.running:
            try:
                # Re-establish connection; guarded against spawning a second recv loop
                await self.connect()
                break
            except Exception as e:
                logger.error(f"Reconnection failed: {e}")
                await asyncio.sleep(self.reconnect_interval)

    def set_receive_handler(self, handler: Callable):
        """Set handler for received messages."""
        self.receive_handler = handler