"""
Runner context providing session information and utilities.
"""

import os
from typing import Dict, Any, Optional
from dataclasses import dataclass, field


@dataclass
class RunnerContext:
    """Context provided to runner adapters."""

    session_id: str
    workspace_path: str
    environment: Dict[str, str] = field(default_factory=dict)
    metadata: Dict[str, Any] = field(default_factory=dict)

    def __post_init__(self):
        """Initialize context after creation."""
        # Set workspace as current directory
        if os.path.exists(self.workspace_path):
            os.chdir(self.workspace_path)

        # Merge environment variables
        self.environment = {**os.environ, **self.environment}

    def get_env(self, key: str, default: Optional[str] = None) -> Optional[str]:
        """Get environment variable."""
        return self.environment.get(key, default)

    def set_metadata(self, key: str, value: Any):
        """Set metadata value."""
        self.metadata[key] = value

    def get_metadata(self, key: str, default: Any = None) -> Any:
        """Get metadata value."""
        return self.metadata.get(key, default)