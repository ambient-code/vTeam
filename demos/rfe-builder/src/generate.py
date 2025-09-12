"""
Generate indices from existing data for the RHOAI backend
"""

import os
from pathlib import Path

from llama_index.core import SimpleDirectoryReader, VectorStoreIndex
from llama_index.core.settings import Settings
from llama_index.core.storage import StorageContext
from llama_index.embeddings.openai import OpenAIEmbedding
from llama_index.llms.openai import OpenAI

from src.settings import init_settings


def generate_indices():
    """Generate vector indices from local data sources"""
    print("🔧 Generating indices for RHOAI backend...")

    # Initialize settings
    init_settings()

    # Check for existing data
    data_dir = Path("../data")
    output_dir = Path("../output/backend-storage")

    if not data_dir.exists():
        print(f"⚠️  Data directory not found: {data_dir}")
        print("💡 Using existing indices from python-rag ingestion")
        return

    # Create output directory
    output_dir.mkdir(parents=True, exist_ok=True)

    # Simple index generation from local data
    try:
        documents = SimpleDirectoryReader(str(data_dir)).load_data()

        if documents:
            print(f"📚 Loading {len(documents)} documents...")

            # Create index
            storage_context = StorageContext.from_defaults(persist_dir=str(output_dir))
            index = VectorStoreIndex.from_documents(
                documents, storage_context=storage_context, show_progress=True
            )

            print(f"💾 Saved index to {output_dir}")

            # Test the index
            query_engine = index.as_query_engine()
            response = query_engine.query("What is this about?")
            print(f"🧪 Test query result: {response}")

        else:
            print("📭 No documents found in data directory")

    except Exception as e:
        print(f"❌ Error generating indices: {e}")
        print("💡 Make sure OPENAI_API_KEY is set and data directory exists")


if __name__ == "__main__":
    generate_indices()
