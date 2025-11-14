import { cn } from '../utils';

describe('cn utility', () => {
  it('merges multiple class names', () => {
    const result = cn('text-red-500', 'bg-blue-500');
    expect(result).toContain('text-red-500');
    expect(result).toContain('bg-blue-500');
  });

  it('handles conditional classes', () => {
    const result = cn('base-class', true && 'conditional-class', false && 'hidden-class');
    expect(result).toContain('base-class');
    expect(result).toContain('conditional-class');
    expect(result).not.toContain('hidden-class');
  });

  it('handles empty inputs', () => {
    const result = cn();
    expect(result).toBe('');
  });

  it('handles undefined and null values', () => {
    const result = cn('base', undefined, null, 'other');
    expect(result).toContain('base');
    expect(result).toContain('other');
  });

  it('merges tailwind classes correctly', () => {
    // Later class should override earlier class in Tailwind
    const result = cn('p-4', 'p-8');
    expect(result).toBe('p-8');
  });
});

