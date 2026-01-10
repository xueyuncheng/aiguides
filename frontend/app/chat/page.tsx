'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

/**
 * 重定向页面：当用户访问 /chat 时
 * 自动生成一个新的 sessionId 并重定向到 /chat/[sessionId]
 */
export default function ChatRedirectPage() {
  const router = useRouter();

  useEffect(() => {
    // 生成随机的 sessionId
    const sessionId = `session-${Date.now()}-${Math.random().toString(36).substring(7)}`;

    // 立即重定向到带 sessionId 的页面
    router.replace(`/chat/${sessionId}`);
  }, [router]);

  // 显示加载状态
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
    </div>
  );
}
