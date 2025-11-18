FROM registry.access.redhat.com/ubi9/python-311@sha256:d0b35f779ca0ae87deaf17cd1923461904f52d3ef249a53dbd487e02bdabdde6

USER 0

RUN npm install -g @anthropic-ai/claude-code

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

# OpenShift compatibility
RUN chmod -R g=u /app && chmod -R g=u /usr/local && chmod g=u /etc/passwd

USER 1001

# Default command - run via runner-shell
CMD ["python", "/app/claude-runner/wrapper.py"]
