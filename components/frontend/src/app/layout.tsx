import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Navigation } from "@/components/navigation";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Ambient Agentic Runner",
  description:
    "Kubernetes application for running automated agentic sessions with Ambient AI",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className={`${inter.className} min-h-screen flex flex-col overflow-hidden`}>
        <Navigation />
        <main className="flex-1 bg-background overflow-hidden">{children}</main>
      </body>
    </html>
  );
}
