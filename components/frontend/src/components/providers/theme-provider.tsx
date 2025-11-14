'use client';

import { createContext, useContext, useEffect, useState } from 'react';

type Theme = 'light' | 'dark';

type ThemeContextValue = {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
};

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>('light');
  const [mounted, setMounted] = useState(false);

  // Read theme on mount (client-side only)
  useEffect(() => {
    setMounted(true);
    const stored = localStorage.getItem('theme');

    // Security: Validate that stored value is exactly 'light' or 'dark'
    // to prevent XSS or DOM manipulation via localStorage poisoning
    if (stored === 'light' || stored === 'dark') {
      setTheme(stored);
    } else {
      // If invalid or missing, fall back to system preference
      if (stored !== null) {
        // Clear invalid value from localStorage
        try {
          localStorage.removeItem('theme');
        } catch (e) {
          console.warn('Failed to remove invalid theme value:', e);
        }
      }
      const systemPrefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      setTheme(systemPrefersDark ? 'dark' : 'light');
    }
  }, []);

  // Sync theme changes to DOM and localStorage
  useEffect(() => {
    if (!mounted) return;

    const root = document.documentElement;
    if (theme === 'dark') {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }

    try {
      localStorage.setItem('theme', theme);
    } catch (e) {
      console.warn('Failed to save theme preference:', e);
    }
  }, [theme, mounted]);

  // Wrapper for setTheme with validation (defense-in-depth)
  const setThemeValidated = (newTheme: Theme) => {
    // Security: Validate input to ensure only valid theme values are accepted
    if (newTheme !== 'light' && newTheme !== 'dark') {
      console.warn('Invalid theme value rejected:', newTheme);
      return;
    }
    setTheme(newTheme);
  };

  const toggleTheme = () => {
    setTheme((prev) => (prev === 'light' ? 'dark' : 'light'));
  };

  return (
    <ThemeContext.Provider value={{ theme, setTheme: setThemeValidated, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider');
  }
  return context;
}
