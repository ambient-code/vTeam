# vTeam Documentation Strategy with MkDocs

## Current State Analysis

### Strengths

- **Excellent README.md**: Comprehensive overview with clear getting started instructions
- **Detailed ARCHITECTURE.md**: Production-ready LlamaDeploy architecture documentation  
- **Comprehensive Agent Framework**: Complete agent personas in `rhoai-ux-agents-vTeam.md`
- **Individual Agent Specs**: Detailed agent definitions in `agents/` directory
- **Technical Depth**: Strong coverage of RAT system, Ambient Agentic Runner, and vTeam tools
- **Component Documentation**: Detailed setup guides for ambient-runner platform

### Gaps Identified

- No structured documentation hierarchy for different audiences
- Scattered user guides across multiple files
- Missing hands-on lab exercises for learning
- No clear developer onboarding path
- Documentation not discoverable or searchable

## Proposed MkDocs Structure

```
docs/
├── index.md                 # Landing page (curated from README)
├── user-guide/             # End-user documentation
│   ├── index.md            # User guide overview
│   ├── getting-started.md  # Quick 5-minute setup
│   ├── creating-rfes.md    # RFE creation walkthrough
│   ├── agent-framework.md  # Working with AI agents
│   ├── agentic-runner.md   # Using Ambient Agentic Runner
│   ├── configuration.md    # Settings and customization
│   └── troubleshooting.md  # Common issues and solutions
├── components/             # Component-specific documentation
│   ├── ambient-runner/     # Ambient Agentic Runner docs
│   │   ├── setup.md        # Kubernetes deployment guide
│   │   ├── api.md         # REST API reference
│   │   └── proposals/      # Enhancement proposals (CREPs)
├── developer-guide/        # Developer/contributor documentation
│   ├── index.md           # Developer guide overview
│   ├── setup.md           # Development environment setup
│   ├── architecture.md    # Deep technical architecture (from existing)
│   ├── plugin-development.md # Extending the system
│   ├── api-reference.md   # API documentation
│   ├── contributing.md    # Contribution guidelines
│   └── testing.md         # Testing strategies and examples
├── labs/                   # Hands-on learning exercises
│   ├── index.md           # Labs overview and prerequisites
│   ├── basic/             # Foundational exercises
│   │   ├── lab-1-first-rfe.md
│   │   ├── lab-2-agent-interaction.md
│   │   └── lab-3-workflow-basics.md
│   ├── advanced/          # Complex scenarios
│   │   ├── lab-4-custom-agents.md
│   │   ├── lab-5-workflow-modification.md
│   │   └── lab-6-integration-testing.md
│   ├── production/        # Enterprise deployment
│   │   ├── lab-7-jira-integration.md
│   │   ├── lab-8-openshift-deployment.md
│   │   └── lab-9-scaling-optimization.md
│   └── solutions/         # Lab solutions and explanations
│       ├── solutions-basic.md
│       ├── solutions-advanced.md
│       └── solutions-production.md
├── reference/             # Reference documentation
│   ├── index.md          # Reference overview  
│   ├── agent-personas.md # Complete agent specifications
│   ├── api-endpoints.md  # REST API reference
│   ├── configuration-schema.md # Config file schemas
│   └── glossary.md       # Terms and definitions
└── assets/               # Shared resources
    ├── images/           # Screenshots, diagrams
    ├── diagrams/         # Architecture diagrams
    └── videos/           # Tutorial videos (future)
```

## MkDocs Configuration (mkdocs.yml)

```yaml
site_name: vTeam Documentation
site_description: AI-powered automation system for engineering refinement
site_author: Red Hat AI Engineering Team
site_url: https://vteam-docs.example.com

repo_name: ambient-code/vTeam
repo_url: https://github.com/ambient-code/vTeam
edit_uri: edit/main/docs/

theme:
  name: material
  palette:
    - scheme: default
      primary: red
      accent: red
      toggle:
        icon: material/brightness-7
        name: Switch to dark mode
    - scheme: slate
      primary: red  
      accent: red
      toggle:
        icon: material/brightness-4
        name: Switch to light mode
  features:
    - navigation.tabs
    - navigation.tabs.sticky
    - navigation.sections
    - navigation.expand
    - navigation.top
    - search.highlight
    - search.suggest
    - content.code.annotate

nav:
  - Home: index.md
  - User Guide:
    - user-guide/index.md
    - Getting Started: user-guide/getting-started.md
    - Creating RFEs: user-guide/creating-rfes.md
    - Agent Framework: user-guide/agent-framework.md
    - Configuration: user-guide/configuration.md
    - Troubleshooting: user-guide/troubleshooting.md
  - Developer Guide:
    - developer-guide/index.md
    - Setup: developer-guide/setup.md
    - Architecture: developer-guide/architecture.md
    - Plugin Development: developer-guide/plugin-development.md
    - API Reference: developer-guide/api-reference.md
    - Contributing: developer-guide/contributing.md
    - Testing: developer-guide/testing.md
  - Labs:
    - labs/index.md
    - Basic:
      - First RFE: labs/basic/lab-1-first-rfe.md
      - Agent Interaction: labs/basic/lab-2-agent-interaction.md
      - Workflow Basics: labs/basic/lab-3-workflow-basics.md
    - Advanced:
      - Custom Agents: labs/advanced/lab-4-custom-agents.md
      - Workflow Modification: labs/advanced/lab-5-workflow-modification.md
      - Integration Testing: labs/advanced/lab-6-integration-testing.md
    - Production:
      - Jira Integration: labs/production/lab-7-jira-integration.md
      - OpenShift Deployment: labs/production/lab-8-openshift-deployment.md
      - Scaling & Optimization: labs/production/lab-9-scaling-optimization.md
    - Solutions: 
      - Basic Solutions: labs/solutions/solutions-basic.md
      - Advanced Solutions: labs/solutions/solutions-advanced.md
      - Production Solutions: labs/solutions/solutions-production.md
  - Reference:
    - reference/index.md
    - Agent Personas: reference/agent-personas.md
    - API Endpoints: reference/api-endpoints.md
    - Configuration Schema: reference/configuration-schema.md
    - Glossary: reference/glossary.md

plugins:
  - search
  - mermaid2

markdown_extensions:
  - admonition
  - attr_list
  - codehilite
  - pymdownx.details
  - pymdownx.superfences:
      custom_fences:
        - name: mermaid
          class: mermaid
          format: !!python/name:pymdownx.superfences.fence_code_format
  - pymdownx.tabbed
  - toc:
      permalink: true
```

## Content Migration Strategy

### Phase 1: Foundation (Week 1)

1. **Setup MkDocs infrastructure**
   - Install MkDocs with Material theme
   - Configure `mkdocs.yml` with navigation structure
   - Setup CI/CD for automated builds

2. **Create core structure**
   - Migrate and curate README.md content for `docs/index.md`
   - Move ARCHITECTURE.md to `docs/developer-guide/architecture.md`
   - Create section index pages with clear navigation

### Phase 2: User Documentation (Week 2)

1. **Getting Started Guide**
   - 5-minute quick start (streamlined from README)
   - Prerequisites and installation
   - First RFE creation walkthrough

2. **User Guide Pages**
   - RFE creation workflows (conversational vs form-based)
   - Agent framework explanation (from `rhoai-ux-agents-vTeam.md`)
   - Configuration and customization options

### Phase 3: Developer Documentation (Week 3)

1. **Development Setup**
   - Environment setup and dependencies
   - Development workflow and standards
   - Testing procedures and CI/CD

2. **Technical Deep Dive**
   - Enhanced architecture documentation
   - Plugin development guide
   - API reference documentation

### Phase 4: Lab Exercises (Week 4)

1. **Basic Labs**
   - Lab 1: Create your first RFE using the chat interface
   - Lab 2: Understand agent interactions and workflows
   - Lab 3: Explore different workflow scenarios

2. **Advanced Labs**
   - Lab 4: Create custom agent personas
   - Lab 5: Modify workflows for specific use cases
   - Lab 6: Integration testing and validation

3. **Production Labs**
   - Lab 7: Set up Jira integration
   - Lab 8: Deploy to OpenShift
   - Lab 9: Performance optimization and scaling

## Lab Exercise Design Principles

### Structure

- **Objective**: Clear learning goals
- **Prerequisites**: Required knowledge and setup
- **Step-by-Step Instructions**: Detailed procedures
- **Expected Outcomes**: What users should achieve
- **Troubleshooting**: Common issues and solutions
- **Further Reading**: Links to relevant documentation

### Example Lab Template

```markdown
# Lab 1: Create Your First RFE

## Objective
Learn to create a Request for Enhancement (RFE) using the conversational AI interface.

## Prerequisites
- vTeam system running locally
- Basic understanding of software requirements
- Access to Anthropic Claude API

## Steps
1. Navigate to the chat interface
2. Describe your feature idea in natural language
3. Follow the AI prompts to refine your request
4. Review the generated RFE structure
5. Submit for agent council review

## Expected Outcomes
- Complete RFE with business justification
- Technical requirements identified
- Success criteria defined
- Agent workflow initiated

## Troubleshooting
- If API key errors occur, check ProjectSettings CR in your namespace
- For agent timeout issues, verify network connectivity and check operator logs

## Further Reading
- [Agent Framework Guide](../user-guide/agent-framework.md)
- [Configuration Reference](../user-guide/configuration.md)
```

## Integration with Existing Tools

### GitHub Integration

- **Source**: Documentation lives in `/docs` directory
- **Editing**: Direct GitHub editing links in MkDocs
- **Issues**: Link to GitHub issues for documentation bugs
- **Contributions**: Pull request workflow for doc updates

### CI/CD Pipeline

- **Build**: Automated MkDocs builds on push to main
- **Deploy**: GitHub Pages or internal hosting
- **Validation**: Link checking and markdown linting
- **Preview**: PR preview deployments

### Search and Discovery

- **Full-text search** via MkDocs search plugin
- **Cross-references** between sections
- **Glossary** for technical terms
- **Tag system** for categorizing content

## Success Metrics

### User Adoption

- **Time to first RFE**: Measure setup to first successful RFE creation
- **Documentation page views**: Track most accessed content
- **Lab completion rates**: Monitor learning engagement
- **Support ticket reduction**: Measure documentation effectiveness

### Developer Experience

- **Contribution velocity**: Track PR frequency and merge time
- **Setup time**: Measure development environment setup duration  
- **API usage**: Monitor developer API adoption
- **Community engagement**: Track discussions and questions

### Content Quality

- **Accuracy**: Regular technical review cycles
- **Freshness**: Automated checks for outdated content
- **Completeness**: Coverage analysis of features vs documentation
- **Accessibility**: Ensure documentation works for all users

## Maintenance Strategy

### Regular Updates

- **Monthly reviews** of user feedback and analytics
- **Quarterly content audits** for accuracy and relevance
- **Version alignment** with software releases
- **Link validation** and broken reference cleanup

### Community Contributions

- **Clear contribution guidelines** for documentation PRs
- **Template system** for consistent formatting
- **Review process** with subject matter experts
- **Recognition system** for documentation contributors

This comprehensive documentation strategy transforms vTeam from a technically excellent but scattered documentation system into a professional, discoverable, and learnable platform that serves both end users and developers effectively.
