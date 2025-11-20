"use client"

import { useEffect } from "react"
import { useTheme } from "next-themes"

/**
 * SyntaxThemeProvider - Runtime Theme Updates (Post-Hydration)
 *
 * Manages highlight.js theme by updating the data-hljs-theme attribute
 * on the document element when the user changes themes.
 *
 * Why this exists alongside the blocking script in layout.tsx:
 *
 * 1. Blocking Script (layout.tsx): Handles INITIAL page load
 *    - Runs during HTML parsing (before React hydration)
 *    - Prevents FOUC on first paint
 *    - Cannot respond to runtime theme changes
 *
 * 2. SyntaxThemeProvider (this file): Handles RUNTIME theme changes
 *    - Runs after React hydration
 *    - Responds to theme toggle clicks
 *    - Updates syntax highlighting when user changes theme
 *
 * Both are necessary:
 * - Without blocking script: Flash of wrong theme on page load
 * - Without this provider: Syntax highlighting doesn't update when theme changes
 *
 * The actual syntax highlighting stylesheets are bundled locally in
 * globals.css (syntax-highlighting.css) rather than loaded from CDN.
 */
export function SyntaxThemeProvider() {
  const { resolvedTheme } = useTheme()

  useEffect(() => {
    // Update data attribute when theme changes (post-hydration)
    // This syncs with the blocking script's initial setting
    if (resolvedTheme === 'dark') {
      document.documentElement.setAttribute('data-hljs-theme', 'dark')
    } else {
      document.documentElement.setAttribute('data-hljs-theme', 'light')
    }
  }, [resolvedTheme])

  return null
}
