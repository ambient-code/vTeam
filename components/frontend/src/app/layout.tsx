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
          Blocking script to prevent FOUC (Flash of Unstyled Content)
          This runs before any CSS or React hydration, ensuring the correct theme
          class is applied immediately based on user's stored preference or system setting.
          Must be inline and blocking to execute before first paint.
        */}
        <script
          dangerouslySetInnerHTML={{
            __html: `
              (function() {
                try {
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
