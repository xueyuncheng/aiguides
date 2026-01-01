'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import { Button } from '@/app/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/app/components/ui/card';
import { Separator } from '@/app/components/ui/separator';

export default function LoginPage() {
  const router = useRouter();
  const { user, loading, login } = useAuth();

  useEffect(() => {
    // If already logged in, redirect to home
    if (user && !loading) {
      router.push('/');
    }
  }, [user, loading, router]);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
        <div className="text-center">
          <div className="w-16 h-16 border-4 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-muted-foreground">加载中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
      <div className="max-w-md w-full mx-4">
        <Card className="shadow-xl">
          <CardHeader className="text-center">
            <div className="inline-flex justify-center mb-4">
              <div className="p-4 bg-blue-100 dark:bg-blue-900 rounded-full">
                <span className="text-5xl">🤖</span>
              </div>
            </div>
            <CardTitle className="text-3xl mb-2">AIGuide</CardTitle>
            <CardDescription>AI 助手平台</CardDescription>
          </CardHeader>

          <CardContent className="space-y-6">
            {/* Login Info */}
            <p className="text-center text-muted-foreground">
              使用 Google 账号登录以访问 AI 助手服务
            </p>

            {/* Google Login Button */}
            <Button
              onClick={login}
              variant="outline"
              className="w-full h-11 gap-3"
              size="lg"
            >
              <svg className="w-6 h-6" viewBox="0 0 24 24">
                <path
                  fill="#4285F4"
                  d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                />
                <path
                  fill="#34A853"
                  d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                />
                <path
                  fill="#FBBC05"
                  d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                />
                <path
                  fill="#EA4335"
                  d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                />
              </svg>
              使用 Google 登录
            </Button>

            {/* Additional Info */}
            <p className="text-sm text-muted-foreground text-center">
              登录即表示您同意我们的服务条款和隐私政策
            </p>

            {/* Features */}
            <div className="pt-4">
              <Separator className="mb-4" />
              <h3 className="text-sm font-semibold mb-3 text-center">
                登录后您可以使用：
              </h3>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li className="flex items-center gap-2">
                  <span className="text-green-500">✓</span>
                  信息检索和事实核查助手
                </li>
                <li className="flex items-center gap-2">
                  <span className="text-green-500">✓</span>
                  网页内容分析助手
                </li>
                <li className="flex items-center gap-2">
                  <span className="text-green-500">✓</span>
                  邮件智能总结助手
                </li>
                <li className="flex items-center gap-2">
                  <span className="text-green-500">✓</span>
                  旅游规划助手
                </li>
              </ul>
            </div>
          </CardContent>
        </Card>

        {/* Footer */}
        <div className="mt-6 text-center">
          <p className="text-sm text-muted-foreground">
            基于 Google ADK 构建 | Powered by Google Gemini
          </p>
        </div>
      </div>
    </div>
  );
}
