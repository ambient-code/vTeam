"use client"

import * as React from "react"
import { Moon, Sun, Monitor } from "lucide-react"
import { useTheme } from "next-themes"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

export function ThemeToggle() {
  const { setTheme, theme } = useTheme()
  const [announcement, setAnnouncement] = React.useState("")
  const timeoutRef = React.useRef<NodeJS.Timeout | null>(null)

  // Cleanup timeout on unmount
  React.useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [])

  const handleThemeChange = (newTheme: string) => {
    setTheme(newTheme)

    // Clear any existing timeout
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }

    // Create accessible announcement for screen readers
    let message = ""
    if (newTheme === "system") {
      const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
      message = `Theme changed to system preference (currently ${systemTheme} mode)`
    } else {
      message = `Theme changed to ${newTheme} mode`
    }

    setAnnouncement(message)

    // Clear announcement after sufficient time for screen readers (3 seconds)
    // This ensures screen readers have time to queue and announce the message
    timeoutRef.current = setTimeout(() => {
      setAnnouncement("")
      timeoutRef.current = null
    }, 3000)
  }

  return (
    <div className="relative">
      {/* ARIA live region for announcing theme changes to screen readers */}
      <div
        role="status"
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      >
        {announcement}
      </div>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="h-9 w-9"
            aria-label="Toggle theme"
          >
            <Sun className="h-[1.2rem] w-[1.2rem] rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
            <Moon className="absolute h-[1.2rem] w-[1.2rem] rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
            <span className="sr-only">Toggle theme</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem
            onClick={() => handleThemeChange("light")}
            className="cursor-pointer"
            aria-label="Switch to light theme"
          >
            <Sun className="mr-2 h-4 w-4" aria-hidden="true" />
            <span>Light</span>
            {theme === "light" && <span className="ml-auto" aria-label="Currently selected">✓</span>}
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => handleThemeChange("dark")}
            className="cursor-pointer"
            aria-label="Switch to dark theme"
          >
            <Moon className="mr-2 h-4 w-4" aria-hidden="true" />
            <span>Dark</span>
            {theme === "dark" && <span className="ml-auto" aria-label="Currently selected">✓</span>}
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => handleThemeChange("system")}
            className="cursor-pointer"
            aria-label="Switch to system theme preference"
          >
            <Monitor className="mr-2 h-4 w-4" aria-hidden="true" />
            <span>System</span>
            {theme === "system" && <span className="ml-auto" aria-label="Currently selected">✓</span>}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
