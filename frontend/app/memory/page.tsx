'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import { Button } from '@/app/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/app/components/ui/card';
import { Alert, AlertDescription } from '@/app/components/ui/alert';
import { Brain, ChevronLeft, Pencil, Plus, Trash2 } from 'lucide-react';
import { cn } from '@/app/lib/utils';
import { MemoryEditorModal } from './MemoryEditorModal';

type MemoryType = 'fact' | 'preference' | 'context';

interface MemoryInfo {
  id: number;
  memory_type: MemoryType;
  content: string;
  importance: number;
  created_at: string;
  updated_at: string;
}

interface MemoryListResponse {
  memories: MemoryInfo[];
}

interface MemoryFormData {
  memory_type: MemoryType;
  content: string;
  importance: number;
}

type ToastType = 'success' | 'error' | 'info';

interface ToastMessage {
  id: number;
  message: string;
  type: ToastType;
}

const filters: Array<{ value: 'all' | MemoryType; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'fact', label: '事实' },
  { value: 'preference', label: '偏好' },
  { value: 'context', label: '上下文' },
];

const defaultFormData: MemoryFormData = {
  memory_type: 'fact',
  content: '',
  importance: 5,
};

function getMemoryTypeLabel(type: MemoryType) {
  switch (type) {
    case 'fact':
      return '事实';
    case 'preference':
      return '偏好';
    case 'context':
      return '上下文';
    default:
      return type;
  }
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

export default function MemoryPage() {
  const router = useRouter();
  const { user, loading, authenticatedFetch } = useAuth();
  const [memories, setMemories] = useState<MemoryInfo[]>([]);
  const [activeFilter, setActiveFilter] = useState<'all' | MemoryType>('all');
  const [showForm, setShowForm] = useState(false);
  const [editingMemoryId, setEditingMemoryId] = useState<number | null>(null);
  const [formData, setFormData] = useState<MemoryFormData>(defaultFormData);
  const [isLoading, setIsLoading] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [toasts, setToasts] = useState<ToastMessage[]>([]);
  const toastIdRef = useRef(0);

  const notify = (message: string, type: ToastType = 'info') => {
    const id = toastIdRef.current++;
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((toast) => toast.id !== id));
    }, 3000);
  };

  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
    }
  }, [loading, router, user]);

  const loadMemories = useCallback(async (keyword: string, type: 'all' | MemoryType) => {
    const params = new URLSearchParams();
    params.set('limit', '100');
    if (type !== 'all') {
      params.set('type', type);
    }
    if (keyword.trim()) {
      params.set('keyword', keyword.trim());
    }

    const response = await authenticatedFetch(`/api/assistant/memories?${params.toString()}`);
    if (!response.ok) {
      throw new Error('加载记忆列表失败');
    }
    const data: MemoryListResponse = await response.json();
    setMemories(data.memories || []);
  }, [authenticatedFetch]);

  const refreshData = useCallback(async (keyword: string, type: 'all' | MemoryType) => {
    setIsLoading(true);
    setErrorMessage('');
    try {
      await loadMemories(keyword, type);
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '加载记忆失败');
    } finally {
      setIsLoading(false);
    }
  }, [loadMemories]);

  useEffect(() => {
    if (!user) {
      return;
    }
    refreshData('', activeFilter);
  }, [activeFilter, refreshData, user]);

  const resetForm = () => {
    setFormData(defaultFormData);
    setEditingMemoryId(null);
    setShowForm(false);
  };

  const handleCreateClick = () => {
    setFormData(defaultFormData);
    setEditingMemoryId(null);
    setShowForm(true);
  };

  const handleEditClick = (memory: MemoryInfo) => {
    setEditingMemoryId(memory.id);
    setFormData({
      memory_type: memory.memory_type,
      content: memory.content,
      importance: memory.importance,
    });
    setShowForm(true);
  };

  const handleDelete = async (memory: MemoryInfo) => {
    if (!confirm(`确定删除这条${getMemoryTypeLabel(memory.memory_type)}记忆吗？`)) {
      return;
    }

    try {
      const response = await authenticatedFetch(`/api/assistant/memories/${memory.id}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        const data = await response.json().catch(() => ({ error: '删除记忆失败' }));
        throw new Error(data.error || '删除记忆失败');
      }

      notify('记忆已删除', 'success');
      await refreshData('', activeFilter);
    } catch (error) {
      notify(error instanceof Error ? error.message : '删除记忆失败', 'error');
    }
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsSubmitting(true);

    try {
      const payload = {
        memory_type: formData.memory_type,
        content: formData.content.trim(),
        importance: formData.importance,
      };

      const isEditing = editingMemoryId !== null;
      const response = await authenticatedFetch(
        isEditing ? `/api/assistant/memories/${editingMemoryId}` : '/api/assistant/memories',
        {
          method: isEditing ? 'PATCH' : 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        }
      );

      if (!response.ok) {
        const data = await response.json().catch(() => ({ error: '保存记忆失败' }));
        throw new Error(data.error || '保存记忆失败');
      }

      notify(isEditing ? '记忆已更新' : '记忆已新增', 'success');
      resetForm();
      await refreshData('', activeFilter);
    } catch (error) {
      notify(error instanceof Error ? error.message : '保存记忆失败', 'error');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (loading || !user) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-950 text-zinc-200">
        <div className="rounded-full border border-zinc-800 bg-zinc-900 px-4 py-2 text-sm">正在加载记忆中心...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <Link href="/chat" className="mb-3 inline-flex items-center gap-2 text-sm text-zinc-400 transition hover:text-zinc-100">
              <ChevronLeft className="h-4 w-4" />
              返回聊天
            </Link>
            <div>
              <h1 className="text-2xl font-semibold tracking-tight text-white">管理记忆</h1>
              <p className="mt-1 text-sm text-zinc-400">这些内容会帮助助手在后续对话里更了解你。</p>
            </div>
          </div>

          <div className="flex gap-3">
            <Button type="button" onClick={handleCreateClick} className="bg-zinc-100 text-zinc-950 hover:bg-zinc-200">
              <Plus className="mr-2 h-4 w-4" />
              新增记忆
            </Button>
          </div>
        </div>

        {errorMessage && (
          <Alert className="mb-6 border-red-900 bg-red-950/40 text-red-100">
            <AlertDescription>{errorMessage}</AlertDescription>
          </Alert>
        )}

        <div className="space-y-4">
            <Card className="border-zinc-800 bg-zinc-900/55 shadow-none">
              <CardHeader className="gap-4 pb-4">
                <div>
                  <CardTitle className="text-lg text-white">筛选</CardTitle>
                  <CardDescription className="text-zinc-500">按类型查看不同类别的记忆。</CardDescription>
                </div>

                <div className="flex flex-wrap gap-2">
                  {filters.map((filter) => {
                    const active = filter.value === activeFilter;
                    return (
                      <button
                        key={filter.value}
                        type="button"
                        onClick={() => setActiveFilter(filter.value)}
                        className={cn(
                          'rounded-full border px-3.5 py-1.5 text-sm transition',
                          active
                            ? 'border-zinc-100 bg-zinc-100 text-zinc-950'
                            : 'border-zinc-800 bg-zinc-950 text-zinc-300 hover:border-zinc-700 hover:text-zinc-100'
                        )}
                      >
                        {filter.label}
                      </button>
                    );
                  })}
                </div>
              </CardHeader>
            </Card>

            <Card className="border-zinc-800 bg-zinc-900/55 shadow-none">
              <CardHeader>
                <CardTitle className="text-lg text-white">记忆列表</CardTitle>
                <CardDescription className="text-zinc-500">
                  当前显示 {memories.length} 条记忆，按重要度和更新时间排序
                </CardDescription>
              </CardHeader>
              <CardContent className="px-0 pb-0">
                {isLoading ? (
                  <div className="mx-6 mb-6 rounded-xl border border-zinc-800 bg-zinc-950 px-4 py-8 text-center text-sm text-zinc-500">
                    正在加载记忆...
                  </div>
                ) : memories.length === 0 ? (
                  <div className="mx-6 mb-6 rounded-xl border border-dashed border-zinc-800 bg-zinc-950 px-6 py-10 text-center">
                    <div className="mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-full border border-zinc-800 bg-zinc-900 text-zinc-400">
                      <Brain className="h-4 w-4" />
                    </div>
                    <div className="text-sm text-zinc-300">当前没有符合条件的记忆</div>
                    <div className="mt-2 text-sm text-zinc-500">你可以手动新增一条，或者先在聊天里继续交流。</div>
                  </div>
                ) : (
                  <div className="divide-y divide-zinc-800 border-t border-zinc-800">
                    {memories.map((memory) => (
                      <div key={memory.id} className="px-6 py-5 transition hover:bg-zinc-950/40">
                        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                          <div className="min-w-0 flex-1">
                            <div className="mb-2 flex flex-wrap items-center gap-2">
                              <span className="rounded-full border border-zinc-700 px-2.5 py-1 text-xs text-zinc-200">
                                {getMemoryTypeLabel(memory.memory_type)}
                              </span>
                              <span className="text-xs text-zinc-500">重要度 {memory.importance}/10</span>
                            </div>
                            <p className="text-sm leading-7 text-zinc-100">{memory.content}</p>
                            <div className="mt-2 text-xs text-zinc-500">
                              更新于 {formatDate(memory.updated_at)}
                            </div>
                          </div>

                          <div className="flex shrink-0 gap-2">
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => handleEditClick(memory)}
                              className="text-zinc-300 hover:bg-zinc-800 hover:text-zinc-100"
                            >
                              <Pencil className="mr-1.5 h-3.5 w-3.5" />
                              编辑
                            </Button>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => handleDelete(memory)}
                              className="text-zinc-300 hover:bg-zinc-800 hover:text-zinc-100"
                            >
                              <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                              删除
                            </Button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
        </div>
      </div>

      <MemoryEditorModal
        isOpen={showForm}
        editingMemoryId={editingMemoryId}
        formData={formData}
        isSubmitting={isSubmitting}
        onClose={resetForm}
        onSubmit={handleSubmit}
        onChange={setFormData}
      />

      <div className="pointer-events-none fixed right-4 top-4 z-50 space-y-2">
        {toasts.map((toast) => {
          const toneClass = toast.type === 'success'
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
