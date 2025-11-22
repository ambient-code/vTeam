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

// Force rebuild timestamp: 2025-11-20T16:38:00

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
