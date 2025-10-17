/**
 * Form Field Wrapper Component
 * Simplifies form field creation with consistent styling
 */

import * as React from 'react';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';
import { AlertCircle, HelpCircle } from 'lucide-react';

export type FormFieldWrapperProps = {
  label: string;
  htmlFor?: string;
  required?: boolean;
  error?: string;
  help?: string;
  className?: string;
  children: React.ReactNode;
};

export function FormFieldWrapper({
  label,
  htmlFor,
  required = false,
  error,
  help,
  className,
  children,
}: FormFieldWrapperProps) {
  return (
    <div className={cn('space-y-2', className)}>
      <Label htmlFor={htmlFor} className="flex items-center gap-1">
        {label}
        {required && <span className="text-destructive">*</span>}
      </Label>
      {children}
      {help && !error && (
        <p className="text-sm text-muted-foreground flex items-center gap-1">
          <HelpCircle className="h-3 w-3" />
          {help}
        </p>
      )}
      {error && (
        <p className="text-sm text-destructive flex items-center gap-1">
          <AlertCircle className="h-3 w-3" />
          {error}
        </p>
      )}
    </div>
  );
}

/**
 * Grid layout for multiple form fields
 */
export type FormFieldsGridProps = {
  children: React.ReactNode;
  columns?: 1 | 2 | 3;
  className?: string;
};

export function FormFieldsGrid({ children, columns = 1, className }: FormFieldsGridProps) {
  const gridClass = {
    1: 'grid-cols-1',
    2: 'grid-cols-1 md:grid-cols-2',
    3: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
  }[columns];

  return <div className={cn('grid gap-4', gridClass, className)}>{children}</div>;
}

/**
 * Form section with title and description
 */
export type FormSectionProps = {
  title: string;
  description?: string;
  children: React.ReactNode;
  className?: string;
};

export function FormSection({ title, description, children, className }: FormSectionProps) {
  return (
    <div className={cn('space-y-4', className)}>
      <div className="space-y-1">
        <h3 className="text-lg font-medium">{title}</h3>
        {description && <p className="text-sm text-muted-foreground">{description}</p>}
      </div>
      {children}
    </div>
  );
}

/**
 * Form actions footer with consistent spacing
 */
export type FormActionsProps = {
  children: React.ReactNode;
  align?: 'left' | 'right' | 'center';
  className?: string;
};

export function FormActions({ children, align = 'right', className }: FormActionsProps) {
  const alignClass = {
    left: 'justify-start',
    right: 'justify-end',
    center: 'justify-center',
  }[align];

  return <div className={cn('flex gap-2 pt-4', alignClass, className)}>{children}</div>;
}
