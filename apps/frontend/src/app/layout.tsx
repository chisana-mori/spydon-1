import type { Metadata, Viewport } from "next";
import localFont from "next/font/local";
// import { Dancing_Script } from "next/font/google"; // 完全移除 Google Fonts 依赖
import "./globals.css";
import { Toaster } from "@/components/ui/sonner";
import { ThemeInitializer } from "@/components/theme-initializer";
import { Providers } from './providers'
import { ConditionalShell } from '@/components/layout/conditional-shell'
import { RuntimeConfigLoader } from '@/components/runtime-config-loader'

// 使用本地 Maple Mono 字体
const mapleMono = localFont({
  src: "./fonts/MapleMono-VF.ttf",
  variable: "--font-sans", // 替换原有的 font-sans 变量，保持全站等宽风格
  weight: "100 900",
});

/*
// 暂时禁用的 Google Fonts
const sourceCodePro = Source_Code_Pro({
  subsets: ["latin"],
  weight: "400",
  variable: "--font-sans",
});
*/


// 使用本地 Dancing Script 字体 (暂时使用 MapleMono 占位，请后续替换为真实字体文件)
const dancingScript = localFont({
  src: "./fonts/DancingScript.ttf",
  variable: "--font-handwriting",
  weight: "700",
});

/*
const dancingScript = Dancing_Script({
  subsets: ['latin'],
  weight: '700',
  variable: '--font-handwriting',
});
*/

export const metadata: Metadata = {
  title: "Spydon · 智能告警管理平台",
  description: "基于Next.js的现代化运维告警管理平台，提供多集群监控、告警分析与根因定位能力",
  keywords: ["告警管理", "Kubernetes", "根因分析", "运维自动化", "监控平台"],
  authors: [{ name: "Spydon Team" }],
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
      <body className={`${mapleMono.variable} ${dancingScript.variable} font-sans antialiased min-h-screen bg-background`}>
        <ThemeInitializer />
        <RuntimeConfigLoader>
          <Providers>
            <ConditionalShell>{children}</ConditionalShell>
          </Providers>
        </RuntimeConfigLoader>
        <Toaster position="top-right" expand richColors />
      </body>
    </html>
  );
}
