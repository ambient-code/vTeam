"use client"

import { useEffect } from "react"
import { useTheme } from "next-themes"

/**
 * Manages highlight.js theme by toggling CSS classes on the document element.
 * The actual theme stylesheets are imported in globals.css to ensure they're
 * bundled locally from node_modules/highlight.js rather than loaded from CDN.
 */
export function SyntaxThemeProvider() {
  const { resolvedTheme } = useTheme()

  useEffect(() => {
    // Add a data attribute to the root element to indicate which hljs theme to use
    // This is used in globals.css to conditionally apply the appropriate theme
    if (resolvedTheme === 'dark') {
      document.documentElement.setAttribute('data-hljs-theme', 'dark')
    } else {
      document.documentElement.setAttribute('data-hljs-theme', 'light')
    }
  }, [resolvedTheme])

  return null
}
