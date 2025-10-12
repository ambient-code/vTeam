/**
 * Component-specific types for forms
 */

export type FormFieldError = {
  message: string;
};

export type FormErrors<T> = {
  [K in keyof T]?: string[];
};

export type ActionState = {
  error?: string;
  errors?: Record<string, string[]>;
};

export type FormState<T = unknown> = {
  success: boolean;
  message?: string;
  errors?: FormErrors<T>;
  data?: T;
};
