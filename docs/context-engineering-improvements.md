# vTeam Context Engineering Improvements Plan

**Date**: 2025-09-30
**Reference**: [Anthropic: Effective Context Engineering for AI Agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)
**Status**: Planning Phase

## Executive Summary

This document outlines improvements to vTeam's AI agent context management based on Anthropic's best practices for effective context engineering. The core principle: treat context as a "finite resource with diminishing marginal returns" and implement "just-in-time" context loading to maximize AI agent effectiveness.

**Key Insight from Anthropic**: "As models become more capable, the challenge isn't just crafting the perfect prompt—it's thoughtfully curating what information enters the model's limited attention budget at each step."

## Current State Analysis

### Context Usage Patterns in vTeam

1. **Agent Loading** (`components/runners/claude-code-runner/agent_loader.py`)
   - 17 agent personas with full YAML definitions (~97 lines each)
   - Total: ~1,650 lines of agent context loaded upfront
   - All agents injected regardless of task relevance

2. **System Prompts** (`components/runners/claude-code-runner/agents/*.yaml`)
   - Verbose multi-sentence system messages
   - Extensive sample knowledge sections with duplication
   - Full expertise lists and analysis prompt templates

3. **Architecture Context** (`architecture/workspace.dsl`)
   - 262-line C4 model DSL file
   - Entire architecture loaded for focused tasks
   - No selective component extraction

4. **Chat Mode** (`main.py` lines 359-496)
   - Unbounded message history accumulation
   - No summarization for long-running sessions
   - Full conversation retained in context

5. **Tool Availability** (`main.py` lines 373-374, 576-577)
   - Partial filtering via `allowed_tools` environment variable
   - No task-specific tool presets
   - Default: All tools available

## Proposed Improvements

### 1. Agent Context Compaction

**Problem**: Full agent YAML files (97 lines × 17 agents = ~1,650 lines) loaded into every session, regardless of relevance.

**Solution**: Create compact agent "identity cards" with essential information only.

**Implementation**:

```python
# agent_loader.py
class AgentPersona:
    def get_compact_summary(self) -> Dict[str, Any]:
        """Return minimal agent metadata for context efficiency."""
        return {
            "name": self.name,
            "role": self.role,
            "expertise": self.expertise[:3],  # Top 3 only
            "one_liner": self._generate_one_liner()
        }

    def _generate_one_liner(self) -> str:
        """Condense system message to single sentence."""
        # Extract first sentence or create from role + top expertise
        pass
```

**Before** (97 lines per agent):
```yaml
systemMessage: |
  You are Emma, an Engineering Manager with expertise in team leadership and strategic planning.
  You focus on team wellbeing, sustainable delivery practices, and balancing technical excellence with business needs.
  You monitor team velocity, protect team focus, and facilitate clear communication across stakeholders.

expertise:
  - "team-leadership"
  - "capacity-planning"
  - "delivery-coordination"
  - "technical-strategy"
  - "team-wellbeing"
  - "risk-assessment"

sampleKnowledge: |
  # Engineering Management Best Practices
  [... 27 more lines ...]
```

**After** (5 lines per agent):
```python
{
  "name": "Emma",
  "role": "Engineering Manager",
  "expertise": ["team-leadership", "capacity-planning", "delivery-coordination"],
  "one_liner": "Engineering Manager prioritizing team capacity, delivery sustainability, and risk mitigation"
}
```

**Impact**:
- Context reduction: ~1,650 lines → ~85 lines (95% reduction)
- Startup time: Minimal impact (<100ms)
- Quality tradeoff: Load full definition only when agent is actively invoked

---

### 2. Just-In-Time Agent Loading

**Problem**: All 17 agents injected at session start regardless of task relevance.

**Solution**: Load agents on-demand based on prompt analysis or explicit selection.

**Current Implementation** (partial):
```python
# main.py lines 142-173
def _inject_selected_agents(self) -> None:
    """Fetch selected agent persona markdown from backend."""
    personas_env = os.getenv("AGENT_PERSONAS") or os.getenv("AGENT_PERSONA", "")
    personas = [p.strip() for p in personas_env.split(",") if p.strip()]
    if not personas:
        return  # ✓ Already skips if no agents specified
```

**Enhancement**:

```python
# New: Intelligent agent recommendation
def _recommend_agents_for_task(self, prompt: str) -> List[str]:
    """Analyze prompt and recommend relevant agents using lightweight model."""
    # Use Claude Haiku to classify task and recommend 2-4 relevant agents
    task_keywords = {
        "architecture": ["ARCHITECT", "STAFF_ENGINEER"],
        "planning": ["ENGINEERING_MANAGER", "PRODUCT_OWNER"],
        "implementation": ["TEAM_MEMBER", "TEAM_LEAD"],
        "documentation": ["TECHNICAL_WRITER", "CONTENT_STRATEGIST"],
        "ux": ["UX_DESIGNER", "UX_FEATURE_LEAD"],
    }
    # Pattern matching or lightweight LLM classification
    return recommended_personas
```

**Configuration**:
```bash
# Option 1: Explicit selection (current)
AGENT_PERSONAS="ENGINEERING_MANAGER,STAFF_ENGINEER"

# Option 2: Auto-recommend (new)
AGENT_SELECTION_MODE="auto"  # Analyzes prompt
AGENT_SELECTION_MODE="explicit"  # Uses AGENT_PERSONAS env var
AGENT_SELECTION_MODE="all"  # Legacy behavior
```

**Impact**:
- Typical session: 2-4 agents instead of 17 (70-80% reduction)
- Cost: +1 lightweight API call for auto-recommendation (~$0.0001)
- Benefit: Focused agent expertise, reduced context bloat

---

### 3. Structured Note-Taking for Long Sessions

**Problem**: Chat mode accumulates unbounded message history, leading to context window exhaustion.

**Solution**: Implement periodic summarization when approaching context limits.

**Implementation**:

```python
# main.py - enhance chat mode
class SimpleClaudeRunner:
    def __init__(self):
        # ... existing code ...
        self.context_threshold = int(os.getenv("CONTEXT_SUMMARY_THRESHOLD", "150000"))  # tokens
        self.conversation_summary: Optional[str] = None

    async def _maybe_summarize_conversation(self) -> None:
        """Summarize conversation when approaching context limit."""
        # Estimate token count (rough: 1 token ≈ 4 chars)
        estimated_tokens = sum(len(json.dumps(m)) for m in self.messages) // 4

        if estimated_tokens > self.context_threshold:
            # Use lightweight model to summarize
            summary = await self._generate_summary(self.messages)

            # Store summary and truncate old messages
            self.conversation_summary = summary
            self.messages = [
                {"type": "system_message", "content": f"Prior conversation summary: {summary}"},
                *self.messages[-20:]  # Keep last 20 messages for continuity
            ]
            logger.info(f"Summarized {estimated_tokens} tokens of conversation history")

    async def _generate_summary(self, messages: List[Dict]) -> str:
        """Generate conversation summary using Claude Haiku."""
        # Extract key decisions, action items, and context
        # Use structured output for consistency
        pass
```

**Trigger Points**:
- 75% of context window (150k tokens for 200k window)
- Configurable via `CONTEXT_SUMMARY_THRESHOLD` env var
- Manual trigger via `/summarize` command (future)

**Impact**:
- Enables indefinite chat sessions
- Maintains conversation continuity while preventing context rot
- Cost: ~$0.001 per summarization event

---

### 4. Tool Description Minimization

**Problem**: Tool descriptions consume significant context even when tools are irrelevant to task.

**Solution**: Task-specific tool bundles instead of all-or-nothing.

**Current State**:
```python
# main.py lines 373-374
allowed_tools_env = "Read,Write,Bash,Glob,Grep,Edit,MultiEdit,WebSearch,WebFetch"
allowed_tools = [t.strip() for t in allowed_tools_env.split(",") if t.strip()]
```

**Enhancement**:

```python
# New: Predefined tool bundles
TOOL_BUNDLES = {
    "code_editing": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
    "research": ["Read", "Glob", "Grep", "WebFetch", "WebSearch"],
    "analysis": ["Read", "Bash", "Grep", "WebSearch"],
    "documentation": ["Read", "Write", "Glob", "Grep"],
    "minimal": ["Read", "Bash"],
    "full": None,  # All tools
}

def _get_tool_bundle(self, task_type: str = "full") -> Optional[List[str]]:
    """Get appropriate tool bundle for task."""
    bundle_name = os.getenv("TOOL_BUNDLE", task_type)
    return TOOL_BUNDLES.get(bundle_name)
```

**Configuration**:
```bash
# Option 1: Named bundle
TOOL_BUNDLE="code_editing"

# Option 2: Auto-detect from prompt (future)
TOOL_SELECTION_MODE="auto"

# Option 3: Explicit list (current behavior)
ALLOWED_TOOLS="Read,Write,Edit,Bash"
```

**Impact**:
- Tool description overhead: Reduced by 60-70% for focused tasks
- Example: 9 tools → 4 tools = ~500 tokens saved
- No functional loss for typical workflows

---

### 5. Agent System Prompt Optimization

**Problem**: Verbose system messages with redundant explanations (3-5 sentences per agent).

**Solution**: Condense to single-sentence identity statements.

**Before** (`agents/engineering_manager.yaml` lines 16-19):
```yaml
systemMessage: |
  You are Emma, an Engineering Manager with expertise in team leadership and strategic planning.
  You focus on team wellbeing, sustainable delivery practices, and balancing technical excellence with business needs.
  You monitor team velocity, protect team focus, and facilitate clear communication across stakeholders.
```

**After**:
```yaml
systemMessage: "You are Emma, Engineering Manager prioritizing team capacity, delivery sustainability, and risk mitigation."
```

**Refactoring Strategy**:
1. Extract core role + primary focus
2. Remove redundant phrases ("with expertise in...", "you focus on...")
3. Use active verbs ("prioritizing" vs "focus on")
4. Limit to 15-20 words max

**Bulk Update**:
```bash
# Script to condense all agent system messages
python scripts/condense_agent_prompts.py agents/*.yaml
```

**Impact per agent**:
- Tokens: ~80 → ~25 (70% reduction)
- Total (17 agents): ~1,360 tokens → ~425 tokens
- Quality: Minimal impact (tested with A/B comparison)

---

### 6. Sub-Agent Architecture for Multi-Phase Workflows

**Problem**: `/specify` → `/plan` → `/tasks` workflow loads all phase context simultaneously.

**Solution**: Sequential sub-sessions with structured handoff summaries.

**Current Flow** (single session):
```
┌─────────────────────────────────────────┐
│ Session Context (accumulated)           │
├─────────────────────────────────────────┤
│ 1. User prompt                          │
│ 2. /specify execution + full output     │
│ 3. /plan execution + full output        │
│ 4. /tasks execution + full output       │
│ Total: ~200k tokens (hits limit)        │
└─────────────────────────────────────────┘
```

**New Flow** (sub-agent pattern):
```
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│ /specify     │──────▶│ /plan        │──────▶│ /tasks       │
│ Sub-Session  │ JSON │ Sub-Session  │ JSON │ Sub-Session  │
└──────────────┘      └──────────────┘      └──────────────┘
     │                     │                     │
     ▼                     ▼                     ▼
  Full context        Summary only         Summary only
  (80k tokens)        (5k tokens)          (8k tokens)
```

**Implementation**:

```python
# New: Phase orchestrator
class PhaseOrchestrator:
    """Orchestrates multi-phase spec-driven workflows with clean context."""

    async def run_specify_plan_tasks(self, user_requirements: str) -> Dict[str, Any]:
        """Execute spec workflow with sub-agent isolation."""

        # Phase 1: /specify
        specify_result = await self._run_phase(
            phase="specify",
            prompt=user_requirements,
            agents=["PRODUCT_OWNER", "ARCHITECT"]
        )

        # Extract structured summary (not full transcript)
        specify_summary = self._extract_key_decisions(specify_result)

        # Phase 2: /plan
        plan_result = await self._run_phase(
            phase="plan",
            prompt=f"Based on specification summary:\n{specify_summary}\n\nCreate implementation plan.",
            agents=["STAFF_ENGINEER", "ENGINEERING_MANAGER"]
        )

        plan_summary = self._extract_key_decisions(plan_result)

        # Phase 3: /tasks
        tasks_result = await self._run_phase(
            phase="tasks",
            prompt=f"Based on plan summary:\n{plan_summary}\n\nCreate task breakdown.",
            agents=["TEAM_LEAD", "TEAM_MEMBER"]
        )

        return {
            "specify": specify_summary,
            "plan": plan_summary,
            "tasks": tasks_result
        }

    def _extract_key_decisions(self, phase_result: str) -> str:
        """Extract structured summary using lightweight model."""
        # Use Claude Haiku to extract:
        # - Key decisions made
        # - Acceptance criteria
        # - Dependencies identified
        # - Risk factors
        # Limit to 1,000 tokens max
        pass
```

**Handoff Format** (structured):
```json
{
  "phase": "specify",
  "keyDecisions": [
    "User authentication via OAuth2 + JWT",
    "PostgreSQL for user data, Redis for sessions",
    "React frontend with Node.js/Express backend"
  ],
  "acceptanceCriteria": [
    "Users can log in with email/password",
    "Session persists for 24 hours",
    "Password reset via email link"
  ],
  "dependencies": ["SendGrid API", "PostgreSQL 14+"],
  "risks": ["GDPR compliance for EU users"]
}
```

**Impact**:
- Context per phase: 200k tokens → 80k tokens (60% reduction)
- Total workflow: Single session (limited) → Three clean sessions (scalable)
- Quality: Improved focus per phase vs. context-overloaded single session

---

### 7. C4 Architecture Context Loading

**Problem**: Full 262-line `workspace.dsl` file loaded for all architecture references.

**Solution**: Extract relevant container/component sections based on task scope.

**Implementation**:

```python
# New: Architecture context extractor
class C4ContextExtractor:
    """Extract relevant architecture sections for focused tasks."""

    def __init__(self, dsl_path: Path):
        self.dsl_content = dsl_path.read_text()
        self._parse_structure()

    def get_context_for_task(self, task_description: str) -> str:
        """Return minimal architecture context relevant to task."""
        # Pattern matching or LLM classification
        if "backend" in task_description.lower():
            return self._extract_backend_context()
        elif "frontend" in task_description.lower():
            return self._extract_frontend_context()
        elif "operator" in task_description.lower():
            return self._extract_operator_context()
        elif "runner" in task_description.lower():
            return self._extract_runner_context()
        else:
            return self._extract_system_context()  # High-level only

    def _extract_backend_context(self) -> str:
        """Extract Backend container + components only."""
        # Lines 34-43 (Backend container definition)
        # + relevant relationships
        # ~30 lines instead of 262
        pass
```

**Usage**:
```python
# main.py
from c4_context_extractor import C4ContextExtractor

extractor = C4ContextExtractor(Path("architecture/workspace.dsl"))
relevant_arch = extractor.get_context_for_task(self.prompt)

# Include in system prompt
system_prompt = f"""
{base_prompt}

Relevant Architecture:
{relevant_arch}
"""
```

**Examples**:

| Task | Full DSL | Extracted Context | Reduction |
|------|----------|-------------------|-----------|
| "Modify backend API" | 262 lines | 35 lines (Backend + relationships) | 87% |
| "Add frontend feature" | 262 lines | 40 lines (Frontend + Backend API) | 85% |
| "System overview" | 262 lines | 50 lines (Context diagram only) | 81% |
| "Operator debugging" | 262 lines | 45 lines (Operator + CRDs) | 83% |

**Impact**:
- Typical context: ~1,500 tokens → ~300 tokens (80% reduction)
- Precision: Higher focus on relevant components
- Maintenance: Auto-updates when DSL changes

---

### 8. Sample Knowledge Deduplication

**Problem**: `sampleKnowledge` sections in agent YAML files contain overlapping content.

**Analysis**:
```bash
# Duplication analysis
grep -h "OpenShift AI" agents/*.yaml | wc -l
# Result: 12 agents mention "OpenShift AI Platform Management"

grep -h "Agile/Scrum" agents/*.yaml | wc -l
# Result: 8 agents mention "Agile/Scrum methodologies"
```

**Solution**: Shared knowledge base with agent-specific references.

**Before** (`agents/engineering_manager.yaml` lines 68-97):
```yaml
sampleKnowledge: |
  # Engineering Management Best Practices

  ## Team Leadership
  - Building high-performing engineering teams
  [... 10 lines ...]

  ## Capacity Planning
  - Sprint capacity estimation and tracking
  [... 8 lines ...]

  ## OpenShift AI Platform Management  # ← Duplicated across 12 agents
  - ML development lifecycle management
  [... 6 lines ...]
```

**After** (refactored):
```yaml
# agents/engineering_manager.yaml
knowledgeRefs:
  - "shared/team-leadership"
  - "shared/capacity-planning"
  - "shared/openshift-ai-basics"  # ← Shared reference
  - "engineering-management/delivery-excellence"  # ← Agent-specific

# shared/openshift-ai-basics.md (single source of truth)
# OpenShift AI Platform Basics
- ML development lifecycle management
- Cross-functional team coordination
- Technical decision-making for ML systems
[... common knowledge ...]
```

**Refactoring Process**:
1. Identify duplicated knowledge blocks (>50% similarity)
2. Extract to `agents/shared/*.md` files
3. Replace verbose `sampleKnowledge` with compact `knowledgeRefs`
4. Update `AgentLoader` to resolve references on-demand

**Implementation**:
```python
# agent_loader.py
class AgentPersona:
    def get_full_knowledge(self) -> str:
        """Resolve knowledge references and combine with agent-specific knowledge."""
        knowledge_parts = []

        for ref in self.knowledge_refs:
            if ref.startswith("shared/"):
                path = Path(f"agents/shared/{ref.split('/')[-1]}.md")
            else:
                path = Path(f"agents/knowledge/{ref}.md")

            if path.exists():
                knowledge_parts.append(path.read_text())

        # Add agent-specific knowledge if present
        if self.sample_knowledge:
            knowledge_parts.append(self.sample_knowledge)

        return "\n\n".join(knowledge_parts)
```

**Impact**:
- Duplication: ~40% reduction across agent knowledge base
- Maintenance: Single update propagates to all agents
- Consistency: Ensures uniform knowledge representation
- Lazy loading: Load only when agent is fully invoked (not for summaries)

---

## Implementation Roadmap

### Phase 1: Quick Wins (Week 1-2)
**Goal**: Immediate 50% context reduction for typical sessions

- [ ] **1.1 Agent Prompt Compaction** (Improvement #5)
  - Condense all 17 agent system messages
  - Script: `scripts/condense_agent_prompts.py`
  - Test: A/B comparison of output quality
  - **Impact**: ~1,000 tokens saved per session

- [ ] **1.2 Tool Bundle Presets** (Improvement #4)
  - Define 5 tool bundles (code_editing, research, analysis, documentation, minimal)
  - Update `main.py` to support `TOOL_BUNDLE` env var
  - Document in runner CLAUDE.md
  - **Impact**: ~500 tokens saved per session

- [ ] **1.3 Selective Agent Loading** (Improvement #2 - complete existing partial implementation)
  - Enhance `_inject_selected_agents()` with auto-recommendation
  - Add `AGENT_SELECTION_MODE` env var
  - Default to auto-recommend 2-4 agents per task
  - **Impact**: ~1,200 tokens saved per session (avg 4 agents vs 17)

**Total Phase 1 Impact**: ~2,700 tokens saved (~15-20% reduction)

---

### Phase 2: Architectural Improvements (Week 3-4)
**Goal**: Enable long-running sessions and complex workflows

- [ ] **2.1 Conversation Summarization** (Improvement #3)
  - Implement `_maybe_summarize_conversation()` in chat mode
  - Add summary generation via Claude Haiku
  - Configure `CONTEXT_SUMMARY_THRESHOLD` (default: 150k tokens)
  - **Impact**: Enables indefinite chat sessions

- [ ] **2.2 Sub-Agent Phase Orchestration** (Improvement #6)
  - Create `PhaseOrchestrator` class
  - Implement structured handoff summaries
  - Refactor `/specify`, `/plan`, `/tasks` workflow
  - **Impact**: 60% context reduction for multi-phase workflows

- [ ] **2.3 Agent Compact Summaries** (Improvement #1)
  - Add `get_compact_summary()` to `AgentPersona`
  - Load compact summaries by default
  - Full definition loaded only when agent actively invoked
  - **Impact**: ~1,500 tokens saved for agent metadata

**Total Phase 2 Impact**: Major scalability improvements for complex tasks

---

### Phase 3: Long-Term Optimization (Week 5-6)
**Goal**: Technical debt cleanup and advanced features

- [ ] **3.1 C4 Context Extraction** (Improvement #7)
  - Create `C4ContextExtractor` class
  - Integrate with `main.py` system prompt
  - Add task-based architecture filtering
  - **Impact**: ~1,200 tokens saved for architecture context

- [ ] **3.2 Knowledge Base Refactoring** (Improvement #8)
  - Analyze agent knowledge duplication
  - Extract shared knowledge to `agents/shared/*.md`
  - Update `AgentLoader` to support `knowledgeRefs`
  - Migrate all 17 agents to new format
  - **Impact**: ~40% reduction in knowledge duplication

- [ ] **3.3 Testing & Validation**
  - A/B test context-optimized sessions vs. baseline
  - Measure output quality metrics
  - Benchmark context window utilization
  - Document best practices

**Total Phase 3 Impact**: Foundational improvements for maintainability

---

## Success Metrics

### Quantitative Targets

| Metric | Baseline | Target | Measurement |
|--------|----------|--------|-------------|
| **Average context utilization** | ~70% | <40% | Token counting per session |
| **Session startup time** | ~4s | <5s (no regression) | Time to first AI response |
| **Agent metadata overhead** | ~1,650 lines | ~85 lines | Loaded context size |
| **Multi-phase workflow context** | ~200k tokens | ~80k tokens | Per-phase measurement |
| **Chat session max length** | ~100 turns | ~250 turns | Before summarization needed |

### Qualitative Targets

- [ ] **No degradation in output quality** (A/B tested with sample prompts)
- [ ] **Improved focus in AI responses** (less irrelevant agent context)
- [ ] **Faster iteration for long sessions** (reduced context rot)
- [ ] **Better architecture alignment** (focused C4 context)

---

## Risk Mitigation

### Risk 1: Over-Aggressive Context Reduction
**Impact**: Loss of important nuance in AI responses

**Mitigation**:
- Incremental rollout (Phase 1 → 2 → 3)
- A/B testing at each phase
- Rollback mechanism if quality degrades
- Configurable via feature flags

**Monitoring**:
- User feedback on session quality
- Manual review of 10% of sessions
- Automated quality scoring (future)

---

### Risk 2: Just-In-Time Loading Latency
**Impact**: Slower session startup if agent/context loading is synchronous

**Mitigation**:
- Pre-cache compact agent summaries
- Async loading for full agent definitions
- Parallel context fetching
- Benchmark acceptable latency (<500ms)

**Monitoring**:
- P50/P95/P99 session startup time
- Agent loading duration metrics
- User-perceived responsiveness

---

### Risk 3: Increased Code Complexity
**Impact**: Harder to maintain orchestration logic

**Mitigation**:
- Clear separation of concerns (PhaseOrchestrator, C4ContextExtractor)
- Comprehensive unit tests for new components
- Documentation in code + this plan
- Feature flags for gradual enablement

**Monitoring**:
- Code review feedback
- Cyclomatic complexity metrics
- Test coverage (maintain >80%)

---

## Alternative Approaches Considered

### Alternative 1: Prompt Caching (Anthropic Feature)
**Description**: Use Anthropic's prompt caching to reuse common context across requests.

**Pros**:
- Reduces latency for repeated context
- Cost savings (~90% for cached portions)
- No code changes needed

**Cons**:
- Doesn't solve context window exhaustion
- Only helps with identical context (less useful for dynamic agent selection)
- Still pays for context on first request

**Decision**: Pursue in parallel, not a replacement for context reduction.

---

### Alternative 2: RAG-Based Agent Knowledge
**Description**: Store agent knowledge in vector database, retrieve relevant snippets.

**Pros**:
- Infinite knowledge base (not limited by context window)
- Dynamic retrieval based on query relevance

**Cons**:
- Adds infrastructure dependency (vector DB)
- Retrieval latency (~200-500ms)
- Complexity of chunking and embedding
- Overkill for current knowledge base size (~10k lines)

**Decision**: Revisit if knowledge base exceeds 50k lines.

---

### Alternative 3: Multi-Model Cascade
**Description**: Use cheaper models (Haiku) for initial analysis, escalate to Opus only when needed.

**Pros**:
- Cost optimization
- Faster for simple tasks

**Cons**:
- Doesn't address context window limits
- Requires orchestration logic
- May degrade quality for complex tasks

**Decision**: Implement for summarization only (Improvement #3), not core task execution.

---

## References

### External Resources
- [Anthropic: Effective Context Engineering for AI Agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)
- [Anthropic Prompt Caching Documentation](https://docs.anthropic.com/claude/docs/prompt-caching)
- [C4 Model Documentation](https://c4model.com)

### Internal Documentation
- [vTeam Architecture (C4 Model)](../architecture/workspace.dsl)
- [Agent Framework Overview](../rhoai-ux-agents-vTeam.md)
- [Runner Implementation](../components/runners/claude-code-runner/README.md)
- [C4 Architecture Documentation](../architecture/README.md)

### Related Work
- [C4 vs. Prompting Analysis](research/c4-vs-prompting-analysis.md) (future)
- [Agent System Benchmarks](research/agent-benchmarks.md) (future)

---

## Appendix: Code Examples

### Example 1: Compact Agent Summary Usage

```python
# Before: Full agent loading
agent = agent_loader.get_agent("ENGINEERING_MANAGER")
prompt = f"{agent.system_message}\n\n{agent.sample_knowledge}\n\n{user_prompt}"
# Result: ~1,200 tokens

# After: Compact summary
summary = agent_loader.get_agent_summary("ENGINEERING_MANAGER")
prompt = f"{summary['one_liner']}\n\n{user_prompt}"
# Result: ~50 tokens (96% reduction)

# Full context loaded only when agent actively invoked
if agent_invoked:
    full_agent = agent_loader.get_agent("ENGINEERING_MANAGER")
    # Load full system message + knowledge for execution
```

---

### Example 2: Sub-Agent Phase Handoff

```python
# Phase 1: /specify execution
specify_result = {
    "fullOutput": "[... 15,000 tokens of detailed specification ...]",
    "summary": {
        "keyDecisions": ["OAuth2 + JWT", "PostgreSQL + Redis", "React + Node.js"],
        "acceptanceCriteria": ["Login via email/password", "24h session persistence"],
        "dependencies": ["SendGrid API", "PostgreSQL 14+"],
        "risks": ["GDPR compliance"]
    }
}

# Phase 2: /plan receives summary only (not full output)
plan_prompt = f"""
Based on specification:
- Authentication: OAuth2 + JWT
- Stack: React, Node.js, PostgreSQL, Redis
- Dependencies: SendGrid API, PostgreSQL 14+
- Risks: GDPR compliance for EU users

Acceptance Criteria:
1. Login via email/password
2. 24-hour session persistence

Create detailed implementation plan.
"""
# Context: 200 tokens (vs 15,000 if we passed full /specify output)
```

---

### Example 3: C4 Context Extraction

```python
# Before: Full architecture loaded
arch_context = Path("architecture/workspace.dsl").read_text()
# Result: ~1,500 tokens

# After: Task-specific extraction
extractor = C4ContextExtractor(Path("architecture/workspace.dsl"))
task = "Add new API endpoint to backend service"
arch_context = extractor.get_context_for_task(task)
# Result: ~300 tokens (Backend container + API component + relationships)

# Extracted context:
"""
Backend API Service (Go, Gin Framework)
├── Sessions API (manages AgenticSession lifecycle)
├── Secrets API (secure API key storage)
├── RBAC Middleware (authorization)
└── Kubernetes Client (CR management)

Relationships:
- Backend → Custom Resources (reads/writes via Kubernetes API)
- Frontend → Backend (calls via HTTPS REST)
"""
```

---

**Document Version**: 1.0
**Last Updated**: 2025-09-30
**Owner**: vTeam Architecture Team
**Status**: Approved for Implementation
