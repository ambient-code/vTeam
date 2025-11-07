import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Navigation } from "@/components/navigation";
import { QueryProvider } from "@/components/providers/query-provider";
import { Toaster } from "@/components/ui/toaster";
import { VersionFooter } from "@/components/version-footer";
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
    <html lang="en" suppressHydrationWarning>
      <head>
        <meta name="backend-ws-base" content={wsBase} />
      </head>
      <body className={`${inter.className} min-h-screen flex flex-col`} suppressHydrationWarning>
        <QueryProvider>
          <Navigation feedbackUrl={feedbackUrl} />
          <main className="flex-1 bg-background overflow-auto">{children}</main>
          <VersionFooter />
          <Toaster />
        </QueryProvider>
      </body>
    </html>
  );
}
