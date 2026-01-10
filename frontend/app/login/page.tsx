'use client';

import { Suspense, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import { Button } from '@/app/components/ui/button';
import { AlertCircle } from 'lucide-react';
import { Alert, AlertDescription } from '@/app/components/ui/alert';

function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const error = searchParams.get('error');
  const { user, loading, login } = useAuth();

  useEffect(() => {
    if (user && !loading) {
      router.push('/');
    }
  }, [user, loading, router]);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-white dark:bg-[#0d0d0d]">
        <div className="w-10 h-10 border-2 border-gray-200 dark:border-gray-800 border-t-gray-600 dark:border-t-gray-400 rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-white dark:bg-[#0d0d0d] p-6">
      <div className="w-full max-w-[400px] space-y-8 flex flex-col items-center">
        {/* Logo */}
        <div className="w-12 h-12 flex items-center justify-center rounded-full bg-black dark:bg-white text-white dark:text-black">
          <span className="text-2xl">🤖</span>
        </div>

        <div className="text-center space-y-2">
          <h1 className="text-[32px] font-bold tracking-tight text-[#2d333a] dark:text-white">
            欢迎回来
          </h1>
          <p className="text-gray-500 dark:text-gray-400">
            登录 AIGuide 以继续
          </p>
        </div>

        <div className="w-full space-y-4">
          {error === 'unauthorized' && (
            <Alert variant="destructive" className="border-none bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 rounded-xl">
              <div className="flex items-center gap-2">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription className="text-sm font-medium">
                  您的邮箱不在允许列表中，请联系管理员。
                </AlertDescription>
              </div>
            </Alert>
          )}

          <Button
            onClick={login}
            className="w-full h-[52px] bg-[#10a37f] hover:bg-[#1a7f64] text-white rounded-md text-base font-medium transition-colors shadow-none"
          >
            使用 Google 账号继续
          </Button>
        </div>

        <div className="flex items-center gap-2 text-sm">
          <span className="text-gray-500 dark:text-gray-400 text-xs">
            还没有账号？请联系系统管理员。
          </span>
        </div>
      </div>

      {/* Footer Branding */}
      <footer className="absolute bottom-8 flex items-center gap-4 grayscale opacity-50 hover:opacity-100 transition-opacity">
        <div className="flex items-center gap-1.5 grayscale">
          <span className="text-sm font-semibold tracking-tighter">AIGuide</span>
        </div>
        <div className="w-px h-3 bg-gray-300 dark:bg-gray-700"></div>
        <span className="text-xs text-gray-500">Terms of use</span>
        <div className="w-px h-3 bg-gray-300 dark:bg-gray-700"></div>
        <span className="text-xs text-gray-500">Privacy policy</span>
      </footer>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center bg-white dark:bg-[#0d0d0d]">
        <div className="w-10 h-10 border-2 border-gray-200 dark:border-gray-800 border-t-gray-600 dark:border-t-gray-400 rounded-full animate-spin"></div>
      </div>
    }>
      <LoginForm />
    </Suspense>
  );
}
