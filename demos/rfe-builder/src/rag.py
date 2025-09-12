#!/usr/bin/env python3
"""
RHOAI RAG Ingestion CLI

A consolidated command-line interface for the RAG data ingestion pipeline.
Combines features from all previous scripts with enhanced configurability.
"""

import json
import os
from pathlib import Path
from typing import Any, Dict, List, Optional

import click
import yaml
from dotenv import load_dotenv
from llama_index.core import Document, Settings, VectorStoreIndex
from llama_index.core.extractors import TitleExtractor
from llama_index.core.ingestion import IngestionPipeline
from llama_index.core.node_parser import SentenceSplitter
from llama_index.core.storage.storage_context import StorageContext

# LLM imports now handled by centralized settings

# Load environment variables
load_dotenv()

# Optional dependencies
try:
    from llama_index.readers.github import GithubClient, GithubRepositoryReader

    GITHUB_AVAILABLE = True
except ImportError:
    GITHUB_AVAILABLE = False

try:
    from llama_index.readers.web import SimpleWebPageReader

    WEB_AVAILABLE = True
except ImportError:
    WEB_AVAILABLE = False


class RAGIngestorError(Exception):
    """Custom exception for RAG ingestion errors"""

    pass


class RAGIngestor:
    """Main RAG ingestion class with consolidated functionality"""

    def __init__(
        self,
        agents_dir: Path,
        output_dir: Path,
        chunking_strategy: str = "sentence",
        verbose: bool = False,
    ):
        self.agents_dir = agents_dir
        self.output_dir = output_dir
        self.chunking_strategy = chunking_strategy
        self.verbose = verbose
        self.github_token = os.getenv("GITHUB_TOKEN")

        # Setup LlamaIndex using centralized settings
        from src.settings import init_settings

        init_settings()

        # Create ingestion pipeline
        self.pipeline = self._create_ingestion_pipeline()

    def _create_ingestion_pipeline(self) -> IngestionPipeline:
        """Create ingestion pipeline with specified chunking strategy"""
        transformations = []

        if self.chunking_strategy == "sentence":
            transformations.append(
                SentenceSplitter(
                    chunk_size=512,
                    chunk_overlap=50,
                    separator=" ",
                )
            )
        elif self.chunking_strategy == "semantic":
            transformations.append(
                SentenceSplitter(
                    chunk_size=1024,
                    chunk_overlap=100,
                )
            )
        elif self.chunking_strategy == "large":
            transformations.append(
                SentenceSplitter(
                    chunk_size=2048,
                    chunk_overlap=200,
                )
            )

        # Add metadata extraction
        transformations.append(TitleExtractor())

        # Add embeddings (uses Settings.embed_model from init_settings)
        transformations.append(Settings.embed_model)

        return IngestionPipeline(transformations=transformations)

    def load_agent_configs(
        self, specific_agents: Optional[List[str]] = None
    ) -> Dict[str, Dict]:
        """Load agent configurations from YAML files"""
        agents = {}

        for yaml_file in self.agents_dir.glob("*.yaml"):
            if yaml_file.name.startswith("agent-schema"):
                continue

            try:
                with open(yaml_file, "r") as f:
                    config = yaml.safe_load(f)

                persona = config.get("persona")
                if not persona:
                    continue

                # Filter by specific agents if requested
                if specific_agents and persona not in specific_agents:
                    continue

                agents[persona] = config
                if self.verbose:
                    click.echo(
                        f"✅ Loaded agent: {persona} ({config.get('name', 'Unknown')})"
                    )

            except Exception as e:
                click.echo(f"❌ Error loading {yaml_file}: {e}", err=True)

        return agents

    def ingest_github_source(
        self, source_config: Dict, use_pipeline: bool = True
    ) -> List[Document]:
        """Ingest documents from GitHub repository"""
        if not GITHUB_AVAILABLE:
            click.echo("⚠️ GitHub reader not available, skipping...", err=True)
            return []

        if not self.github_token:
            click.echo("⚠️ GITHUB_TOKEN not set, skipping GitHub source...", err=True)
            return []

        try:
            github_client = GithubClient(
                github_token=self.github_token, verbose=self.verbose
            )

            # Parse repository
            if "/" in source_config["source"]:
                owner, repo = source_config["source"].split("/", 1)
            else:
                raise ValueError(
                    f"Invalid GitHub source format: {source_config['source']}"
                )

            options = source_config.get("options", {})
            branch = options.get("branch", "main")

            if self.verbose:
                click.echo(f"📥 Loading from {owner}/{repo} (branch: {branch})")

            # Set up filters
            filter_dirs = None
            filter_files = None

            if "path" in options:
                filter_dirs = (
                    [options["path"]],
                    GithubRepositoryReader.FilterType.INCLUDE,
                )
                if self.verbose:
                    click.echo(f"🗂️  Filtering to path: {options['path']}")

            if "fileTypes" in options:
                filter_files = (
                    options["fileTypes"],
                    GithubRepositoryReader.FilterType.INCLUDE,
                )
                if self.verbose:
                    click.echo(f"📄 Filtering to file types: {options['fileTypes']}")

            reader = GithubRepositoryReader(
                github_client=github_client,
                owner=owner,
                repo=repo,
                use_parser=False,
                verbose=self.verbose,
                filter_directories=filter_dirs,
                filter_file_extensions=filter_files,
            )

            # Load raw documents
            raw_documents = reader.load_data(branch=branch)
            click.echo(f"📚 Loaded {len(raw_documents)} documents from {owner}/{repo}")

            if use_pipeline and raw_documents:
                # Process through pipeline
                if self.verbose:
                    click.echo("🔄 Processing through ingestion pipeline...")

                processed_nodes = self.pipeline.run(documents=raw_documents)

                # Convert nodes back to documents
                processed_docs = []
                for node in processed_nodes:
                    doc = Document(
                        text=node.get_content(),
                        metadata={
                            **node.metadata,
                            "source_type": "github",
                            "repository": f"{owner}/{repo}",
                            "branch": branch,
                            "agent_source": source_config["name"],
                            "processed": True,
                        },
                    )
                    processed_docs.append(doc)

                click.echo(
                    f"✅ Pipeline produced {len(processed_docs)} processed chunks"
                )
                return processed_docs
            else:
                # Simple processing without pipeline
                for doc in raw_documents:
                    doc.metadata.update(
                        {
                            "source_type": "github",
                            "repository": f"{owner}/{repo}",
                            "branch": branch,
                            "agent_source": source_config["name"],
                            "processed": False,
                        }
                    )

                return raw_documents

        except Exception as e:
            click.echo(
                f"❌ Error ingesting GitHub source {source_config['name']}: {e}",
                err=True,
            )
            return []

    def ingest_web_source(self, source_config: Dict) -> List[Document]:
        """Ingest documents from web URL"""
        if not WEB_AVAILABLE:
            click.echo("⚠️ Web reader not available, skipping...", err=True)
            return []

        try:
            reader = SimpleWebPageReader(html_to_text=True)
            documents = reader.load_data([source_config["source"]])

            for doc in documents:
                doc.metadata.update(
                    {
                        "source_type": "web",
                        "url": source_config["source"],
                        "agent_source": source_config["name"],
                    }
                )

            click.echo(
                f"🌐 Loaded {len(documents)} documents from {source_config['source']}"
            )
            return documents

        except Exception as e:
            click.echo(
                f"❌ Error ingesting web source {source_config['name']}: {e}", err=True
            )
            return []

    def ingest_agent_data(
        self, agent_persona: str, agent_config: Dict, use_pipeline: bool = True
    ) -> List[Document]:
        """Ingest all data sources for a specific agent"""
        click.echo(
            f"\n🔄 Ingesting data for {agent_persona} ({agent_config.get('name', 'Unknown')})..."
        )

        all_documents = []
        data_sources = agent_config.get("dataSources", [])

        for source in data_sources:
            # Handle simple string sources (local directories)
            if isinstance(source, str):
                if self.verbose:
                    click.echo(
                        f"📁 Skipping local directory: {source} (handled by TypeScript)"
                    )
                continue

            # Handle advanced source configurations
            if isinstance(source, dict):
                source_type = source.get("type", "directory")

                if source_type == "github":
                    docs = self.ingest_github_source(source, use_pipeline)
                    all_documents.extend(docs)
                elif source_type == "web":
                    docs = self.ingest_web_source(source)
                    all_documents.extend(docs)
                else:
                    click.echo(f"⚠️ Unsupported source type: {source_type}")

        click.echo(f"✅ Total documents for {agent_persona}: {len(all_documents)}")
        return all_documents

    def create_vector_index(
        self, documents: List[Document], agent_persona: str
    ) -> Optional[VectorStoreIndex]:
        """Create and persist vector index"""
        if not documents:
            click.echo(f"⚠️ No documents to index for {agent_persona}")
            return None

        # Create agent-specific output directory
        agent_output_dir = self.output_dir / agent_persona.lower()
        agent_output_dir.mkdir(parents=True, exist_ok=True)

        # Create index first with default storage context
        click.echo(f"🔮 Creating vector index for {agent_persona}...")
        index = VectorStoreIndex.from_documents(documents, show_progress=True)

        # Persist index to the specified directory
        index.storage_context.persist(persist_dir=str(agent_output_dir))

        # Save metadata
        metadata = {
            "agent_persona": agent_persona,
            "document_count": len(documents),
            "chunking_strategy": self.chunking_strategy,
            "sources": list(
                set(doc.metadata.get("agent_source", "unknown") for doc in documents)
            ),
            "sample_files": [
                doc.metadata.get("file_path", "unknown") for doc in documents[:5]
            ],
            "created_at": str(Path().absolute()),
            "index_type": "VectorStoreIndex",
        }

        with open(agent_output_dir / "metadata.json", "w") as f:
            json.dump(metadata, f, indent=2)

        click.echo(f"💾 Saved index for {agent_persona} to {agent_output_dir}")
        return index

    def test_index(
        self, index: VectorStoreIndex, test_query: str = "How to deploy"
    ) -> None:
        """Test the created index with a sample query"""
        if not index:
            return

        click.echo(f"\n🧪 Testing index with query: '{test_query}'")

        try:
            retriever = index.as_retriever(similarity_top_k=3)
            nodes = retriever.retrieve(test_query)

            click.echo(f"📖 Retrieved {len(nodes)} relevant documents:")
            for i, node in enumerate(nodes[:2], 1):
                content_preview = (
                    node.get_content()[:150] + "..."
                    if len(node.get_content()) > 150
                    else node.get_content()
                )
                file_path = node.metadata.get("file_path", "unknown")
                click.echo(f"  {i}. {file_path}: {content_preview}")

        except Exception as e:
            click.echo(f"❌ Test failed: {e}", err=True)


@click.group(invoke_without_command=True)
@click.option("--version", is_flag=True, help="Show version")
@click.pass_context
def cli(ctx, version):
    """RHOAI RAG Ingestion Pipeline - Consolidated CLI Tool"""
    if version:
        from .rag.rhoai_rag_ingestion import __version__

        click.echo(f"rhoai-rag-ingestion {__version__}")
        return

    if ctx.invoked_subcommand is None:
        click.echo(ctx.get_help())


@cli.command()
@click.option(
    "--agents-dir",
    "-a",
    type=click.Path(exists=True, path_type=Path),
    default=Path("src/agents"),
    help="Directory containing agent YAML configs",
)
@click.option(
    "--output-dir",
    "-o",
    type=click.Path(path_type=Path),
    default=Path("output/python-rag"),
    help="Output directory for vector stores",
)
@click.option(
    "--chunking-strategy",
    "-c",
    type=click.Choice(["sentence", "semantic", "large"]),
    default="sentence",
    help="Text chunking strategy",
)
@click.option(
    "--agents", "-ag", multiple=True, help="Specific agents to process (default: all)"
)
@click.option(
    "--use-pipeline/--no-pipeline", default=True, help="Use advanced ingestion pipeline"
)
@click.option("--verbose", "-v", is_flag=True, help="Verbose output")
@click.option("--test", "-t", is_flag=True, help="Test created indexes")
@click.option("--test-query", default="How to deploy", help="Query for testing indexes")
def ingest(
    agents_dir,
    output_dir,
    chunking_strategy,
    agents,
    use_pipeline,
    verbose,
    test,
    test_query,
):
    """Ingest data from configured sources into vector stores"""

    # Validate prerequisites
    if not os.getenv("OPENAI_API_KEY"):
        click.echo("❌ OPENAI_API_KEY environment variable required", err=True)
        return

    # Initialize ingestor
    ingestor = RAGIngestor(
        agents_dir=agents_dir,
        output_dir=output_dir,
        chunking_strategy=chunking_strategy,
        verbose=verbose,
    )

    click.echo("🚀 Starting RAG Data Ingestion Pipeline\n")

    # Load agent configurations
    click.echo("📋 Loading agent configurations...")
    agent_configs = ingestor.load_agent_configs(list(agents) if agents else None)

    if not agent_configs:
        click.echo("❌ No agent configurations found!", err=True)
        return

    # Create output directory
    output_dir.mkdir(parents=True, exist_ok=True)

    # Process each agent
    indexes_created = []
    for persona, config in agent_configs.items():
        try:
            # Ingest documents
            documents = ingestor.ingest_agent_data(persona, config, use_pipeline)

            # Create vector index
            index = ingestor.create_vector_index(documents, persona)
            if index:
                indexes_created.append((persona, index))

        except Exception as e:
            click.echo(f"❌ Error processing {persona}: {e}", err=True)
            continue

    # Test indexes if requested
    if test and indexes_created:
        click.echo(f"\n🧪 Testing created indexes...")
        for persona, index in indexes_created:
            click.echo(f"\n--- Testing {persona} ---")
            ingestor.test_index(index, test_query)

    click.echo(f"\n🎉 RAG ingestion complete!")
    click.echo(f"   Processed {len(indexes_created)} agents")
    click.echo(f"   Output location: {output_dir}")


@cli.command()
@click.option(
    "--agents-dir",
    "-a",
    type=click.Path(exists=True, path_type=Path),
    default=Path("../src/agents"),
    help="Directory containing agent YAML configs",
)
def list_agents(agents_dir):
    """List available agents and their configurations"""

    click.echo("📋 Available Agents:\n")

    for yaml_file in agents_dir.glob("*.yaml"):
        if yaml_file.name.startswith("agent-schema"):
            continue

        try:
            with open(yaml_file, "r") as f:
                config = yaml.safe_load(f)

            persona = config.get("persona", "Unknown")
            name = config.get("name", "Unknown")
            data_sources = config.get("dataSources", [])

            click.echo(f"• {persona} ({name})")
            click.echo(f"  File: {yaml_file.name}")
            click.echo(f"  Data Sources: {len(data_sources)}")

            for source in data_sources[:3]:  # Show first 3 sources
                if isinstance(source, dict):
                    source_type = source.get("type", "directory")
                    source_name = source.get("name", source.get("source", "unknown"))
                    click.echo(f"    - {source_type}: {source_name}")
                else:
                    click.echo(f"    - directory: {source}")

            if len(data_sources) > 3:
                click.echo(f"    ... and {len(data_sources) - 3} more")

            click.echo("")

        except Exception as e:
            click.echo(f"❌ Error reading {yaml_file}: {e}", err=True)


if __name__ == "__main__":
    cli()
