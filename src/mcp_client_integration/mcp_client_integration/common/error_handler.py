#!/usr/bin/env python3
"""
MCP Error Handling Utilities

This module provides standardized error handling for MCP operations,
consolidating error handling patterns and providing consistent error
responses across all MCP components.
"""

import asyncio
import logging
import traceback
from contextlib import asynccontextmanager
from typing import Dict, Any, Optional, Union, Type
from dataclasses import dataclass
from enum import Enum

# Configure logging
logger = logging.getLogger(__name__)


class MCPErrorCategory(Enum):
    """Categories of MCP errors for structured error handling."""
    CONNECTION = "connection"
    CONFIGURATION = "configuration"
    VALIDATION = "validation"
    PROTOCOL = "protocol"
    TIMEOUT = "timeout"
    AUTHENTICATION = "authentication"
    UNKNOWN = "unknown"


@dataclass
class MCPErrorContext:
    """Context information for MCP errors."""
    operation: str
    endpoint: Optional[str] = None
    capability: Optional[str] = None
    message_id: Optional[str] = None
    additional_info: Optional[Dict[str, Any]] = None


class MCPError(Exception):
    """
    Base MCP error class with structured error information.
    
    This class provides a standardized way to represent errors
    across all MCP operations.
    """
    
    def __init__(
        self,
        message: str,
        category: MCPErrorCategory = MCPErrorCategory.UNKNOWN,
        context: Optional[MCPErrorContext] = None,
        original_error: Optional[Exception] = None
    ):
        """
        Initialize MCP error.
        
        Args:
            message: Human-readable error message
            category: Error category for classification
            context: Context information about the error
            original_error: Original exception that caused this error
        """
        super().__init__(message)
        self.message = message
        self.category = category
        self.context = context or MCPErrorContext(operation="unknown")
        self.original_error = original_error
        
        logger.debug(f"MCPError created: {category.value} - {message}")
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert error to dictionary format."""
        error_dict = {
            "error": True,
            "message": self.message,
            "category": self.category.value,
            "operation": self.context.operation
        }
        
        if self.context.endpoint:
            error_dict["endpoint"] = self.context.endpoint
        
        if self.context.capability:
            error_dict["capability"] = self.context.capability
        
        if self.context.message_id:
            error_dict["message_id"] = self.context.message_id
        
        if self.context.additional_info:
            error_dict["additional_info"] = self.context.additional_info
        
        if self.original_error:
            error_dict["original_error"] = str(self.original_error)
            error_dict["original_error_type"] = type(self.original_error).__name__
        
        return error_dict


class MCPConnectionError(MCPError):
    """Error related to MCP connection issues."""
    
    def __init__(self, message: str, endpoint: str, original_error: Optional[Exception] = None):
        context = MCPErrorContext(operation="connection", endpoint=endpoint)
        super().__init__(message, MCPErrorCategory.CONNECTION, context, original_error)


class MCPConfigurationError(MCPError):
    """Error related to MCP configuration issues."""
    
    def __init__(self, message: str, original_error: Optional[Exception] = None):
        context = MCPErrorContext(operation="configuration")
        super().__init__(message, MCPErrorCategory.CONFIGURATION, context, original_error)


class MCPValidationError(MCPError):
    """Error related to MCP validation issues."""
    
    def __init__(self, message: str, endpoint: Optional[str] = None, original_error: Optional[Exception] = None):
        context = MCPErrorContext(operation="validation", endpoint=endpoint)
        super().__init__(message, MCPErrorCategory.VALIDATION, context, original_error)


class MCPProtocolError(MCPError):
    """Error related to MCP protocol issues."""
    
    def __init__(self, message: str, capability: Optional[str] = None, message_id: Optional[str] = None, original_error: Optional[Exception] = None):
        context = MCPErrorContext(operation="protocol", capability=capability, message_id=message_id)
        super().__init__(message, MCPErrorCategory.PROTOCOL, context, original_error)


class MCPTimeoutError(MCPError):
    """Error related to MCP timeout issues."""
    
    def __init__(self, message: str, endpoint: str, timeout_seconds: int, original_error: Optional[Exception] = None):
        context = MCPErrorContext(
            operation="timeout", 
            endpoint=endpoint,
            additional_info={"timeout_seconds": timeout_seconds}
        )
        super().__init__(message, MCPErrorCategory.TIMEOUT, context, original_error)


class MCPErrorHandler:
    """
    Standardized error handling for MCP operations.
    
    This class provides consistent error handling patterns,
    logging, and error response formatting across all MCP components.
    """
    
    def __init__(self, logger_name: Optional[str] = None, max_error_history: int = 1000):
        """
        Initialize error handler.
        
        Args:
            logger_name: Optional logger name, defaults to class module
            max_error_history: Maximum number of error entries to track (prevents memory leaks)
        """
        self.logger = logging.getLogger(logger_name or __name__)
        self._error_counts: Dict[str, int] = {}
        self._max_error_history = max_error_history
        
        logger.debug(f"MCPErrorHandler initialized (max_error_history={max_error_history})")
    
    @asynccontextmanager
    async def handle_connection_errors(self, operation_name: str, endpoint: str):
        """
        Context manager for connection error handling.
        
        Args:
            operation_name: Name of the operation being performed
            endpoint: Endpoint being connected to
        """
        try:
            yield
        except asyncio.TimeoutError as e:
            error_msg = f"Connection timeout during {operation_name} to {endpoint}"
            self.logger.error(error_msg)
            self._increment_error_count(f"timeout_{operation_name}")
            raise MCPTimeoutError(error_msg, endpoint, 30, e)
        except ConnectionError as e:
            error_msg = f"Connection failed during {operation_name} to {endpoint}: {e}"
            self.logger.error(error_msg)
            self._increment_error_count(f"connection_{operation_name}")
            raise MCPConnectionError(error_msg, endpoint, e)
        except Exception as e:
            error_msg = f"Unexpected error during {operation_name} to {endpoint}: {e}"
            self.logger.error(error_msg, exc_info=True)
            self._increment_error_count(f"unknown_{operation_name}")
            raise MCPError(error_msg, MCPErrorCategory.UNKNOWN, 
                          MCPErrorContext(operation_name, endpoint), e)
    
    @asynccontextmanager
    async def handle_protocol_errors(self, operation_name: str, capability: Optional[str] = None):
        """
        Context manager for protocol error handling.
        
        Args:
            operation_name: Name of the operation being performed
            capability: Optional capability being used
        """
        try:
            yield
        except MCPError:
            # Re-raise MCP errors as-is
            raise
        except asyncio.TimeoutError as e:
            error_msg = f"Protocol timeout during {operation_name}"
            if capability:
                error_msg += f" for capability {capability}"
            self.logger.error(error_msg)
            self._increment_error_count(f"protocol_timeout_{operation_name}")
            raise MCPTimeoutError(error_msg, capability or "unknown", 30, e)
        except Exception as e:
            error_msg = f"Protocol error during {operation_name}: {e}"
            self.logger.error(error_msg, exc_info=True)
            self._increment_error_count(f"protocol_{operation_name}")
            raise MCPProtocolError(error_msg, capability, original_error=e)
    
    def handle_configuration_error(self, operation_name: str, error: Exception) -> MCPConfigurationError:
        """
        Handle configuration errors.
        
        Args:
            operation_name: Name of the operation that failed
            error: Original error
            
        Returns:
            MCPConfigurationError with standardized format
        """
        error_msg = f"Configuration error during {operation_name}: {error}"
        self.logger.error(error_msg)
        self._increment_error_count(f"config_{operation_name}")
        return MCPConfigurationError(error_msg, error)
    
    def handle_validation_error(self, operation_name: str, endpoint: Optional[str], error: Exception) -> MCPValidationError:
        """
        Handle validation errors.
        
        Args:
            operation_name: Name of the operation that failed
            endpoint: Optional endpoint being validated
            error: Original error
            
        Returns:
            MCPValidationError with standardized format
        """
        error_msg = f"Validation error during {operation_name}"
        if endpoint:
            error_msg += f" for endpoint {endpoint}"
        error_msg += f": {error}"
        
        self.logger.error(error_msg)
        self._increment_error_count(f"validation_{operation_name}")
        return MCPValidationError(error_msg, endpoint, error)
    
    def format_error_response(self, error: Union[Exception, MCPError], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Standardize error response format.
        
        Args:
            error: The error to format
            context: Optional additional context
            
        Returns:
            Standardized error response dictionary
        """
        if isinstance(error, MCPError):
            response = error.to_dict()
        else:
            response = {
                "error": True,
                "message": str(error),
                "category": MCPErrorCategory.UNKNOWN.value,
                "operation": "unknown",
                "original_error_type": type(error).__name__
            }
        
        # Add additional context if provided
        if context:
            response["context"] = context
        
        # Add timestamp
        response["timestamp"] = asyncio.get_event_loop().time()
        
        # Add error tracking info
        response["error_counts"] = self.get_error_summary()
        
        return response
    
    def create_success_response(self, data: Any, operation: str, context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Create standardized success response.
        
        Args:
            data: The success data
            operation: Operation that succeeded
            context: Optional additional context
            
        Returns:
            Standardized success response dictionary
        """
        response = {
            "error": False,
            "success": True,
            "data": data,
            "operation": operation,
            "timestamp": asyncio.get_event_loop().time()
        }
        
        if context:
            response["context"] = context
        
        return response
    
    def _increment_error_count(self, error_type: str) -> None:
        """Increment error count for tracking with memory limit protection."""
        # Prevent unbounded growth by cleaning up old entries when limit is reached
        if len(self._error_counts) >= self._max_error_history:
            # Remove oldest half of entries (simple cleanup strategy)
            sorted_keys = sorted(self._error_counts.keys())
            keys_to_remove = sorted_keys[:len(sorted_keys) // 2]
            for key in keys_to_remove:
                del self._error_counts[key]
            logger.debug(f"Cleaned up {len(keys_to_remove)} old error count entries")
        
        self._error_counts[error_type] = self._error_counts.get(error_type, 0) + 1
    
    def get_error_summary(self) -> Dict[str, int]:
        """Get summary of error counts."""
        return self._error_counts.copy()
    
    def reset_error_counts(self) -> None:
        """Reset error count tracking."""
        self._error_counts.clear()
        logger.debug("Error counts reset")
    
    def log_error_with_context(
        self, 
        error: Exception, 
        operation: str,
        endpoint: Optional[str] = None,
        capability: Optional[str] = None,
        additional_context: Optional[Dict[str, Any]] = None
    ) -> None:
        """
        Log error with structured context information.
        
        Args:
            error: The error to log
            operation: Operation that failed
            endpoint: Optional endpoint
            capability: Optional capability
            additional_context: Optional additional context
        """
        context_info = {
            "operation": operation,
            "error_type": type(error).__name__,
            "error_message": str(error)
        }
        
        if endpoint:
            context_info["endpoint"] = endpoint
        
        if capability:
            context_info["capability"] = capability
        
        if additional_context:
            context_info.update(additional_context)
        
        self.logger.error(
            f"MCP operation failed: {operation}",
            extra={"mcp_context": context_info},
            exc_info=True
        )


# Global error handler instance for convenience
default_error_handler = MCPErrorHandler()


def handle_mcp_errors(operation: str, endpoint: Optional[str] = None):
    """
    Decorator for automatic MCP error handling.
    
    Args:
        operation: Name of the operation
        endpoint: Optional endpoint being operated on
    """
    def decorator(func):
        if asyncio.iscoroutinefunction(func):
            async def async_wrapper(*args, **kwargs):
                try:
                    return await func(*args, **kwargs)
                except MCPError:
                    raise  # Re-raise MCP errors as-is
                except Exception as e:
                    default_error_handler.log_error_with_context(
                        e, operation, endpoint
                    )
                    raise MCPError(
                        f"Error in {operation}: {e}",
                        MCPErrorCategory.UNKNOWN,
                        MCPErrorContext(operation, endpoint),
                        e
                    )
            return async_wrapper
        else:
            def sync_wrapper(*args, **kwargs):
                try:
                    return func(*args, **kwargs)
                except MCPError:
                    raise  # Re-raise MCP errors as-is
                except Exception as e:
                    default_error_handler.log_error_with_context(
                        e, operation, endpoint
                    )
                    raise MCPError(
                        f"Error in {operation}: {e}",
                        MCPErrorCategory.UNKNOWN,
                        MCPErrorContext(operation, endpoint),
                        e
                    )
            return sync_wrapper
    return decorator