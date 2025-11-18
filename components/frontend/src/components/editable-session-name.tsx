/**
 * EditableSessionName component
 * Allows inline editing of session display names with auto-edit mode for default names
 */

import { useState, useEffect, useRef } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { Loader2 } from 'lucide-react';

type EditableSessionNameProps = {
  currentName: string;
  onSave: (newName: string) => Promise<void>;
  isSaving?: boolean;
  className?: string;
};

export function EditableSessionName({
  currentName,
  onSave,
  isSaving = false,
  className,
}: EditableSessionNameProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [inputValue, setInputValue] = useState(currentName);
  const [hasChanges, setHasChanges] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // Focus input when entering edit mode
  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [isEditing]);

  // Update input value when currentName changes
  useEffect(() => {
    setInputValue(currentName);
    setHasChanges(false);
  }, [currentName]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    setInputValue(newValue);
    setHasChanges(newValue.trim() !== currentName && newValue.trim() !== '');
  };

  const handleSave = async () => {
    const trimmedValue = inputValue.trim();
    if (!trimmedValue || trimmedValue === currentName) {
      setIsEditing(false);
      setInputValue(currentName);
      setHasChanges(false);
      return;
    }

    try {
      await onSave(trimmedValue);
      setIsEditing(false);
      setHasChanges(false);
    } catch (error) {
      // Error handling is done by the parent component via toast
      console.error('Failed to save session name:', error);
    }
  };

  const handleCancel = () => {
    setInputValue(currentName);
    setIsEditing(false);
    setHasChanges(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleSave();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      handleCancel();
    }
  };

  // If in edit mode, show input
  if (isEditing) {
    return (
      <div className="flex items-center gap-2">
        <Input
          ref={inputRef}
          type="text"
          value={inputValue}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onBlur={() => {
            // Don't auto-close on blur if there are changes - user might want to click the Update button
            if (!hasChanges) {
              handleCancel();
            }
          }}
          placeholder="New session..."
          disabled={isSaving}
          className={cn('h-auto py-1 px-2 w-auto min-w-[300px] flex-1 !text-[1em] !font-[inherit] !leading-[inherit]', className)}
        />
        <Button
          onClick={handleSave}
          disabled={isSaving || !hasChanges}
          size="sm"
          className="whitespace-nowrap"
        >
          {isSaving ? (
            <>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              Saving...
            </>
          ) : (
            'Update'
          )}
        </Button>
      </div>
    );
  }

  // If not editing, show clickable title
  return (
    <h1
      className={cn(
        'cursor-pointer hover:text-primary transition-colors',
        className
      )}
      onClick={() => setIsEditing(true)}
      title="Click to edit session name"
    >
      {currentName}
    </h1>
  );
}

