import type { AgenticSessionPhase } from "@/types/agentic-session";

/**
 * Get the color classes for a session phase badge
 */
export const getPhaseColor = (phase: AgenticSessionPhase): string => {
  switch (phase) {
    case "Pending":
      return "bg-yellow-100 text-yellow-800";
    case "Creating":
    case "Running":
      return "bg-blue-100 text-blue-800";
    case "Completed":
      return "bg-green-100 text-green-800";
    case "Failed":
    case "Error":
      return "bg-red-100 text-red-800";
    case "Stopped":
      return "bg-gray-100 text-gray-800";
    default:
      return "bg-gray-100 text-gray-800";
  }
};

