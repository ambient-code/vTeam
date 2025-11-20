import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Navigation } from "@/components/navigation";
import { QueryProvider } from "@/components/providers/query-provider";
import { ThemeProvider } from "@/components/providers/theme-provider";
import { SyntaxThemeProvider } from "@/components/providers/syntax-theme-provider";
import { Toaster } from "@/components/ui/toaster";
import { env } from "@/lib/env";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Ambient Code Platform",
  description:
    "ACP is an AI-native agentic-powered enterprise software development platform",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const wsBase = env.BACKEND_URL.replace(/^http:/, 'ws:').replace(/^https:/, 'wss:')
  const feedbackUrl = env.FEEDBACK_URL
  return (
    // suppressHydrationWarning is required for next-themes to prevent hydration mismatch
    // between server-rendered content and client-side theme application
    <html lang="en" suppressHydrationWarning>
      <head>
        <meta name="backend-ws-base" content={wsBase} />
        {/*
          FOUC Prevention - Blocking Script (Pre-Hydration)

          This script runs BEFORE React hydration to prevent Flash of Unstyled Content.
          It sets the initial theme state synchronously during HTML parsing.

          Why this is needed:
          1. Executes before CSS loads and before React hydration
          2. Prevents visible flash when switching from light to dark (or vice versa)
          3. Sets both 'dark' class AND 'data-hljs-theme' attribute immediately

          Works in tandem with SyntaxThemeProvider:
          - This script: Initial page load (pre-hydration) ← You are here
          - SyntaxThemeProvider: Runtime theme changes (post-hydration)

          Both set the same attributes but at different lifecycle stages:
          - Blocking script: Runs during HTML parsing (before first paint)
          - SyntaxThemeProvider: Runs after React hydration (responds to theme toggle)

          This duplication is intentional and necessary for a flicker-free experience.

          Test Environment Handling:
          - Skips execution in Cypress (window.Cypress detected) to avoid hydration mismatches
          - In tests, next-themes will handle theme after hydration (acceptable tradeoff)
        */}
        <script
          dangerouslySetInnerHTML={{
            __html: `
              (function() {
                try {
                  // Skip in test environments to avoid hydration mismatches
                  // Tests will rely on next-themes to set theme after hydration
                  if (typeof window !== 'undefined' && window.Cypress) {
                    return;
                  }

                  // Check for theme in localStorage (next-themes default key)
                  var storedTheme = localStorage.getItem('theme');
                  var systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';

                  // Determine effective theme: use stored preference, fallback to system
                  var effectiveTheme = storedTheme === 'system' || !storedTheme ? systemTheme : storedTheme;

                  // Apply dark class immediately (before any paint/CSS loading)
                  if (effectiveTheme === 'dark') {
                    document.documentElement.classList.add('dark');
                  }

                  // Set attribute for syntax highlighting (our custom implementation)
                  // NOTE: SyntaxThemeProvider will update this reactively after hydration
                  document.documentElement.setAttribute('data-hljs-theme', effectiveTheme);
                } catch (e) {
                  // Graceful degradation: if script fails, React will set theme after hydration
                  // This ensures the app still works even if localStorage is blocked or script errors
                }
              })();
            `,
          }}
        />
      </head>
      {/* suppressHydrationWarning is needed here as well since ThemeProvider modifies the class attribute */}
      <body className={`${inter.className} min-h-screen flex flex-col`} suppressHydrationWarning>
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          <SyntaxThemeProvider />
          <QueryProvider>
            <Navigation feedbackUrl={feedbackUrl} />
            <main className="flex-1 bg-background overflow-auto">{children}</main>
            <Toaster />
          </QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
