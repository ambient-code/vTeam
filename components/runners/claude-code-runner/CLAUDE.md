# Ambient Agentic Runner Project

## Project Overview
This is an automated agentic system that uses Ambient Code AI CLI with Playwright MCP server integration and spek-kit spec-driven development to perform comprehensive tasks including website analysis, automation, data processing, and specification creation.

## Key Components
- **Python Runner**: Orchestrates the agentic session and handles status updates
- **AI CLI**: Executes tasks with integrated MCP server capabilities
- **Playwright MCP Server**: Provides headless browser tools via MCP protocol for navigation, screenshots, content extraction
- **Spek-kit Integration**: Enables spec-driven development with `/specify`, `/plan`, and `/tasks` commands
- **Backend Integration**: Reports progress and results to Kubernetes agentic session API

## Agentic Capabilities

### Browser Automation (Playwright MCP)
Ambient AI has access to advanced browser automation tools through the integrated Playwright MCP server:
- Navigate to websites and handle loading/timeouts
- Take screenshots for visual analysis
- Extract text content and metadata
- Interact with forms and page elements
- Handle dynamic content and SPAs
- Multi-step browsing and analysis

### Spec-Driven Development (spek-kit)
The runner supports spec-driven development workflows with these commands:
- **`/specify [requirements]`**: Create comprehensive feature specifications from natural language requirements
- **`/plan [tech details]`**: Generate detailed implementation plans with architecture and tech stack decisions
- **`/tasks [implementation notes]`**: Break down features into actionable development tasks with effort estimates

## Instructions for Sessions

### Standard Agentic Sessions (Website Analysis)
When executing website analysis tasks:
1. **Always start with navigation** - go to the target website first
2. **Take a screenshot** - capture visual state for reference
3. **Extract comprehensive content** - get all text, links, metadata
4. **Be methodical** - explore different sections if relevant
5. **Provide detailed analysis** - comprehensive findings aligned with task objectives
6. **Include actionable insights** - practical recommendations based on findings

### Spec-Driven Development Sessions
When using spek-kit commands:
1. **Start with /specify** - create the feature specification first
2. **Follow with /plan** - define technical implementation approach
3. **Complete with /tasks** - break down into actionable development tasks
4. **Review generated artifacts** - specifications, plans, and task breakdowns are automatically created
5. **Enhance with Claude Code** - the AI will improve and elaborate on generated content

## Technical Notes
- Container runs AI CLI with integrated Playwright MCP server and spek-kit
- Chrome runs headless with optimized flags via MCP server
- Spek-kit workspace created in `/tmp/spek-workspace` for each session
- Direct prompt passing eliminates temporary file creation
- Timeout handling: 5-minute default per agentic session
- Vision capabilities enabled for screenshot analysis through MCP
- Generated specifications, plans, and tasks returned as session artifacts

## Example Usage

### Website Analysis
```
Prompt: "Analyze the pricing page of example.com and summarize the pricing tiers"
Result: Complete analysis with screenshots and extracted pricing information
```

### Spec-Driven Development
```
Prompt: "/specify Build a user authentication system with email/password login, social auth, and password reset functionality"
Result: Complete specification document with user stories, requirements, and acceptance criteria

Prompt: "/plan Use Node.js with Express, PostgreSQL database, and React frontend. Include JWT for authentication and SendGrid for emails"
Result: Detailed implementation plan with architecture, phases, and technical decisions

Prompt: "/tasks Focus on the backend API implementation first, then frontend components"
Result: Granular task breakdown with effort estimates and dependencies
```
