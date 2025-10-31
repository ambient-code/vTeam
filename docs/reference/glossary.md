# Glossary

This glossary defines key terms, concepts, and acronyms used throughout the Ambient Code Platform system and documentation.

## Core Concepts

### **Agent Council**
The 7-member AI agent team that collaboratively reviews and refines RFEs. Each agent has specialized expertise and realistic seniority levels matching real software teams.

### **Agent Persona**
A specialized AI character with defined role, expertise, seniority level, and analysis framework. Examples include Parker (PM), Archie (Architect), and Stella (Staff Engineer).

### **Conversational RFE Creation**
The process of creating Requirements for Enhancement using natural language chat with AI, as opposed to filling out traditional forms.

### **Kubernetes Operator**
Custom controller that extends Kubernetes functionality by watching Custom Resources and reconciling them to desired state. The Ambient Code Platform uses an operator to manage agentic sessions.

### **Multi-Agent Workflow**  
The coordinated process where multiple AI agents sequentially analyze an RFE, each contributing specialized expertise to create comprehensive requirements.

### **RAG (Retrieval-Augmented Generation)**
AI technique that enhances agent responses by retrieving relevant information from knowledge bases before generating analysis.

### **RFE (Request for Enhancement)**
A structured document describing a desired software feature, including business justification, technical requirements, and success criteria.

### **Refinement Agent Team (RAT)**
The complete AI-powered system that automates engineering refinement processes, reducing meeting time and improving ticket quality.

### **Workflow Orchestration**
The automated management of task sequences, state transitions, and agent coordination within the Ambient Code Platform.

## Technical Terms

### **API Endpoint**
RESTful web service interface for programmatic access to platform functionality.

### **FAISS**
Facebook AI Similarity Search - vector database used for efficient document retrieval in RAG systems.

### **LlamaIndex** 
Framework for building RAG applications with document indexing and retrieval capabilities.

### **Pydantic**
Python library for data validation and serialization using type hints.

### **Next.js**
React-based web framework used for the platform's frontend, providing server-side rendering, routing, and modern UI components via Shadcn UI.

### **Vector Embedding**
Numerical representation of text that enables semantic similarity search in AI systems.

### **WebSocket**
Communication protocol enabling real-time bidirectional data exchange between client and server.

## Agent Roles & Personas

### **Archie (Architect)**
Principal-level AI agent responsible for technical feasibility assessment, architecture review, and design validation.

### **Derek (Delivery Owner)**  
Delivery Manager-level agent focused on implementation planning, ticket creation, and timeline estimation.

### **Lee (Team Lead)**
Engineering Manager-level agent handling team coordination, resource allocation, and execution oversight.

### **Olivia (Product Owner)**
Senior Product Owner agent managing acceptance criteria definition, scope validation, and stakeholder alignment.

### **Parker (Product Manager)**
Senior PM-level agent focused on business value assessment, prioritization, and stakeholder communication.

### **Stella (Staff Engineer)**
Staff-level engineering agent providing implementation complexity analysis and technical decision-making.

### **Taylor (Team Member)**
Senior Engineer-level agent handling detailed implementation considerations and development task breakdown.

## Workflow States

### **Agent Analysis**
Individual agent processing phase where a single agent analyzes the RFE from their specialized perspective.

### **Collaborative Review**
Multi-agent phase where agents build upon each other's analysis to create comprehensive requirements.

### **Criteria Refinement**
Process of improving and validating acceptance criteria for testability and completeness.

### **Decision Point**
Critical workflow stage where agents make approve/reject/needs-info recommendations.

### **Implementation Planning**
Final phase where approved RFEs are converted into actionable development tickets and timelines.

### **Stakeholder Communication**
Process of updating requestors and stakeholders on RFE status and next steps.

## Integration Terms

### **Anthropic Claude**
Primary large language model API used for agent intelligence and conversational capabilities.

### **GitHub Integration**
Connection to GitHub repositories for code context, documentation access, and issue management.

### **Jira Integration**
Connection to Atlassian Jira for automated epic and story creation from refined RFEs.

### **OpenAI Embeddings**
Text embedding service used for document similarity search in RAG systems.

### **Vertex AI**
Google Cloud AI platform providing alternative language model capabilities.

## Configuration Terms

### **Agent Configuration**
YAML-based definition specifying agent behavior, knowledge sources, and analysis prompts.

### **Data Sources**
External information sources (local files, GitHub repos, web pages) used to build agent knowledge bases.

### **Environment Variables**
System configuration settings for API keys, service endpoints, and runtime parameters.

### **Secrets Configuration**
Secure storage of API keys, authentication tokens, and sensitive system settings.

### **Template Variables**
Placeholder values in agent prompts that are replaced with actual content during analysis.

## Quality & Performance

### **Acceptance Criteria**
Specific, measurable, testable conditions that must be met for an RFE to be considered complete.

### **Business Value Score**
Numerical rating (1-10) assigned by PM agents to quantify the business impact of an RFE.

### **Complexity Rating**
Assessment (low/medium/high) of implementation difficulty and resource requirements.

### **Response Time**
Duration from RFE submission to complete agent analysis, target < 3 minutes for full workflow.

### **Token Limit**
Maximum amount of text content that can be processed by AI agents in a single analysis.

## Deployment Terms

### **Container Orchestration**
Management of containerized platform services using Docker and Kubernetes platforms.

### **Health Check**
Automated system monitoring endpoint that reports service status and availability.

### **Horizontal Scaling**
Adding more instances of platform services to handle increased load.

### **Load Balancing**
Distribution of requests across multiple platform service instances for optimal performance.

### **Production Deployment**
Enterprise-grade installation of the Ambient Code Platform with high availability, monitoring, and security.

## Development Terms

### **CI/CD (Continuous Integration/Continuous Deployment)**
Automated pipeline for testing, building, and deploying platform code changes.

### **Pre-commit Hooks**
Automated code quality checks that run before Git commits are allowed.

### **Test Coverage**
Percentage of code exercised by automated tests, target minimum 80% for new features.

### **Type Checking**
Static analysis to verify Python code type correctness using mypy.

### **Virtual Environment**
Isolated Python environment for managing project dependencies without system conflicts.

## Acronyms

### **AI** - Artificial Intelligence
### **API** - Application Programming Interface  
### **CD** - Continuous Deployment
### **CI** - Continuous Integration
### **CPU** - Central Processing Unit
### **CRUD** - Create, Read, Update, Delete
### **HTTP** - HyperText Transfer Protocol
### **JSON** - JavaScript Object Notation
### **LLM** - Large Language Model
### **MVP** - Minimum Viable Product
### **PM** - Product Manager / Product Management
### **PO** - Product Owner  
### **QA** - Quality Assurance
### **RAM** - Random Access Memory
### **REST** - Representational State Transfer
### **SLA** - Service Level Agreement
### **UI** - User Interface
### **URL** - Uniform Resource Locator
### **UX** - User Experience
### **YAML** - Yet Another Markup Language / YAML Ain't Markup Language

## Measurement Units

### **Story Points**
Relative estimation unit for development effort, typically using Fibonacci sequence (1, 2, 3, 5, 8, 13).

### **Sprint**  
Time-boxed development iteration, typically 1-2 weeks for RFE implementation.

### **Velocity**
Team's average story point completion rate per sprint, used for capacity planning.

---

## Contributing to the Glossary

Found a missing term or unclear definition? 

- **Submit a pull request** with new definitions
- **Create an issue** to suggest improvements  
- **Follow the format**: **Term** followed by clear, concise definition
- **Include context** where the term is commonly used
- **Cross-reference** related terms when helpful

This glossary is a living document that evolves with the Ambient Code Platform. Your contributions help make the platform more accessible to new users and contributors.