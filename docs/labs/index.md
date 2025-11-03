# Labs & Exercises

Welcome to the vTeam hands-on learning labs! These practical exercises will guide you through mastering the Refinement Agent Team system, from basic RFE creation to advanced production deployments.

## Learning Path

Our lab curriculum is designed for progressive skill building:

### 🎯 Basic Labs (30-45 minutes each)

Perfect for getting started with vTeam fundamentals.

- **[Lab 1: Create Your First RFE](basic/lab-1-first-rfe.md)**  
  Learn the conversational AI interface and basic RFE creation workflow
  
- **[Lab 2: Agent Interaction Deep Dive](basic/lab-2-agent-interaction.md)**  
  Understand how the 7-agent council processes and refines your requests
  
- **[Lab 3: Workflow Management](basic/lab-3-workflow-basics.md)**  
  Master workflow states, progress tracking, and result interpretation

### 🔧 Advanced Labs (60-90 minutes each)  

For users ready to customize and extend vTeam capabilities.

- **[Lab 4: Custom Agent Creation](advanced/lab-4-custom-agents.md)**  
  Build specialized agent personas for your domain expertise
  
- **[Lab 5: Workflow Modification](advanced/lab-5-workflow-modification.md)**  
  Adapt the refinement process to your team's unique needs
  
- **[Lab 6: Integration Testing](advanced/lab-6-integration-testing.md)**  
  Validate custom configurations and ensure system reliability

### 🚀 Production Labs (90-120 minutes each)

Enterprise deployment and scaling considerations.

- **[Lab 7: Jira Integration Setup](production/lab-7-jira-integration.md)**  
  Connect vTeam to your existing project management workflow
  
- **[Lab 8: OpenShift Deployment](production/lab-8-openshift-deployment.md)**  
  Deploy vTeam in a production Kubernetes environment
  
- **[Lab 9: Scaling & Optimization](production/lab-9-scaling-optimization.md)**  
  Performance tuning, monitoring, and high-availability setup

## Lab Format

Each lab follows a consistent structure for optimal learning:

### **Objective** 🎯

Clear learning goals and expected outcomes

### **Prerequisites** 📋

Required knowledge, tools, and setup before starting

### **Estimated Time** ⏱️

Realistic time commitment for completion

### **Step-by-Step Instructions** 📝

Detailed procedures with code examples and screenshots

### **Validation Checkpoints** ✅

Verify your progress at key milestones

### **Troubleshooting** 🛠️

Common issues and solutions

### **Further Exploration** 🔍

Additional resources and next steps

## Prerequisites

Before starting the labs, ensure you have:

- [ ] **vTeam installed and working** - Complete [Getting Started Guide](../user-guide/getting-started.md)
- [ ] **Basic understanding** of software requirements and agile processes
- [ ] **Anthropic Claude API access** with available credits
- [ ] **Python development environment** (for advanced labs)
- [ ] **Git familiarity** for version control operations

## Lab Environment Setup

### Recommended Setup

```bash
# Clone vTeam repository
git clone https://github.com/ambient-code/vTeam.git
cd vTeam

# Follow the deployment guide for your environment
# See: docs/user-guide/getting-started.md for detailed instructions
```

### Lab-Specific Data

Some labs include sample data and configurations:

```
labs/
├── data/                       # Sample datasets
│   ├── sample-rfes.json       # Example RFE submissions
│   ├── agent-configs/         # Custom agent examples
│   └── integration-tests/     # Test scenarios
├── solutions/                  # Complete lab solutions
└── assets/                    # Screenshots and diagrams
```

## Skills You'll Develop

### **Product Management Skills**

- Writing clear, actionable requirements
- Collaborating with AI agents for requirement refinement
- Stakeholder communication through agent interactions

### **Technical Skills**

- Python development for agent customization
- YAML configuration for agent personas
- REST API integration and testing
- Docker and Kubernetes deployment

### **Process Skills**

- Agile refinement best practices
- Workflow optimization and measurement
- Quality assurance for AI-generated content

## Success Metrics

Track your learning progress:

- **Basic Labs**: Successfully create and refine RFEs using the agent workflow
- **Advanced Labs**: Build and deploy custom agent configurations  
- **Production Labs**: Implement enterprise-ready vTeam deployments

### Self-Assessment Checklist

After completing the lab series, you should be able to:

- [ ] Create comprehensive RFEs using conversational AI
- [ ] Understand and interpret agent analysis results
- [ ] Customize agent personas for your domain
- [ ] Modify workflows to match team processes
- [ ] Deploy vTeam in production environments
- [ ] Troubleshoot common integration issues
- [ ] Optimize system performance and reliability

## Getting Help

### During Labs

- **Stuck on a step?** Check the troubleshooting section in each lab
- **Unexpected results?** Compare with provided solution examples
- **Technical issues?** Reference the [User Guide Troubleshooting](../user-guide/troubleshooting.md)

### Community Support

- **Questions about labs**: [GitHub Discussions](https://github.com/ambient-code/vTeam/discussions)
- **Bug reports**: [GitHub Issues](https://github.com/ambient-code/vTeam/issues)
- **Lab improvements**: Submit pull requests with your suggestions

## Solutions & Answers

Complete solutions are available after you've attempted each lab:

- **[Basic Lab Solutions](solutions/solutions-basic.md)** - Full walkthroughs and explanations
- **[Advanced Lab Solutions](solutions/solutions-advanced.md)** - Code examples and configurations  
- **[Production Lab Solutions](solutions/solutions-production.md)** - Deployment templates and scripts

!!! tip "Learning Best Practice"
    Try to complete each lab independently before consulting the solutions. The learning comes from working through challenges!

## Choose Your Starting Point

- **Never used vTeam before?** → Start with [Lab 1: First RFE](basic/lab-1-first-rfe.md)
- **Familiar with basics?** → Jump to [Lab 4: Custom Agents](advanced/lab-4-custom-agents.md)
- **Ready for production?** → Begin with [Lab 7: Jira Integration](production/lab-7-jira-integration.md)

Let's start building your vTeam expertise! 🚀
