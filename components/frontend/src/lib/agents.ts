import { AgentPersona } from "@/types/agentic-session";

// Agent persona definitions based on the YAML files in claude-runner
export const AVAILABLE_AGENTS: AgentPersona[] = [
  {
    persona: "emma-engineering_manager",
    name: "Emma Engineering Manager",
    role: "Engineering Manager",
    description: "Engineering Manager Agent focused on team wellbeing, strategic planning, and delivery coordination. Use PROACTIVELY for team management, capacity planning, and balancing technical excellence with business needs."
  },
  {
    persona: "stella-staff_engineer",
    name: "Stella Staff Engineer",
    role: "Staff Engineer",
    description: "Staff Engineer Agent focused on technical leadership, implementation excellence, and mentoring. Use PROACTIVELY for complex technical problems, code review, and bridging architecture to implementation."
  },
  {
    persona: "parker-product_manager",
    name: "Parker Product Manager",
    role: "Product Manager",
    description: "Product Manager Agent focused on market strategy, customer feedback, and business value delivery. Use PROACTIVELY for product roadmap decisions, competitive analysis, and translating business requirements to technical features."
  },
  {
    persona: "lee-team_lead",
    name: "Lee Team Lead",
    role: "Team Lead",
    description: "Team Lead Agent focused on team coordination, technical decision facilitation, and delivery execution. Use PROACTIVELY for sprint leadership, technical planning, and cross-team communication."
  },
  {
    persona: "sam-scrum_master",
    name: "Sam Scrum Master",
    role: "Scrum Master",
    description: "Scrum Master Agent focused on agile facilitation, impediment removal, and team process optimization. Use PROACTIVELY for sprint planning, retrospectives, and process improvement."
  },
  {
    persona: "aria-ux_architect",
    name: "Aria UX Architect",
    role: "UX Architect",
    description: "UX Architect Agent focused on user experience strategy, journey mapping, and design system architecture. Use PROACTIVELY for holistic UX planning, ecosystem design, and user research strategy."
  },
  {
    persona: "uma-ux_team_lead",
    name: "Uma UX Team Lead",
    role: "UX Team Lead",
    description: "UX Team Lead Agent focused on design quality, team coordination, and design system governance. Use PROACTIVELY for design process management, critique facilitation, and design team leadership."
  },
  {
    persona: "felix-ux_feature_lead",
    name: "Felix UX Feature Lead",
    role: "UX Feature Lead",
    description: "UX Feature Lead Agent focused on component design, pattern reusability, and accessibility implementation. Use PROACTIVELY for detailed feature design, component specification, and accessibility compliance."
  },
  {
    persona: "ryan-ux_researcher",
    name: "Ryan UX Researcher",
    role: "UX Researcher",
    description: "UX Researcher Agent focused on user insights, data analysis, and evidence-based design decisions. Use PROACTIVELY for user research planning, usability testing, and translating insights to design recommendations."
  },
  {
    persona: "casey-content_strategist",
    name: "Casey Content Strategist",
    role: "Content Strategist",
    description: "Content Strategist Agent focused on information architecture, content standards, and strategic content planning. Use PROACTIVELY for content taxonomy, style guidelines, and content effectiveness measurement."
  },
  {
    persona: "terry-technical_writer",
    name: "Terry Technical Writer",
    role: "Technical Writer",
    description: "Technical Writer Agent focused on user-centered documentation, procedure testing, and clear technical communication. Use PROACTIVELY for hands-on documentation creation and technical accuracy validation."
  },
  {
    persona: "tessa-writing_manager",
    name: "Tessa Writing Manager",
    role: "Writing Manager",
    description: "Technical Writing Manager Agent focused on documentation strategy, team coordination, and content quality. Use PROACTIVELY for documentation planning, writer management, and content standards."
  },
  {
    persona: "jack-delivery_owner",
    name: "Jack Delivery Owner",
    role: "Delivery Owner",
    description: "Delivery Owner Agent focused on cross-team coordination, dependency tracking, and milestone management. Use PROACTIVELY for release planning, risk mitigation, and delivery status reporting."
  },
  {
    persona: "phoenix-pxe_specialist",
    name: "Phoenix PXE Specialist",
    role: "PXE Specialist",
    description: "PXE (Product Experience Engineering) Agent focused on customer impact assessment, lifecycle management, and field experience insights. Use PROACTIVELY for upgrade planning, risk assessment, and customer telemetry analysis."
  },
  {
    persona: "phoenix-pxe_specialist",
    name: "Neil Test Engineer",
    role: "Test Engineer/Architect",
    description: "Test engineer Agent focused on, the testing requirements i.e. whether the changes are testable, implementation matches product/customer requirements, cross component impact, automation testing, performance & security impact"
  }
];

// Default agent selections for different workflow types
export const DEFAULT_AGENT_SELECTIONS = {
  BALANCED: ["emma-engineering_manager", "stella-staff_engineer", "parker-product_manager", "lee-team_lead"],
  TECHNICAL: ["stella-staff_engineer", "emma-engineering_manager", "aria-ux_architect", "terry-technical_writer"],
  PRODUCT: ["parker-product_manager", "ryan-ux_researcher", "casey-content_strategist", "jack-delivery_owner"],
  DESIGN: ["aria-ux_architect", "felix-ux_feature_lead", "ryan-ux_researcher", "casey-content_strategist"],
  PROCESS: ["sam-scrum_master", "emma-engineering_manager", "jack-delivery_owner", "lee-team_lead"]
};

// Utility functions
export function getAgentByPersona(persona: string): AgentPersona | undefined {
  return AVAILABLE_AGENTS.find(agent => agent.persona === persona);
}

export function getDefaultAgents(type: keyof typeof DEFAULT_AGENT_SELECTIONS = "BALANCED"): AgentPersona[] {
  return DEFAULT_AGENT_SELECTIONS[type].map(persona => getAgentByPersona(persona)!).filter(Boolean);
}

export function getAgentsByExpertise(expertise: string): AgentPersona[] {
  return AVAILABLE_AGENTS.filter(agent =>
      agent.description.toLowerCase().includes(expertise.toLowerCase()) ||
      agent.role.toLowerCase().includes(expertise.toLowerCase())
  );
}

export function groupAgentsByRole(): { [role: string]: AgentPersona[] } {
  const groups: { [role: string]: AgentPersona[] } = {};

  AVAILABLE_AGENTS.forEach(agent => {
    const category = getCategoryForRole(agent.role);
    if (!groups[category]) {
      groups[category] = [];
    }
    groups[category].push(agent);
  });

  return groups;
}

function getCategoryForRole(role: string): string {
  if (role.includes("Engineering") || role.includes("Technical")) return "Engineering";
  if (role.includes("UX") || role.includes("Design")) return "Design";
  if (role.includes("Product") || role.includes("Strategy")) return "Product";
  if (role.includes("Content") || role.includes("Documentation")) return "Content";
  return "Process & Leadership";
}