import "./global.css";
import { RootProvider } from "fumadocs-ui/provider/next";
import { Navbar } from "@/components/navbar";
import CustomSearchDialog from "@/components/search";
import type { ReactNode } from "react";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: {
    default: "AGH",
    template: "%s | AGH",
  },
  description:
    "Agent Operating System — spawn, observe, and orchestrate AI agent sessions via ACP.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className="dark" suppressHydrationWarning>
      <body
        className="bg-[var(--color-canvas)] font-sans text-[var(--color-text-primary)] antialiased"
        style={{ minHeight: "100vh" }}
      >
        <RootProvider
          theme={{
            defaultTheme: "dark",
            forcedTheme: "dark",
          }}
          search={{
            SearchDialog: CustomSearchDialog,
          }}
        >
          <Navbar />
          {children}
        </RootProvider>
      </body>
    </html>
  );
}
