import type { AgenticSessionPhase } from "@/types/agentic-session";
import { getSessionPhaseColor } from "@/lib/status-colors";

/**
 * Get the color classes for a session phase badge
 * @deprecated Use getSessionPhaseColor from @/lib/status-colors instead
 */
export const getPhaseColor = (phase: AgenticSessionPhase): string => {
  return getSessionPhaseColor(phase);
};

