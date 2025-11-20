"use client"

import { useEffect } from "react"
import { useTheme } from "next-themes"

export function SyntaxThemeProvider() {
  const { resolvedTheme } = useTheme()

  useEffect(() => {
    // Remove any existing highlight.js theme stylesheets
    const existingLinks = document.querySelectorAll('link[data-hljs-theme]')
    existingLinks.forEach(link => link.remove())

    // Add the appropriate theme based on current mode
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.setAttribute('data-hljs-theme', 'true')

    if (resolvedTheme === 'dark') {
      link.href = 'https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css'
    } else {
      link.href = 'https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github.min.css'
    }

    document.head.appendChild(link)
  }, [resolvedTheme])

  return null
}
