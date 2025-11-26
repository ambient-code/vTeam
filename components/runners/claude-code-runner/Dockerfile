FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    ca-certificates \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && npm install -g @anthropic-ai/claude-code \
    && rm -rf /var/lib/apt/lists/*

# Create working directory
WORKDIR /app

# Copy and install runner-shell package (expects build context at components/runners)
COPY runner-shell /app/runner-shell
RUN cd /app/runner-shell && pip install --no-cache-dir .

# Copy claude-runner specific files
COPY claude-code-runner /app/claude-runner

# Install runner wrapper as a package (pulls dependencies like claude-agent-sdk)
RUN pip install --no-cache-dir /app/claude-runner \
    && pip install --no-cache-dir aiofiles

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV RUNNER_TYPE=claude
ENV HOME=/app
ENV SHELL=/bin/bash
ENV TERM=xterm-256color

# Set umask to make files readable by other containers (fixes content service access)
# 0022 creates files as rw-r--r-- (644) instead of default rw------- (600)
RUN echo "umask 0022" >> /etc/profile && \
    echo "umask 0022" >> /root/.bashrc

# OpenShift compatibility
RUN chmod -R g=u /app && chmod -R g=u /usr/local && chmod g=u /etc/passwd

# Run as UID 1001 to match content service (fixes permission issues)
USER 1001

# Default command - run via runner-shell
CMD ["/bin/bash", "-c", "umask 0022 && python /app/claude-runner/wrapper.py"]
