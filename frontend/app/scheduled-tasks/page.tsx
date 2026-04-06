'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import { Button } from '@/app/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/app/components/ui/card';
import { Alert, AlertDescription } from '@/app/components/ui/alert';
import { CalendarClock, ChevronLeft, Trash2 } from 'lucide-react';
import { cn } from '@/app/lib/utils';

type ScheduleType = 'once' | 'daily' | 'weekly';

interface ScheduledTaskInfo {
  id: number;
  title: string;
  action: string;
  schedule_type: ScheduleType;
  run_at: string;
  weekday: number;
  timezone: string;
  target_email?: string;
  enabled: boolean;
  last_run_at?: string;
  next_run_at: string;
  created_at: string;
  updated_at: string;
}

interface ListScheduledTasksResponse {
  tasks: ScheduledTaskInfo[];
  total: number;
}

type ToastType = 'success' | 'error' | 'info';

interface ToastMessage {
  id: number;
  message: string;
  type: ToastType;
}

const WEEKDAY_LABELS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];

function getScheduleTypeLabel(type: ScheduleType) {
  switch (type) {
    case 'once':
      return '单次';
    case 'daily':
      return '每日';
    case 'weekly':
      return '每周';
    default:
      return type;
  }
}

function formatRunAt(task: ScheduledTaskInfo) {
  if (task.schedule_type === 'once') {
    return formatDate(task.run_at);
  }
  if (task.schedule_type === 'weekly') {
    return `${WEEKDAY_LABELS[task.weekday] ?? ''} ${task.run_at}`;
  }
  return task.run_at;
}

function formatDate(value?: string) {
  if (!value) {
    return '暂无';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}

function getStatusBadge(task: ScheduledTaskInfo) {
  if (!task.enabled) {
    return { label: '已停用', className: 'border-zinc-700 text-zinc-400' };
  }
  const now = new Date();
  const nextRun = new Date(task.next_run_at);
  if (task.schedule_type !== 'once' && nextRun < now) {
    return { label: '过期', className: 'border-amber-800 text-amber-400' };
  }
  return { label: '运行中', className: 'border-emerald-800 text-emerald-400' };
}

export default function ScheduledTasksPage() {
  const router = useRouter();
  const { user, loading, authenticatedFetch } = useAuth();
  const [tasks, setTasks] = useState<ScheduledTaskInfo[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [toasts, setToasts] = useState<ToastMessage[]>([]);
  const [backPath, setBackPath] = useState('/chat');
  const toastIdRef = useRef(0);

  useEffect(() => {
    const last = localStorage.getItem('aiguide:lastChatPath');
    if (last) {
      setBackPath(last);
    }
  }, []);

  const notify = (message: string, type: ToastType = 'info') => {
    const id = toastIdRef.current++;
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 3000);
  };

  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
    }
  }, [loading, router, user]);

  const loadTasks = useCallback(async () => {
    const response = await authenticatedFetch('/api/assistant/scheduled-tasks');
    if (!response.ok) {
      throw new Error('加载定时任务失败');
    }
    const data: ListScheduledTasksResponse = await response.json();
    setTasks(data.tasks || []);
  }, [authenticatedFetch]);

  const refreshData = useCallback(async () => {
    setIsLoading(true);
    setErrorMessage('');
    try {
      await loadTasks();
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '加载定时任务失败');
    } finally {
      setIsLoading(false);
    }
  }, [loadTasks]);

  useEffect(() => {
    if (!user) {
      return;
    }
    refreshData();
  }, [refreshData, user]);

  const handleDelete = async (task: ScheduledTaskInfo) => {
    if (!confirm(`确定删除定时任务「${task.title}」吗？`)) {
      return;
    }
    try {
      const response = await authenticatedFetch(`/api/assistant/scheduled-tasks/${task.id}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        const data = await response.json().catch(() => ({ error: '删除失败' }));
        throw new Error(data.error || '删除失败');
      }
      notify('定时任务已删除', 'success');
      await refreshData();
    } catch (error) {
      notify(error instanceof Error ? error.message : '删除失败', 'error');
    }
  };

  const handleToggleEnabled = async (task: ScheduledTaskInfo) => {
    try {
      const response = await authenticatedFetch(`/api/assistant/scheduled-tasks/${task.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: !task.enabled }),
      });
      if (!response.ok) {
        const data = await response.json().catch(() => ({ error: '更新失败' }));
        throw new Error(data.error || '更新失败');
      }
      notify(task.enabled ? '定时任务已停用' : '定时任务已启用', 'success');
      await refreshData();
    } catch (error) {
      notify(error instanceof Error ? error.message : '更新失败', 'error');
    }
  };

  if (loading || !user) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-950 text-zinc-200">
        <div className="rounded-full border border-zinc-800 bg-zinc-900 px-4 py-2 text-sm">正在加载定时任务...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <Link href={backPath} className="mb-3 inline-flex items-center gap-2 text-sm text-zinc-400 transition hover:text-zinc-100">
              <ChevronLeft className="h-4 w-4" />
              返回聊天
            </Link>
            <div>
              <h1 className="text-2xl font-semibold tracking-tight text-white">定时任务</h1>
              <p className="mt-1 text-sm text-zinc-400">在聊天中创建的定时任务会在这里显示，可以管理启停和删除。</p>
            </div>
          </div>
        </div>

        {errorMessage && (
          <Alert className="mb-6 border-red-900 bg-red-950/40 text-red-100">
            <AlertDescription>{errorMessage}</AlertDescription>
          </Alert>
        )}

        <div className="space-y-4">
          <Card className="border-zinc-800 bg-zinc-900/55 shadow-none">
            <CardHeader>
              <CardTitle className="text-lg text-white">任务列表</CardTitle>
              <CardDescription className="text-zinc-500">
                共 {tasks.length} 个定时任务
              </CardDescription>
            </CardHeader>
            <CardContent className="px-0 pb-0">
              {isLoading ? (
                <div className="mx-6 mb-6 rounded-xl border border-zinc-800 bg-zinc-950 px-4 py-8 text-center text-sm text-zinc-500">
                  正在加载任务...
                </div>
              ) : tasks.length === 0 ? (
                <div className="mx-6 mb-6 rounded-xl border border-dashed border-zinc-800 bg-zinc-950 px-6 py-10 text-center">
                  <div className="mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-full border border-zinc-800 bg-zinc-900 text-zinc-400">
                    <CalendarClock className="h-4 w-4" />
                  </div>
                  <div className="text-sm text-zinc-300">暂无定时任务</div>
                  <div className="mt-2 text-sm text-zinc-500">在聊天中告诉助手帮你创建定时任务即可，例如&ldquo;每天早上 8 点发送市场摘要到邮箱&rdquo;。</div>
                </div>
              ) : (
                <div className="divide-y divide-zinc-800 border-t border-zinc-800">
                  {tasks.map((task) => {
                    const badge = getStatusBadge(task);
                    return (
                      <div key={task.id} className="px-6 py-5 transition hover:bg-zinc-950/40">
                        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                          <div className="min-w-0 flex-1">
                            <div className="mb-2 flex flex-wrap items-center gap-2">
                              <span className={cn('rounded-full border px-2.5 py-1 text-xs', badge.className)}>
                                {badge.label}
                              </span>
                              <span className="rounded-full border border-zinc-700 px-2.5 py-1 text-xs text-zinc-300">
                                {getScheduleTypeLabel(task.schedule_type)}
                              </span>
                            </div>
                            <p className="font-medium text-sm text-zinc-100">{task.title}</p>
                            <p className="mt-1 text-xs text-zinc-400 line-clamp-2">{task.action}</p>
                            <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-zinc-500">
                              <span>执行时间：{formatRunAt(task)}</span>
                              <span>下次执行：{formatDate(task.next_run_at)}</span>
                              {task.last_run_at && <span>上次执行：{formatDate(task.last_run_at)}</span>}
                              {task.target_email && <span>目标邮箱：{task.target_email}</span>}
                              <span>时区：{task.timezone}</span>
                            </div>
                          </div>

                          <div className="flex shrink-0 gap-2">
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => handleToggleEnabled(task)}
                              className="text-zinc-300 hover:bg-zinc-800 hover:text-zinc-100"
                            >
                              {task.enabled ? '停用' : '启用'}
                            </Button>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => handleDelete(task)}
                              className="text-zinc-300 hover:bg-zinc-800 hover:text-zinc-100"
                            >
                              <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                              删除
                            </Button>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      <div className="pointer-events-none fixed right-4 top-4 z-50 space-y-2">
        {toasts.map((toast) => {
          const toneClass =
            toast.type === 'success'
              ? 'border-emerald-900 bg-emerald-950/80 text-emerald-100'
              : toast.type === 'error'
                ? 'border-red-900 bg-red-950/80 text-red-100'
                : 'border-zinc-700 bg-zinc-900 text-zinc-100';

          return (
            <div
              key={toast.id}
              className={cn(
                'pointer-events-auto flex min-w-[220px] items-center gap-3 rounded-xl border px-4 py-3 text-sm shadow-lg',
                toneClass
              )}
            >
              <span className="h-2 w-2 rounded-full bg-current opacity-80" />
              <span>{toast.message}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
