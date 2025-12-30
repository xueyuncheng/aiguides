import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "AIGuide - AI 助手平台",
  description: "基于 Google ADK 构建的智能助手服务，提供信息检索、网页总结、邮件分析和旅游规划等功能",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN">
      <body className="antialiased">
        {children}
      </body>
    </html>
  );
}
