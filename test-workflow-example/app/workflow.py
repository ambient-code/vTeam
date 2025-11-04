#!/usr/bin/env python3
"""
Example LangGraph workflow for testing the LangGraph MVP system.
This workflow demonstrates:
- Simple state management
- Multiple nodes with dependencies
- Input/output handling
"""

from langgraph.graph import StateGraph, END
from typing import TypedDict, Annotated
from operator import add


class State(TypedDict):
    """Workflow state."""
    message: str
    step: int
    result: Annotated[str, add]  # Accumulate results
    counter: int


def build_app():
    """
    Build the workflow graph.
    
    This is the entry point that the LangGraph runner will call.
    Must return a compiled graph.
    """
    graph = StateGraph(State)
    
    def step_one(state: State) -> State:
        """First processing step."""
        msg = state.get("message", "default")
        return {
            "message": f"Step 1 processed: {msg}",
            "step": 1,
            "result": f"[Step 1] {msg}\n",
            "counter": state.get("counter", 0) + 1
        }
    
    def step_two(state: State) -> State:
        """Second processing step."""
        return {
            "message": state["message"],
            "step": 2,
            "result": state.get("result", "") + f"[Step 2] Processed message\n",
            "counter": state.get("counter", 0) + 1
        }
    
    def step_three(state: State) -> State:
        """Final step that produces output."""
        return {
            "message": state["message"],
            "step": 3,
            "result": state.get("result", "") + f"[Step 3] Final result: {state['message']}\n",
            "counter": state.get("counter", 0) + 1
        }
    
    # Add nodes
    graph.add_node("step_one", step_one)
    graph.add_node("step_two", step_two)
    graph.add_node("step_three", step_three)
    
    # Define flow
    graph.set_entry_point("step_one")
    graph.add_edge("step_one", "step_two")
    graph.add_edge("step_two", "step_three")
    graph.add_edge("step_three", END)
    
    # Compile and return
    return graph.compile()


# For testing locally
if __name__ == "__main__":
    import asyncio
    
    async def test():
        """Test the workflow locally."""
        app = build_app()
        result = await app.ainvoke({
            "message": "Hello from local test!",
            "step": 0,
            "result": "",
            "counter": 0
        })
        print("Workflow result:")
        print(result)
    
    asyncio.run(test())


