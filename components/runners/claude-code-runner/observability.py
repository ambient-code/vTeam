"""
Observability manager for Claude Code runner - SDK v3 simplified pattern.

Provides Langfuse LLM observability for Claude sessions:
- propagate_attributes() context manager sets user_id/session_id/tags in OpenTelemetry context
- update_current_trace() sets trace-level name, input, output, and metadata
- Generations and tool spans are tracked as direct children of the trace
- Security features (secret sanitization, timeouts)

SDK v3 CORRECT pattern:
1. propagate_attributes() context sets user_id/session_id/tags in OpenTelemetry context
2. start_as_current_span() creates root span to establish the trace (REQUIRED!)
3. update_current_trace() sets trace-level metadata (visible in UI)
4. Generations and tool spans created via start_as_current_generation() and start_as_current_span()
5. All child observations automatically inherit user_id/session_id/tags from propagate_attributes()
6. Usage data tracked in trace metadata for visibility
7. Summary generation with Anthropic-specific usage_details enables auto-cost calculation

Anthropic-specific cost tracking:
- Claude SDK only provides session-level usage in ResultMessage (no per-turn usage)
- We create a "session_summary" generation at the end with all usage data
- usage_details fields (input_tokens, output_tokens, cache_read_input_tokens, cache_creation_input_tokens)
- Langfuse automatically calculates costs using configured Claude model pricing
- Reference: https://langfuse.com/docs/observability/features/token-and-cost-tracking
"""

import os
import logging
from typing import Any
from urllib.parse import urlparse

from security_utils import sanitize_exception_message, with_sync_timeout

# Langfuse will be imported lazily only when enabled

class ObservabilityManager:
    """Manages Langfuse observability for Claude sessions.

    Handles initialization, event tracking, and cleanup for Langfuse with
    security features (secret sanitization, timeouts).

    SDK v3 CORRECT approach:
    - propagate_attributes() context manager sets user_id/session_id/tags in OTel context
    - start_as_current_span() creates root span to establish the trace (REQUIRED!)
    - update_current_trace() sets trace-level metadata
    - Generations and tool spans are created as children of the root span
    - All child observations automatically inherit user_id/session_id/tags
    - Usage data tracked in trace metadata for visibility
    - Summary generation with Anthropic-specific usage_details enables auto-cost calculation

    Anthropic cost tracking:
    - Claude SDK only provides session-level usage (no per-turn data)
    - finalize() creates "session_summary" generation with usage_details
    - Langfuse auto-calculates costs using configured Claude model pricing
    """

    def __init__(self, session_id: str, user_id: str, user_name: str):
        """Initialize observability manager.

        Args:
            session_id: Unique session identifier
            user_id: Sanitized user ID (for user tracking in traces)
            user_name: Sanitized user name (for user tracking in traces)
        """
        self.session_id = session_id
        self.user_id = user_id
        self.user_name = user_name

        # Langfuse state
        self.langfuse_client = None
        self._propagate_ctx = None  # propagate_attributes() context manager
        self._root_span_ctx = None  # Root span context manager
        self._langfuse_tool_spans: dict[str, Any] = {}

        # Track configured model for usage_details in summary generation
        self.configured_model = None

    async def initialize(self, prompt: str, namespace: str) -> bool:
        """Initialize Langfuse observability.

        Args:
            prompt: Initial prompt for the session
            namespace: Kubernetes namespace for the session

        Returns:
            True if Langfuse initialized successfully
        """
        return await self._init_langfuse(prompt, namespace)

    async def _init_langfuse(self, prompt: str, namespace: str) -> bool:
        """Initialize Langfuse observability with security checks.

        Sets up propagate_attributes() context and updates trace metadata.

        Args:
            prompt: Initial prompt for the session
            namespace: Kubernetes namespace for the session

        Returns:
            True if Langfuse initialized successfully
        """
        langfuse_enabled = os.getenv("LANGFUSE_ENABLED", "").strip().lower() in (
            "1",
            "true",
            "yes",
        )
        if not langfuse_enabled:
            return False

        # Import langfuse only when it's actually needed
        try:
            from langfuse import Langfuse
        except ImportError:
            logging.debug("Langfuse not available - continuing without LLM observability")
            return False

        # Check if required Langfuse keys are present
        public_key = os.getenv("LANGFUSE_PUBLIC_KEY", "").strip()
        secret_key = os.getenv("LANGFUSE_SECRET_KEY", "").strip()
        host = os.getenv("LANGFUSE_HOST", "").strip()

        if not public_key or not secret_key:
            logging.warning(
                "LANGFUSE_ENABLED is true but LANGFUSE_PUBLIC_KEY or LANGFUSE_SECRET_KEY is missing. "
                "Langfuse observability will be disabled for this session. "
                "To enable Langfuse, platform admin must create the 'ambient-admin-langfuse-secret' secret "
                "in the operator's namespace with all LANGFUSE_* keys: "
                "LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY, LANGFUSE_HOST, LANGFUSE_ENABLED. "
                "See e2e/langfuse/README.md for setup instructions."
            )
            return False

        if not host:
            logging.warning(
                "LANGFUSE_HOST is missing. Langfuse observability will be disabled for this session. "
                "Add LANGFUSE_HOST to the 'ambient-admin-langfuse-secret' secret "
                "(e.g., LANGFUSE_HOST=http://langfuse-web.langfuse.svc.cluster.local:3000)."
            )
            return False

        # Validate LANGFUSE_HOST format to prevent malformed URLs
        try:
            parsed = urlparse(host)
            # URL must have a scheme (http/https) and a valid network location (hostname)
            if not parsed.scheme or not parsed.netloc:
                logging.warning(
                    f"LANGFUSE_HOST has invalid format (missing scheme or hostname): {host}. "
                    "Expected format: http://hostname:port or https://hostname:port. "
                    "Langfuse observability will be disabled for this session."
                )
                return False
            # Validate scheme is http or https
            if parsed.scheme not in ("http", "https"):
                logging.warning(
                    f"LANGFUSE_HOST has unsupported scheme '{parsed.scheme}'. "
                    "Only http and https are supported. "
                    "Langfuse observability will be disabled for this session."
                )
                return False
        except Exception as e:
            logging.warning(
                f"Failed to parse LANGFUSE_HOST URL '{host}': {e}. "
                "Langfuse observability will be disabled for this session."
            )
            return False

        try:
            # Import propagate_attributes for SDK v3 pattern
            from langfuse import propagate_attributes

            # Initialize Langfuse client
            self.langfuse_client = Langfuse(
                public_key=public_key, secret_key=secret_key, host=host
            )

            trace_tags = ["claude-code", f"namespace:{namespace}"]

            # SDK v3 CORRECT pattern:
            # 1. Enter propagate_attributes() context (sets user_id/session_id/tags in OTel context)
            # 2. Create a root span/observation to establish the trace
            # 3. Update the trace with metadata
            # 4. All child generations/tool spans automatically inherit user_id/session_id/tags

            # Step 1: Enter propagate_attributes context (WITHOUT metadata - keep under 200 chars)
            # Only session-level attributes that apply to ALL observations
            self._propagate_ctx = propagate_attributes(
                user_id=self.user_id,
                session_id=self.session_id,
                tags=trace_tags
            )
            self._propagate_ctx.__enter__()

            # Step 2: Create root span to establish the trace
            # This is REQUIRED for update_current_trace() to work
            self._root_span_ctx = self.langfuse_client.start_as_current_span(
                name="claude_agent_session",
                input={"prompt": prompt[:1000] if len(prompt) > 1000 else prompt},
                metadata={"namespace": namespace, "user_name": self.user_name}
            )
            self._root_span_ctx.__enter__()

            # Step 3: Update trace-level attributes
            # This sets metadata visible at the trace level in Langfuse UI
            self.langfuse_client.update_current_trace(
                name="claude_agent_session",
                metadata={"namespace": namespace, "user_name": self.user_name}
            )

            if self.user_id:
                logging.info(
                    f"Langfuse: Tracking session for user {self.user_name} ({self.user_id})"
                )

            logging.info(
                f"Langfuse: Trace initialized with propagate_attributes - "
                f"user_id={self.user_id}, session_id={self.session_id}, tags={trace_tags}"
            )
            logging.info("Langfuse tracing enabled for session")
            return True

        except Exception as e:
            # Sanitize error message to prevent API key and host leakage
            # NEVER log exception object - only sanitized message string
            secrets = {
                "public_key": public_key,
                "secret_key": secret_key,
                "host": host,
            }
            error_msg = sanitize_exception_message(e, secrets)

            # Log sanitized warning without exception object or traceback
            logging.warning(
                f"Langfuse initialization failed: {error_msg}. "
                f"Observability will be disabled for this session. "
                f"Check that your Langfuse keys are valid and the LANGFUSE_HOST is reachable."
            )
            logging.debug(f"Langfuse initialization error type: {type(e).__name__}")

            # Cleanup any partially initialized state
            if self._root_span_ctx:
                try:
                    self._root_span_ctx.__exit__(None, None, None)
                except Exception:
                    pass
            if self._propagate_ctx:
                try:
                    self._propagate_ctx.__exit__(None, None, None)
                except Exception:
                    pass

            # Continue without Langfuse - don't fail the session
            self.langfuse_client = None
            self._propagate_ctx = None
            self._root_span_ctx = None
            return False

    def track_generation(self, message: Any, model: str, turn_count: int) -> None:
        """Track Claude generation in Langfuse using SDK v3 start_as_current_generation().

        User_id, session_id, and tags are automatically inherited via propagate_attributes().

        NOTE: AssistantMessage does NOT contain usage data. Usage tracking happens in finalize()
        when we receive the ResultMessage with session-level usage metrics.

        Args:
            message: AssistantMessage from Claude SDK
            model: Model name (e.g., "claude-3-5-sonnet-20241022")
            turn_count: Current turn number
        """
        if not self.langfuse_client:
            logging.debug(f"Langfuse: Skipping generation tracking - client not initialized")
            return

        # Store configured model for use in summary generation
        if not self.configured_model:
            self.configured_model = model
            logging.debug(f"Langfuse: Captured configured model: {model}")

        logging.debug(f"Langfuse: track_generation called for turn {turn_count}, model={model}")
        try:
            # Import here to avoid circular dependency
            from claude_agent_sdk import TextBlock

            # Extract text content for generation
            text_content = []
            message_content = getattr(message, "content", []) or []
            logging.debug(f"Langfuse: Processing message with {len(message_content)} content blocks")
            for blk in message_content:
                if isinstance(blk, TextBlock):
                    text_content.append(getattr(blk, "text", ""))
                else:
                    logging.debug(f"Langfuse: Skipping non-TextBlock: {type(blk).__name__}")

            if not text_content:
                logging.debug(f"Langfuse: No text content found in message, skipping generation tracking")
                return

            output_text = "\n".join(text_content)
            logging.debug(f"Langfuse: Extracted {len(output_text)} chars of output text")

            # SDK v3 pattern: Use client.start_as_current_generation()
            # This creates a generation as a direct child of the trace
            # User_id/session_id/tags automatically inherited via propagate_attributes()
            logging.debug(f"Langfuse: Creating generation for turn {turn_count}")

            gen_ctx = self.langfuse_client.start_as_current_generation(
                name=f"claude_response_turn_{turn_count}",
                input=[{"role": "user", "content": f"Turn {turn_count}"}],
                model=model,
                metadata={"turn": turn_count}
            )
            generation = gen_ctx.__enter__()

            # Update with output (usage will be added in finalize when we get ResultMessage)
            generation.update(output=output_text)

            # End the generation
            gen_ctx.__exit__(None, None, None)

            logging.info(f"Langfuse: Tracked generation for turn {turn_count} with {len(output_text)} chars (usage pending)")
        except Exception as e:
            logging.error(f"Langfuse: Failed to create generation: {e}", exc_info=True)

    def track_tool_use(self, tool_name: str, tool_id: str, tool_input: dict) -> None:
        """Track tool decision in Langfuse using SDK v3 start_as_current_span().

        User_id, session_id, and tags are automatically inherited via propagate_attributes().

        Args:
            tool_name: Name of the tool being used
            tool_id: Unique tool use ID
            tool_input: Tool input parameters
        """
        if self.langfuse_client:
            try:
                # SDK v3 pattern: Use client.start_as_current_span()
                # This creates a span as a direct child of the trace
                # User_id/session_id/tags automatically inherited via propagate_attributes()
                # Store context manager to update with result later (in track_tool_result)
                tool_span_ctx = self.langfuse_client.start_as_current_span(
                    name=f"tool_{tool_name}",
                    input=tool_input,
                    metadata={
                        "tool_id": tool_id,
                        "tool_name": tool_name,
                    }
                )
                # Enter context and store both context manager and span object
                tool_span = tool_span_ctx.__enter__()
                self._langfuse_tool_spans[tool_id] = (tool_span_ctx, tool_span)
            except Exception as e:
                logging.debug(f"Failed to create Langfuse tool span: {e}")

    def track_tool_result(self, tool_use_id: str, content: Any, is_error: bool) -> None:
        """Track tool result in Langfuse.

        Args:
            tool_use_id: Tool use ID from track_tool_use
            content: Tool result content
            is_error: Whether the tool execution failed
        """
        # Update Langfuse tool span with result
        if tool_use_id in self._langfuse_tool_spans:
            try:
                tool_span_ctx, tool_span = self._langfuse_tool_spans[tool_use_id]
                # Truncate long results with indicator
                if content:
                    result_text = str(content)
                    if len(result_text) > 500:
                        result_text = result_text[:500] + "...[truncated]"
                else:
                    result_text = "No output"

                # SDK v3 pattern: Update span then exit context manager
                tool_span.update(
                    output={"result": result_text},
                    level="ERROR" if is_error else "DEFAULT",
                    metadata={"is_error": is_error or False}
                )

                # Exit the context manager (ends the span)
                tool_span_ctx.__exit__(None, None, None)
                del self._langfuse_tool_spans[tool_use_id]
            except Exception as e:
                logging.debug(f"Failed to update Langfuse tool span: {e}")

    async def finalize(self, result_payload: dict | None) -> None:
        """Finalize trace and flush observability data (success path).

        Updates trace with final metrics and exits propagate_attributes context.

        Args:
            result_payload: ResultMessage payload with final metrics (may be None)
                Expected structure: {
                    "num_turns": int,
                    "duration_ms": int,
                    "total_cost_usd": float,
                    "subtype": str,
                    "usage": {
                        "input_tokens": int,
                        "output_tokens": int,
                        "total_tokens": int,
                        "cache_read_input_tokens": int,
                        "cache_creation_input_tokens": int
                    }
                }
        """
        logging.debug(f"Langfuse: finalize() called with result_payload={result_payload is not None}")

        # Cleanup and flush
        if self.langfuse_client:
            logging.debug(f"Langfuse: Finalizing trace")
            try:
                # Update trace with final metadata if we have results
                if result_payload:
                    # Extract usage data from ResultMessage
                    usage_data = result_payload.get("usage")

                    # Build usage dict for metadata
                    usage_dict = None
                    if usage_data and isinstance(usage_data, dict):
                        usage_dict = {
                            "input_tokens": usage_data.get("input_tokens", 0),
                            "output_tokens": usage_data.get("output_tokens", 0),
                            "total_tokens": usage_data.get("total_tokens", 0),
                            "cache_read_input_tokens": usage_data.get("cache_read_input_tokens", 0),
                            "cache_creation_input_tokens": usage_data.get("cache_creation_input_tokens", 0),
                        }
                        logging.info(f"Langfuse: Extracted usage from ResultMessage for metadata: {usage_dict}")

                    # Update the trace with output and metadata including usage
                    trace_metadata = {
                        "num_turns": result_payload.get("num_turns"),
                        "duration_ms": result_payload.get("duration_ms"),
                        "total_cost_usd": result_payload.get("total_cost_usd"),
                        "subtype": result_payload.get("subtype"),
                    }

                    # Add usage to metadata for visibility
                    if usage_dict:
                        trace_metadata["usage"] = usage_dict
                        logging.info(f"Langfuse: Adding usage to trace metadata: {usage_dict}")

                    self.langfuse_client.update_current_trace(
                        output=result_payload,
                        metadata=trace_metadata
                    )
                    logging.info("Langfuse: Updated trace with final metrics and usage")

                    # CRITICAL: Create a summary generation with usage_details for Anthropic-specific cost calculation
                    # This enables Langfuse to automatically calculate costs using custom Claude model pricing
                    # Reference: https://langfuse.com/docs/observability/features/token-and-cost-tracking
                    if usage_dict:
                        logging.info("Langfuse: Creating summary generation with Anthropic-specific usage_details for auto-cost calculation")
                        try:
                            # Use the model captured from track_generation() calls
                            # Falls back to environment variable if no generations were tracked
                            model_name = self.configured_model or os.getenv('LLM_MODEL', 'claude-3-5-sonnet-20241022')

                            # Create summary generation with Anthropic-specific usage_details
                            # Langfuse expects: input_tokens, output_tokens, cache_read_input_tokens
                            gen_ctx = self.langfuse_client.start_as_current_generation(
                                name="session_summary",
                                model=model_name,
                                input=[{"role": "system", "content": "Session usage summary"}],
                                metadata={
                                    "summary": True,
                                    "num_turns": result_payload.get("num_turns"),
                                    "duration_ms": result_payload.get("duration_ms"),
                                }
                            )
                            generation = gen_ctx.__enter__()

                            # Update with output and Anthropic-specific usage_details
                            # Field names match Anthropic API response format
                            generation.update(
                                output=f"Session completed with {result_payload.get('num_turns')} turns",
                                usage_details={
                                    "input_tokens": usage_dict.get("input_tokens", 0),
                                    "output_tokens": usage_dict.get("output_tokens", 0),
                                    "cache_read_input_tokens": usage_dict.get("cache_read_input_tokens", 0),
                                    # Include cache creation tokens if available
                                    "cache_creation_input_tokens": usage_dict.get("cache_creation_input_tokens", 0),
                                }
                            )

                            # End the generation
                            gen_ctx.__exit__(None, None, None)

                            logging.info(
                                f"Langfuse: Created session_summary generation with usage_details: "
                                f"input={usage_dict.get('input_tokens', 0)}, "
                                f"output={usage_dict.get('output_tokens', 0)}, "
                                f"cache_read={usage_dict.get('cache_read_input_tokens', 0)}, "
                                f"cache_creation={usage_dict.get('cache_creation_input_tokens', 0)} "
                                f"for model {model_name}. "
                                f"Langfuse will automatically calculate cost using custom model pricing."
                            )
                        except Exception as gen_err:
                            logging.error(f"Langfuse: Failed to create summary generation with usage_details: {gen_err}", exc_info=True)

                # Exit contexts in reverse order (root span first, then propagate_attributes)
                if self._root_span_ctx:
                    self._root_span_ctx.__exit__(None, None, None)
                    logging.info("Langfuse root span context exited")
                if self._propagate_ctx:
                    self._propagate_ctx.__exit__(None, None, None)
                    logging.info("Langfuse propagate_attributes context exited")

                # Flush to ensure all data is sent
                # Use 30s timeout to handle network latency and batch uploads
                success, _ = await with_sync_timeout(
                    self.langfuse_client.flush, 30.0, "Langfuse flush"
                )
                if success:
                    logging.info("Langfuse flush completed successfully")
                else:
                    # Error level for flush timeouts - this means observability data was lost
                    logging.error(
                        "Langfuse flush timed out after 30s - observability data may not be sent. "
                        "Check network connectivity to LANGFUSE_HOST."
                    )
            except Exception as e:
                logging.error(f"Failed to complete Langfuse session trace: {e}", exc_info=True)

    async def cleanup_on_error(self, error: Exception) -> None:
        """Finalize trace on error and flush observability data (error path).

        Updates trace with error details and exits propagate_attributes context.

        Args:
            error: Exception that caused the failure
        """
        # Cleanup and flush
        if self.langfuse_client:
            try:
                # Update trace with error details
                self.langfuse_client.update_current_trace(
                    output={"error": str(error)},
                    metadata={"error_type": type(error).__name__}
                )
                logging.debug("Langfuse: Marked trace as ERROR")

                # Exit contexts in reverse order (root span first, then propagate_attributes)
                if self._root_span_ctx:
                    self._root_span_ctx.__exit__(None, None, None)
                    logging.info("Langfuse root span context exited (error path)")
                if self._propagate_ctx:
                    self._propagate_ctx.__exit__(None, None, None)
                    logging.info("Langfuse propagate_attributes context exited (error path)")

                # Flush with 30s timeout (same as success path)
                success, _ = await with_sync_timeout(
                    self.langfuse_client.flush, 30.0, "Langfuse error cleanup flush"
                )
                if not success:
                    logging.error(
                        "Langfuse error cleanup flush timed out after 30s - "
                        "error trace may not be sent."
                    )
            except Exception as cleanup_err:
                logging.error(f"Failed to cleanup Langfuse trace: {cleanup_err}", exc_info=True)
