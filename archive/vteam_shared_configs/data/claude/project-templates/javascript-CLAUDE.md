# CLAUDE.md - JavaScript/Node.js Project

This file provides guidance to Claude Code (claude.ai/code) when working with this JavaScript project.

## Development Commands

### Environment Setup
```bash
# Install dependencies
npm install
# or
yarn install

# Install development dependencies
npm install --only=dev
```

### Code Quality
```bash
# Format code
npm run format
# or 
npx prettier --write .

# Lint code
npm run lint
# or
npx eslint .

# Fix auto-fixable lint issues
npm run lint:fix
# or
npx eslint . --fix
```

### Testing
```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage

# Run specific test file
npm test -- test-file.test.js
```

### Build and Development
```bash
# Start development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

## Project Architecture

<!-- Describe your project's key components, structure, and architecture here -->

## Configuration

### Node.js Version
- Target: Node.js 18+ (LTS and current)

### Code Style
- Formatter: Prettier
- Linter: ESLint
- Line length: 80-100 characters
- Semicolons: Consistent with project preference

### Testing Framework
- Test runner: Jest/Vitest/Mocha (specify which)
- Coverage tool: Built-in coverage
- Test location: tests/ or __tests__/ directory

## Pre-commit Requirements

Before any commit, ALWAYS run:
1. `npm run format` (or `npx prettier --write .`)
2. `npm run lint` (or `npx eslint .`)
3. `npm test`

All commands must pass without errors or warnings.