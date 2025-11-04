FROM registry.access.redhat.com/ubi9/python-311:latest

USER 0

WORKDIR /app

# Install system dependencies
RUN dnf install -y git procps && dnf clean all

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy runner server
COPY runner/ ./runner/

# Set environment variables
ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1

# Expose port for FastAPI server
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD python -c "import requests; requests.get('http://localhost:8000/health')" || exit 1

# Run the server
CMD ["uvicorn", "runner.server:app", "--host", "0.0.0.0", "--port", "8000"]

