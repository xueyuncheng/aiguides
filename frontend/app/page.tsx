'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';

/**
 * 主页：直接重定向到 Assistant AI Agent 聊天页面
 */
export default function Home() {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (!loading) {
      if (!user) {
        // 未登录，跳转到登录页
        router.push('/login');
      } else {
        // 已登录，直接跳转到聊天页面
        router.push('/chat');
      }
    }
  }, [user, loading, router]);

  // 显示加载状态
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="text-center">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-3"></div>
        <p className="text-sm text-muted-foreground">加载中...</p>
      </div>
    </div>
  );
}
