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
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="w-6 h-6 border-2 border-muted border-t-foreground rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-background p-6">
      <div className="w-full max-w-[350px] space-y-6">
        <div className="flex flex-col space-y-2 text-center text-zinc-950 dark:text-zinc-50">
          <h1 className="text-2xl font-semibold tracking-tight">
            欢迎回来
          </h1>
          <p className="text-sm text-muted-foreground">
            登录 AIGuide 以继续使用智能助手
          </p>
        </div>

        <div className="grid gap-6">
          {error === 'unauthorized' && (
            <Alert variant="destructive" className="rounded-lg shadow-sm">
              <div className="flex items-center gap-2">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription className="text-xs">
                  您的邮箱不在允许列表中，请联系管理员。
                </AlertDescription>
              </div>
            </Alert>
          )}

          <Button
            onClick={login}
            className="w-full h-10 px-4 py-2 bg-primary text-primary-foreground hover:bg-primary/90 rounded-md text-sm font-medium transition-colors shadow-sm"
          >
            使用 Google 账号继续
          </Button>
        </div>

        <p className="px-8 text-center text-xs text-muted-foreground">
          还没有账号？请联系系统管理员获取访问权限。
        </p>
      </div>

      {/* Footer Branding */}
      <footer className="absolute bottom-8 flex items-center gap-4 text-muted-foreground/60 transition-colors hover:text-muted-foreground">
        <span className="text-xs font-medium tracking-tight">AIGuide</span>
        <div className="w-px h-3 bg-border"></div>
        <span className="text-[10px] uppercase tracking-widest">Terms of use</span>
        <div className="w-px h-3 bg-border"></div>
        <span className="text-[10px] uppercase tracking-widest">Privacy policy</span>
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
