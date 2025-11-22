"use client"

import { useEffect } from "react"
import { useTheme } from "next-themes"

/**
 * SyntaxThemeProvider - Manages Syntax Highlighting Theme
 *
 * Manages highlight.js theme by updating the data-hljs-theme attribute
 * on the document element when the theme changes.
 *
 * This works in tandem with next-themes:
 * - next-themes handles the 'dark' class and localStorage persistence
 * - This provider syncs the syntax highlighting theme attribute
 *
 * The useEffect ensures this only runs after React hydration on the client,
 * preventing hydration mismatches with SSR.
 *
 * The actual syntax highlighting stylesheets are bundled locally in
 * globals.css (syntax-highlighting.css) rather than loaded from CDN.
 */
export function SyntaxThemeProvider() {
  const { resolvedTheme } = useTheme()

  useEffect(() => {
    // Update data attribute when theme changes (client-side only)
    // This keeps syntax highlighting in sync with the active theme
    if (resolvedTheme === 'dark') {
      document.documentElement.setAttribute('data-hljs-theme', 'dark')
    } else {
      document.documentElement.setAttribute('data-hljs-theme', 'light')
    }
  }, [resolvedTheme])

  return null
}
