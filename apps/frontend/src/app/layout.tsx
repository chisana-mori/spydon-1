import type { Metadata, Viewport } from "next";
import { Source_Code_Pro, Dancing_Script } from "next/font/google";
import "./globals.css";
import { Toaster } from "@/components/ui/sonner";
import { ThemeInitializer } from "@/components/theme-initializer";
import { Providers } from './providers'
import { ConditionalShell } from '@/components/layout/conditional-shell'

const sourceCodePro = Source_Code_Pro({
  subsets: ["latin"],
  weight: "400",
  variable: "--font-sans",
});

const dancingScript = Dancing_Script({
  subsets: ['latin'],
  weight: '700',
  variable: '--font-handwriting',
});

export const metadata: Metadata = {
  title: "Robusta Hub · 智能告警管理平台",
  description: "基于Next.js的现代化运维告警管理平台，提供多集群监控、告警分析与根因定位能力",
  keywords: ["告警管理", "Kubernetes", "根因分析", "运维自动化", "监控平台"],
  authors: [{ name: "Robusta Team" }],
};

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <body className={`${sourceCodePro.variable} ${dancingScript.variable} font-sans antialiased min-h-screen bg-background`}>
        <ThemeInitializer />
        <Providers>
          <ConditionalShell>{children}</ConditionalShell>
        </Providers>
        <Toaster position="top-right" expand richColors />
      </body>
    </html>
  );
}
