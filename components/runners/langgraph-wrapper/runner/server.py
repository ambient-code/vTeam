#!/usr/bin/env python3
"""
LangGraph workflow runner server.
Provides HTTP endpoints for starting, resuming, and querying LangGraph workflow runs.
"""

import os
import sys
import importlib
import asyncio
import logging
from typing import Dict, Any, Optional
from pathlib import Path

from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from langgraph.checkpoint.postgres import PostgresSaver

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

app = FastAPI(title="LangGraph Runner Server")

# Global state
_graph: Optional[Any] = None
_checkpointer: Optional[PostgresSaver] = None
_run_id: Optional[str] = None
_artifacts_dir: Optional[str] = None
_backend_url: Optional[str] = None
_stream_task: Optional[asyncio.Task] = None

# Request models
class StartRequest(BaseModel):
    run_id: str
    inputs: Dict[str, Any]

class ResumeRequest(BaseModel):
    checkpoint_id: str
    values: Dict[str, Any]

class EventPayload(BaseModel):
    run_id: str
    seq: int
    ts: str
    type: str
    node: Optional[str] = None
    checkpoint_id: Optional[str] = None
    payload: Dict[str, Any]


def load_graph(entry: str):
    """Load a compiled graph from the specified module:function entry point."""
    try:
        module_path, function_name = entry.split(":", 1)
        logger.info(f"Loading graph from {module_path}:{function_name}")
        
        # Add workflow directory to path if it exists
        workflow_dir = Path("/app/workflow")
        if workflow_dir.exists():
            sys.path.insert(0, str(workflow_dir))
        
        # Import module
        module = importlib.import_module(module_path)
        
        # Get function
        build_func = getattr(module, function_name)
        
        # Call function to get graph
        graph = build_func()
        
        logger.info(f"Successfully loaded graph from {entry}")
        return graph
    except Exception as e:
        logger.error(f"Failed to load graph from {entry}: {e}", exc_info=True)
        raise ValueError(f"Failed to load graph from {entry}: {e}")


def initialize_checkpointer():
    """Initialize PostgresSaver checkpointer."""
    pg_dsn = os.getenv("PG_DSN")
    if not pg_dsn:
        # Build DSN from individual env vars
        pg_host = os.getenv("POSTGRES_HOST", "postgres-service")
        pg_port = os.getenv("POSTGRES_PORT", "5432")
        pg_user = os.getenv("POSTGRES_USER", "langgraph")
        pg_password = os.getenv("POSTGRES_PASSWORD", "langgraph-change-me")
        pg_db = os.getenv("POSTGRES_DB", "langgraph")
        pg_dsn = f"postgresql://{pg_user}:{pg_password}@{pg_host}:{pg_port}/{pg_db}"
    
    logger.info(f"Initializing PostgresSaver with DSN: {pg_dsn.replace(pg_password, '***')}")
    checkpointer = PostgresSaver.from_conn_string(pg_dsn)
    
    # Setup tables (idempotent)
    try:
        checkpointer.setup()
        logger.info("PostgresSaver tables initialized")
    except Exception as e:
        logger.warning(f"Failed to setup PostgresSaver tables (may already exist): {e}")
    
    return checkpointer


async def emit_event(event_type: str, node: Optional[str] = None, checkpoint_id: Optional[str] = None, payload: Dict[str, Any] = None):
    """Emit an event to the backend."""
    if not _backend_url or not _run_id:
        logger.debug("Skipping event emission (backend_url or run_id not set)")
        return
    
    if payload is None:
        payload = {}
    
    import httpx
    from datetime import datetime
    
    event = {
        "run_id": _run_id,
        "seq": int(datetime.now().timestamp() * 1000),  # Simple sequence number
        "ts": datetime.now().isoformat(),
        "type": event_type,
        "node": node,
        "checkpoint_id": checkpoint_id,
        "payload": payload,
    }
    
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            await client.post(
                f"{_backend_url}/projects/{os.getenv('PROJECT', 'default')}/runs/{_run_id}/events",
                json=event,
            )
        logger.debug(f"Emitted event: {event_type} for node {node}")
    except Exception as e:
        logger.warning(f"Failed to emit event to backend: {e}")


async def stream_graph(inputs: Dict[str, Any], run_id: str):
    """Stream graph execution and emit events."""
    global _graph, _checkpointer, _run_id
    
    try:
        config = {"configurable": {"thread_id": run_id}}
        
        await emit_event("node_start", payload={"inputs": inputs})
        
        # Stream graph execution
        async for event in _graph.astream(inputs, config, stream_mode="updates"):
            for node_name, node_data in event.items():
                await emit_event("node_update", node=node_name, payload={"data": node_data})
                
                # Check if this node has an interrupt
                state = await _graph.aget_state(config)
                if state.next and state.next != ():
                    # Check if any node in next is waiting for interrupt
                    for next_node in state.next:
                        # If graph is paused, emit interrupt event
                        if state.metadata and state.metadata.get("source") == "interrupt":
                            checkpoint_id = state.config.get("configurable", {}).get("checkpoint_id")
                            await emit_event("interrupt", node=next_node, checkpoint_id=checkpoint_id, payload={
                                "state": state.values,
                            })
                            return  # Pause execution
        
        await emit_event("node_end", payload={"completed": True})
        
    except Exception as e:
        logger.error(f"Error streaming graph: {e}", exc_info=True)
        await emit_event("error", payload={"error": str(e)})
        raise


@app.get("/health")
async def health():
    """Health check endpoint."""
    return {"status": "healthy"}


@app.get("/ready")
async def ready():
    """Readiness check endpoint."""
    # Graph loads lazily on first /start, so we're ready if server is running
    return {"status": "ready"}


@app.post("/start")
async def start(request: StartRequest, background_tasks: BackgroundTasks):
    """Start a workflow run."""
    global _graph, _checkpointer, _run_id, _artifacts_dir, _backend_url, _stream_task
    
    if _graph is None:
        # Load graph on first start
        graph_entry = os.getenv("GRAPH_ENTRY")
        if not graph_entry:
            raise HTTPException(status_code=400, detail="GRAPH_ENTRY environment variable not set")
        
        try:
            _graph = load_graph(graph_entry)
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e))
        
        # Initialize checkpointer
        _checkpointer = initialize_checkpointer()
        
        # Compile graph with checkpointer if not already compiled
        if not hasattr(_graph, "astream"):
            # Assume it's a workflow that needs compilation
            _graph = _graph.compile(checkpointer=_checkpointer)
        elif hasattr(_graph, "compile") and _checkpointer:
            # Recompile with checkpointer
            _graph = _graph.compile(checkpointer=_checkpointer)
    
    _run_id = request.run_id
    _artifacts_dir = os.getenv("ARTIFACTS_DIR", "/workspace/artifacts")
    _backend_url = os.getenv("BACKEND_API_URL", "http://backend-service:8080/api")
    
    # Ensure artifacts directory exists
    Path(_artifacts_dir).mkdir(parents=True, exist_ok=True)
    
    # Start streaming in background
    _stream_task = asyncio.create_task(stream_graph(request.inputs, request.run_id))
    
    return {"status": "started", "run_id": request.run_id}


@app.post("/resume")
async def resume(request: ResumeRequest):
    """Resume a workflow from an interrupt."""
    global _graph, _checkpointer, _run_id, _stream_task
    
    # Load graph if not already loaded (for pod restart scenario)
    if _graph is None:
        graph_entry = os.getenv("GRAPH_ENTRY")
        if not graph_entry:
            raise HTTPException(status_code=400, detail="GRAPH_ENTRY environment variable not set")
        
        try:
            _graph = load_graph(graph_entry)
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e))
        
        # Initialize checkpointer
        _checkpointer = initialize_checkpointer()
        
        # Compile graph with checkpointer if not already compiled
        if not hasattr(_graph, "astream"):
            _graph = _graph.compile(checkpointer=_checkpointer)
        elif hasattr(_graph, "compile") and _checkpointer:
            _graph = _graph.compile(checkpointer=_checkpointer)
    
    # Use run_id from request if not set (for pod restart)
    if not _run_id:
        # Extract run_id from checkpoint_id if it contains it, otherwise use checkpoint_id as-is
        # Checkpoint IDs from LangGraph are typically in format "thread_id:checkpoint_id"
        # For now, assume run_id is the session name from env or checkpoint_id
        _run_id = os.getenv("RUN_ID", request.checkpoint_id.split(":")[0] if ":" in request.checkpoint_id else request.checkpoint_id)
    
    _artifacts_dir = os.getenv("ARTIFACTS_DIR", "/workspace/artifacts")
    _backend_url = os.getenv("BACKEND_API_URL", "http://backend-service:8080/api")
    
    config = {
        "configurable": {
            "thread_id": _run_id,
            "checkpoint_id": request.checkpoint_id,
        }
    }
    
    try:
        # Get current state first to verify checkpoint exists
        state = await _graph.aget_state(config)
        
        # Update state with decision values
        await _graph.aupdate_state(config, values=request.values)
        
        # Get updated state to continue streaming
        state = await _graph.aget_state(config)
        
        # Continue streaming from current state
        _stream_task = asyncio.create_task(stream_graph(state.values, _run_id))
        
        return {"status": "resumed", "run_id": _run_id, "checkpoint_id": request.checkpoint_id}
    except Exception as e:
        logger.error(f"Error resuming workflow: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/status")
async def status():
    """Get current workflow run status."""
    global _graph, _run_id
    
    if _graph is None:
        return {"status": "not_started"}
    
    if not _run_id:
        return {"status": "no_active_run"}
    
    try:
        config = {"configurable": {"thread_id": _run_id}}
        state = await _graph.aget_state(config)
        
        # Determine current node
        currentNode = None
        if state.next and len(state.next) > 0:
            currentNode = state.next[0]
        
        checkpoint_id = state.config.get("configurable", {}).get("checkpoint_id")
        
        return {
            "status": "running" if currentNode else "completed",
            "currentNode": currentNode,
            "checkpoint_id": checkpoint_id,
            "values": state.values,
        }
    except Exception as e:
        logger.error(f"Error getting status: {e}", exc_info=True)
        return {"status": "error", "error": str(e)}


@app.post("/stop")
async def stop():
    """Stop the current workflow run."""
    global _stream_task
    
    if _stream_task:
        _stream_task.cancel()
        try:
            await _stream_task
        except asyncio.CancelledError:
            pass
    
    return {"status": "stopped"}


if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", "8000"))
    uvicorn.run(app, host="0.0.0.0", port=port)

