// Bot management types for the Ambient Agentic Runner frontend
// Extends the project.ts types with detailed bot management functionality

export interface BotConfig {
  name: string;
  description?: string;
  enabled: boolean;
  token?: string; // Only shown to admins
  createdAt?: string;
  lastUsed?: string;
}

export interface CreateBotRequest {
  name: string;
  description?: string;
  enabled?: boolean;
}

export interface UpdateBotRequest {
  description?: string;
  enabled?: boolean;
}

export interface BotListResponse {
  items: BotConfig[];
}

export interface BotResponse {
  bot: BotConfig;
}

export interface User {
  id: string;
  username: string;
  roles: string[];
  permissions: string[];
}

// User role and permission types for admin checking
export enum UserRole {
  ADMIN = "admin",
  USER = "user",
  VIEWER = "viewer"
}

export enum Permission {
  CREATE_BOT = "create_bot",
  DELETE_BOT = "delete_bot",
  VIEW_BOT_TOKEN = "view_bot_token",
  MANAGE_BOTS = "manage_bots"
}

// Form validation types
export interface BotFormData {
  name: string;
  description: string;
  enabled: boolean;
}

export interface BotFormErrors {
  name?: string;
  description?: string;
  enabled?: string;
}

// Bot status types
export enum BotStatus {
  ACTIVE = "active",
  INACTIVE = "inactive",
  ERROR = "error"
}

// API error response
export interface ApiError {
  message: string;
  code?: string;
  details?: string;
}