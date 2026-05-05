'use client';

import { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import { Button } from '@/app/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/app/components/ui/card';
import { Alert, AlertDescription } from '@/app/components/ui/alert';
import { ArrowLeft, Calendar, CheckCircle2, AlertTriangle, Unlink } from 'lucide-react';

interface CalendarStatus {
  connected: boolean;
  email?: string;
}

export default function GoogleCalendarSettingsPage() {
  const router = useRouter();
  const { user } = useAuth();
  const [status, setStatus] = useState<CalendarStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStatus = useCallback(async () => {
    try {
      const res = await fetch('/api/calendar/status');
      if (!res.ok) throw new Error('Failed to fetch calendar status');
      const data: CalendarStatus = await res.json();
      setStatus(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (user) fetchStatus();
  }, [user, fetchStatus]);

  const handleRevoke = async () => {
    if (!confirm('确认断开 Google Calendar 授权？断开后 AI 将无法访问你的日历。')) return;
    setActionLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/calendar/status', { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to revoke calendar access');
      setStatus({ connected: false });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-background p-6">
      <div className="max-w-2xl mx-auto space-y-6">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="sm" onClick={() => router.back()}>
            <ArrowLeft className="h-4 w-4 mr-1" />
            返回
          </Button>
          <div className="flex items-center gap-2">
            <Calendar className="h-5 w-5" />
            <h1 className="text-xl font-semibold">Google Calendar</h1>
          </div>
        </div>

        {error && (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <Card>
          <CardHeader>
            <CardTitle>日历授权</CardTitle>
            <CardDescription>
              授权后，AI 助手可以帮你查询、创建和管理 Google Calendar 日历事件。
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {loading ? (
              <p className="text-sm text-muted-foreground">加载中...</p>
            ) : status?.connected ? (
              <div className="space-y-4">
                <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
                  <CheckCircle2 className="h-4 w-4" />
                  <span>已授权</span>
                  {status.email && (
                    <span className="text-muted-foreground">({status.email})</span>
                  )}
                </div>
                <div className="rounded-md bg-muted p-3 text-sm text-muted-foreground space-y-1">
                  <p>AI 助手现在可以：</p>
                  <ul className="list-disc list-inside space-y-0.5 ml-2">
                    <li>查看你的日历事件</li>
                    <li>创建和更新事件</li>
                    <li>删除事件</li>
                  </ul>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleRevoke}
                  disabled={actionLoading}
                  className="text-destructive hover:text-destructive"
                >
                  <Unlink className="h-4 w-4 mr-2" />
                  断开授权
                </Button>
              </div>
            ) : (
              <div className="space-y-4">
                <p className="text-sm text-muted-foreground">
                  尚未授权 Google Calendar。点击下方按钮，通过 Google 账号完成授权。
                </p>
                <Button asChild disabled={actionLoading}>
                  <a href="/api/auth/login/google/reauth">
                    <Calendar className="h-4 w-4 mr-2" />
                    连接 Google Calendar
                  </a>
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">使用示例</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>授权后，你可以在对话中直接说：</p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>「帮我看看明天有什么安排」</li>
              <li>「下周一下午 3 点帮我创建一个和 Alice 的会议」</li>
              <li>「把今天下午 2 点的会议推迟到 4 点」</li>
              <li>「列出本周所有的日历事件」</li>
            </ul>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
