# Hands-On Labs

Welcome to the Ambient Code Platform hands-on learning labs! These practical exercises will guide you through mastering AI-powered automation using AgenticSessions in a Kubernetes-native environment.

## Lab 1: Your First Agentic Session

This foundational lab introduces you to the platform's core workflow by creating and monitoring an AgenticSession. You'll learn how to configure sessions, monitor execution, and interpret results.

**[Start Lab 1 →](basic/lab-1-first-rfe.md)**

**Time**: 30-45 minutes
**Level**: Beginner
**Prerequisites**: Completed [Getting Started Guide](../user-guide/getting-started.md)

### What You'll Learn

- Create AgenticSessions using the web interface
- Understand interactive vs headless execution modes
- Configure single-repo and multi-repo sessions
- Monitor real-time session execution and status
- Review session results and Kubernetes resource lifecycle
- Troubleshoot common issues

### Lab Scenario

You'll automate code analysis and documentation generation tasks by creating AgenticSessions that:
- Analyze a Python repository and generate README documentation
- Perform interactive refactoring conversations
- Compare patterns across multiple repositories

## Lab Format

Each lab follows this structure:

### **Objective** 🎯
Clear learning goals and expected outcomes

### **Prerequisites** 📋
Required knowledge, tools, and setup before starting

### **Estimated Time** ⏱️
Realistic time commitment for completion

### **Step-by-Step Instructions** 📝
Detailed procedures with code examples and validation checkpoints

### **Troubleshooting** 🛠️
Common issues and solutions

### **Key Learnings** 📚
Summary of concepts mastered

## Prerequisites

Before starting Lab 1, ensure you have:

- [ ] **Ambient Code Platform installed and running** - Complete [Getting Started Guide](../user-guide/getting-started.md)
- [ ] **Anthropic API key** configured in ProjectSettings
- [ ] **At least one project** created
- [ ] **Web browser** for accessing the platform interface
- [ ] **Basic Git familiarity** (optional, for multi-repo exercises)

## Lab Environment Setup

### Local Development Setup

```bash
# Clone repository
git clone https://github.com/ambient-code/platform.git
cd platform

# Start local development environment (OpenShift Local/CRC)
make dev-start

# Access the frontend
echo "https://$(oc get route vteam-frontend -n vteam-dev -o jsonpath='{.spec.host}')"
```

See the [Getting Started Guide](../user-guide/getting-started.md) for detailed deployment instructions.

## Skills You'll Develop

### **Technical Skills**
- Kubernetes Custom Resource management (AgenticSessions, ProjectSettings)
- REST API usage for session lifecycle management
- Kubernetes CLI operations (kubectl/oc)
- Multi-repository workflows

### **AI Automation Skills**
- Writing effective prompts for code analysis and generation
- Understanding AI agent execution models
- Monitoring long-running AI tasks
- Interpreting AI-generated results

### **DevOps Skills**
- Container orchestration with Kubernetes
- Job-based execution patterns
- Secret management for API keys
- Resource monitoring and troubleshooting

## Success Criteria

After completing Lab 1, you should be able to:

- [ ] Create AgenticSessions via web UI and understand the underlying Kubernetes resources
- [ ] Choose appropriate session modes (interactive vs headless) for different tasks
- [ ] Configure single-repo and multi-repo sessions
- [ ] Monitor session execution using both UI and CLI
- [ ] Troubleshoot common session failures
- [ ] Interpret session results and status information
- [ ] Explain the platform's Kubernetes-native architecture

## Getting Help

### During the Lab

- **Stuck on a step?** Check the troubleshooting section in Lab 1
- **Unexpected results?** Verify your prerequisites and environment setup
- **Technical issues?** Reference the [Getting Started troubleshooting](../user-guide/getting-started.md#common-issues)

### Community Support

- **Questions about labs**: [GitHub Discussions](https://github.com/ambient-code/platform/discussions)
- **Bug reports**: [GitHub Issues](https://github.com/ambient-code/platform/issues)
- **Lab improvements**: Submit pull requests with your suggestions

## Next Steps After Lab 1

Once you've completed Lab 1, explore advanced AgenticSession capabilities:

- **Multi-repo patterns**: Experiment with cross-repository analysis and migration workflows
- **Interactive sessions**: Build iterative development workflows using inbox/outbox communication
- **Custom ProjectSettings**: Configure default models, timeouts, and team-specific settings
- **API integration**: Automate session creation via REST API for CI/CD pipelines
- **CLAUDE.md exploration**: Deep-dive into the complete AgenticSession specification and backend architecture

## Ready to Start?

**[Begin Lab 1: Your First Agentic Session →](basic/lab-1-first-rfe.md)**

Learn by doing! This lab provides hands-on experience with the platform's core capabilities in a safe, local development environment.
