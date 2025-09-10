# Use Python 3.11 slim image
FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install system dependencies + debugging tools + browser dependencies
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    ca-certificates \
    gnupg \
    strace \
    procps \
    lsof \
    xvfb \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js 18+ (required for Claude Code)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs

# Copy requirements first for better Docker layer caching
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Install Claude Code CLI and MCP server for direct SDK usage
RUN npm install -g @anthropic-ai/claude-code @playwright/mcp

# Create a local package.json for proper Playwright dependency management
RUN cd /app && echo '{"name":"claude-runner","dependencies":{"playwright":"*","@playwright/mcp":"*"}}' > package.json && \
    npm install

# Install Playwright system dependencies and browsers as ROOT
RUN npx playwright install-deps chromium && \
    npx playwright install chromium


# Move browsers to tmp directory for OpenShift compatibility
RUN if [ -d /root/.cache/ms-playwright ]; then \
        cp -r /root/.cache/ms-playwright/* /tmp/.cache/ms-playwright/ 2>/dev/null || true; \
    fi && \
    chmod -R 777 /tmp/.cache/ms-playwright && \
    find /tmp/.cache/ms-playwright -name "chromium*" -type f -exec chmod +x {} \; 2>/dev/null || true

# Create startup script for virtual display (OpenShift compatible)
RUN echo '#!/bin/bash\n' \
         'export HOME=/app\n' \
         'export DISPLAY=:99\n' \
         'Xvfb :99 -screen 0 1024x768x24 -ac +extension GLX +render -noreset > /dev/null 2>&1 &\n' \
         'sleep 2\n' \
         'exec "$@"' > /usr/local/bin/start-xvfb && \
    chmod +x /usr/local/bin/start-xvfb

# Copy the application files with proper permissions
COPY main.py /app/
COPY CLAUDE.md /app/
RUN chmod 755 /app/main.py && chmod 644 /app/CLAUDE.md

# Set OpenShift-compatible environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV NODE_OPTIONS="--max-old-space-size=2048"
ENV HOME=/app
ENV SHELL=/bin/bash
ENV TERM=xterm-256color

# OpenShift runs with random UID - don't set USER directive
# Ensure /app directory is accessible to any UID
RUN chmod -R g=u /app && chmod -R g=u /usr/local && chmod g=u /etc/passwd

# Run the application with virtual display
CMD ["start-xvfb", "python", "main.py"]