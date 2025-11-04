FROM quay.io/gkrumbach07/langgraph-wrapper:base

WORKDIR /app/workflow

# Copy workflow code
COPY app/ ./app/
COPY requirements.txt ./

# Install workflow-specific dependencies (if any)
# The base image already includes langgraph, but we can add more here
RUN pip install --no-cache-dir -r requirements.txt || true

# Ensure app module is importable
ENV PYTHONPATH=/app/workflow:$PYTHONPATH

# Verify import works at build time
RUN python -c "import app.workflow; print('✅ Workflow import successful')"
