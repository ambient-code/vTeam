# CLAUDE.md - Python Project

This file provides guidance to Claude Code (claude.ai/code) when working with this Python project.

## Development Commands

### Environment Setup
```bash
# Create virtual environment
uv venv
source .venv/bin/activate  # Linux/macOS
# .venv\Scripts\activate   # Windows

# Install dependencies
uv pip install -r requirements.txt
uv pip install -r requirements-dev.txt  # If dev requirements exist
```

### Code Quality
```bash
# Format code
black .

# Sort imports  
isort .

# Lint code
flake8 .

# Type checking (if using mypy)
mypy .
```

### Testing
```bash
# Run all tests
python -m pytest

# Run with coverage
python -m pytest --cov=.

# Run specific test file
python -m pytest tests/test_example.py

# Run specific test
python -m pytest tests/test_example.py::test_function_name
```

### Package Management
```bash
# Add new dependency
uv add package-name

# Add development dependency  
uv add --dev package-name

# Update dependencies
uv lock --upgrade
```

## Project Architecture

<!-- Describe your project's key components, modules, and architecture here -->

## Configuration

### Python Version
- Target: Python 3.11+ (latest stable and N-1)

### Code Style
- Line length: 88 characters (black default)
- Import sorting: isort with black compatibility
- Linting: flake8 with project-specific rules

### Testing Framework
- Test runner: pytest
- Coverage tool: pytest-cov
- Test location: tests/ directory

## Pre-commit Requirements

Before any commit, ALWAYS run:
1. `black .`
2. `isort .` 
3. `flake8 .`
4. `python -m pytest`

All commands must pass without errors or warnings.