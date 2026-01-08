'use client';

import { useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';

/**
 * 重定向页面：当用户访问 /chat/[agentId] 时
 * 自动生成一个新的 sessionId 并重定向到 /chat/[agentId]/[sessionId]
 */
export default function AgentRedirectPage() {
  const params = useParams();
  const router = useRouter();
  const agentId = params.agentId as string;

  useEffect(() => {
    // 生成随机的 sessionId
    const sessionId = `session-${Date.now()}-${Math.random().toString(36).substring(7)}`;

    // 立即重定向到带 sessionId 的页面
    router.replace(`/chat/${agentId}/${sessionId}`);
  }, [agentId, router]);

  // 显示加载状态
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
    </div>
  );
}
