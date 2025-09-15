This project, vTeam, is an AI-powered multi-agent system designed to streamline the engineering refinement process. It uses a "council" of AI agents to take a Request for Enhancement (RFE) from initial submission to a well-defined, implementation-ready ticket. The goal is to reduce meeting times and improve ticket quality.

The file structure is organized with the main application in `demos/rfe-builder/`, which is a Streamlit application. The core logic for the AI agents and workflow is in `demos/r-builder/src/`. Shared configurations for the vTeam agents are located in `src/vteam_shared_configs/`. The project uses Python, with dependencies managed in `pyproject.toml` and `requirements.txt` files.

To run the application, you need Python 3.12+ and to install the dependencies using `uv` or `pip`. The main application is run with `streamlit run app.py` from the `demos/rfe-builder/` directory. Configuration for API keys (Anthropic, Vertex AI, Jira) is done in a `.streamlit/secrets.toml` file.

For new developers, it's important to understand the agent council workflow, where each agent has a specific role in the refinement process. The `README.md` provides a good overview of this. The file `rhoai-ux-agents-vTeam.md` contains more detailed information about the agent framework.
