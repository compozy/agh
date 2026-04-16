import "./global.css";
import { RootProvider } from "fumadocs-ui/provider/next";
import { Inter, JetBrains_Mono, Playfair_Display } from "next/font/google";
import type { ReactNode } from "react";
import type { Metadata } from "next";

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
  display: "swap",
});

const playfairDisplay = Playfair_Display({
  subsets: ["latin"],
  variable: "--font-display",
  display: "swap",
  weight: ["400", "500"],
});

const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-jetbrains-mono",
  display: "swap",
});

export const metadata: Metadata = {
  title: {
    default: "AGH",
    template: "%s | AGH",
  },
  description: "Durable runtime and open coordination layer for real agent work.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html
      lang="en"
      className={`dark ${inter.variable} ${playfairDisplay.variable} ${jetbrainsMono.variable}`}
      suppressHydrationWarning
    >
      <body className="flex min-h-screen flex-col bg-fd-background font-sans text-fd-foreground antialiased">
        <RootProvider
          theme={{
            defaultTheme: "dark",
            forcedTheme: "dark",
            enabled: false,
          }}
        >
          {children}
        </RootProvider>
      </body>
    </html>
  );
}
