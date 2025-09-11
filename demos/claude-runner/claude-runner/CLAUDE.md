# Claude Research Runner Project

## Project Overview
This is an automated web research system that uses Claude Code CLI with Playwright MCP server integration to perform comprehensive website analysis.

## Key Components
- **Python Runner**: Orchestrates the research session and handles status updates
- **Claude Code CLI**: Executes research with integrated MCP server capabilities  
- **Playwright MCP Server**: Provides headless browser tools via MCP protocol for navigation, screenshots, content extraction
- **Backend Integration**: Reports progress and results to Kubernetes research session API

## Research Capabilities
Claude Code has access to advanced browser automation tools through the integrated Playwright MCP server:
- Navigate to websites and handle loading/timeouts
- Take screenshots for visual analysis 
- Extract text content and metadata
- Interact with forms and page elements
- Handle dynamic content and SPAs
- Multi-step browsing and analysis

## Instructions for Research Sessions
When conducting research:
1. **Always start with navigation** - go to the target website first
2. **Take a screenshot** - capture visual state for reference
3. **Extract comprehensive content** - get all text, links, metadata
4. **Be methodical** - explore different sections if relevant
5. **Provide detailed analysis** - comprehensive findings aligned with research objectives
6. **Include actionable insights** - practical recommendations based on findings

## Technical Notes
- Container runs Claude Code CLI with integrated Playwright MCP server
- Chrome runs headless with optimized flags via MCP server
- Direct prompt passing eliminates temporary file creation
- Timeout handling: 5-minute default per research session
- Vision capabilities enabled for screenshot analysis through MCP
