import os
from typing import Any, Dict, Optional, Type

from dotenv import load_dotenv
from llama_index.core.embeddings import BaseEmbedding
from llama_index.core.llms import LLM
from llama_index.core.settings import Settings


class ProviderRegistry:
    """Registry for LLM and embedding providers using configuration-driven approach"""

    # Configuration for each provider - eliminates boilerplate
    LLM_CONFIGS = {
        "openai": {
            "module": "llama_index.llms.openai",
            "class": "OpenAI",
            "defaults": {
                "model": "gpt-4",
                "temperature": 0.1,
                "streaming": False,
                "max_tokens": 4000,
            },
            "env_map": {"api_key": "OPENAI_API_KEY", "api_base": "OPENAI_API_BASE"},
        },
        "anthropic": {
            "module": "llama_index.llms.anthropic",
            "class": "Anthropic",
            "defaults": {
                "model": "claude-3-haiku-20240307",
                "temperature": 0.1,
                "max_tokens": 4000,
            },
            "env_map": {"api_key": "ANTHROPIC_API_KEY"},
            "package": "llama-index-llms-anthropic",
        },
        "ollama": {
            "module": "llama_index.llms.ollama",
            "class": "Ollama",
            "defaults": {
                "model": "llama2",
                "temperature": 0.1,
                "base_url": "http://localhost:11434",
                "max_tokens": 4000,
            },
            "package": "llama-index-llms-ollama",
        },
    }

    EMBEDDING_CONFIGS = {
        "openai": {
            "module": "llama_index.embeddings.openai",
            "class": "OpenAIEmbedding",
            "defaults": {"model": "text-embedding-3-small"},
            "env_map": {"api_key": "OPENAI_API_KEY"},
        },
        "huggingface": {
            "module": "llama_index.embeddings.huggingface",
            "class": "HuggingFaceEmbedding",
            "defaults": {"model_name": "sentence-transformers/all-MiniLM-L6-v2"},
            "package": "llama-index-embeddings-huggingface",
        },
        "ollama": {
            "module": "llama_index.embeddings.ollama",
            "class": "OllamaEmbedding",
            "defaults": {"model_name": "llama2", "base_url": "http://localhost:11434"},
            "package": "llama-index-embeddings-ollama",
        },
    }

    @classmethod
    def _create_instance(
        cls, provider: str, config_dict: Dict, user_kwargs: Dict
    ) -> Any:
        """Generic factory method to create provider instances"""
        if provider not in config_dict:
            raise ValueError(
                f"Unsupported provider: {provider}. Available: {list(config_dict.keys())}"
            )

        config = config_dict[provider]

        # Import the class dynamically
        try:
            module = __import__(config["module"], fromlist=[config["class"]])
            provider_class = getattr(module, config["class"])
        except ImportError as e:
            package = config.get("package", f"llama-index-{provider}")
            raise ImportError(
                f"{provider} not installed. Run: pip install {package}"
            ) from e

        # Merge defaults with environment variables and user overrides
        kwargs = config["defaults"].copy()

        # Map environment variables
        for param, env_var in config.get("env_map", {}).items():
            if env_value := os.getenv(env_var):
                kwargs[param] = env_value

        # Apply user overrides
        kwargs.update(user_kwargs)

        return provider_class(**kwargs)

    @classmethod
    def get_llm(cls, provider: str = "openai", **kwargs) -> LLM:
        """Get LLM instance using configuration"""
        return cls._create_instance(provider.lower(), cls.LLM_CONFIGS, kwargs)

    @classmethod
    def get_embedding_model(cls, provider: str = "openai", **kwargs) -> BaseEmbedding:
        """Get embedding model instance using configuration"""
        return cls._create_instance(provider.lower(), cls.EMBEDDING_CONFIGS, kwargs)


def init_settings(
    llm_provider: Optional[str] = None,
    embedding_provider: Optional[str] = None,
    llm_config: Optional[Dict[str, Any]] = None,
    embedding_config: Optional[Dict[str, Any]] = None,
    **global_settings,
) -> None:
    """Initialize LlamaIndex settings with minimal configuration"""
    load_dotenv()

    # Use environment-first approach with sensible defaults
    llm_provider = llm_provider or os.getenv("LLM_PROVIDER", "openai")
    embedding_provider = embedding_provider or os.getenv("EMBEDDING_PROVIDER", "openai")

    # Initialize providers
    Settings.llm = ProviderRegistry.get_llm(llm_provider, **(llm_config or {}))
    Settings.embed_model = ProviderRegistry.get_embedding_model(
        embedding_provider, **(embedding_config or {})
    )

    # Configure global settings with environment fallbacks
    Settings.chunk_size = global_settings.get(
        "chunk_size", int(os.getenv("CHUNK_SIZE", "512"))
    )
    Settings.chunk_overlap = global_settings.get(
        "chunk_overlap", int(os.getenv("CHUNK_OVERLAP", "50"))
    )


def configure_mcp_integration(
    config_override: Optional[Dict[str, Any]] = None,
) -> Dict[str, Any]:
    """
    Configure MCP integration for RFE Builder.

    Args:
        config_override: Optional configuration overrides

    Returns:
        MCP configuration dictionary
    """
    from .mcp_config_validator import RFEBuilderMCPConfig

    # Create MCP configuration
    mcp_config = RFEBuilderMCPConfig()

    # Validate configuration
    validation_result = mcp_config.validate_configuration()

    # Apply any overrides
    if config_override:
        # Apply configuration overrides here if needed
        pass

    return {
        "mcp_validation": validation_result,
        "mcp_enabled": validation_result.get("valid", False),
        "production_mode": mcp_config.production_mode,
    }


def init_settings_with_mcp(
    llm_provider: Optional[str] = None,
    embedding_provider: Optional[str] = None,
    llm_config: Optional[Dict[str, Any]] = None,
    embedding_config: Optional[Dict[str, Any]] = None,
    mcp_config: Optional[Dict[str, Any]] = None,
    **global_settings,
) -> Dict[str, Any]:
    """
    Initialize LlamaIndex settings with MCP integration.

    Returns:
        Initialization results including MCP status
    """
    # Initialize LlamaIndex settings
    init_settings(
        llm_provider,
        embedding_provider,
        llm_config,
        embedding_config,
        **global_settings,
    )

    # Configure MCP integration
    mcp_status = configure_mcp_integration(mcp_config)

    return {"llama_index_initialized": True, "mcp_status": mcp_status}
