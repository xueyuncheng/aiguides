'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import { Button } from '@/app/components/ui/button';
import { Input } from '@/app/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/app/components/ui/card';
import { Alert, AlertDescription } from '@/app/components/ui/alert';
import {
  AlertTriangle,
  Plus,
  Trash2,
  Edit2,
  Check,
  X,
  Mail,
  ArrowLeft,
  Eye,
  EyeOff,
} from 'lucide-react';

interface EmailServerConfig {
  id: number;
  server: string;
  smtp_server?: string;
  username: string;
  password?: string;
  mailbox: string;
  name: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

interface FormData {
  server: string;
  smtp_server: string;
  username: string;
  password: string;
  mailbox: string;
  name: string;
  is_default: boolean;
}

type ToastType = 'success' | 'error' | 'info';

interface ToastMessage {
  id: number;
  message: string;
  type: ToastType;
}

const DEFAULT_FORM: FormData = {
  server: '',
  smtp_server: '',
  username: '',
  password: '',
  mailbox: 'INBOX',
  name: '',
  is_default: false,
};

export default function EmailServersPage() {
  const router = useRouter();
  const { user, loading } = useAuth();
  const [configs, setConfigs] = useState<EmailServerConfig[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<EmailServerConfig | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [formData, setFormData] = useState<FormData>(DEFAULT_FORM);
  const [isLoading, setIsLoading] = useState(false);
  const [toasts, setToasts] = useState<ToastMessage[]>([]);
  const toastId = useRef(0);

  const notify = useCallback((message: string, type: ToastType = 'info') => {
    const id = toastId.current++;
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 3200);
  }, []);

  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
    }
  }, [user, loading, router]);

  // Close form modal on Escape
  useEffect(() => {
    if (!showForm) return;

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isLoading) {
        setShowForm(false);
        setFormData(DEFAULT_FORM);
        setEditingId(null);
        setShowPassword(false);
      }
    };

    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, [showForm, isLoading]);

  // Close delete modal on Escape
  useEffect(() => {
    if (!deleteTarget) return;

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && deletingId === null) {
        setDeleteTarget(null);
      }
    };

    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, [deleteTarget, deletingId]);

  const loadConfigs = useCallback(async () => {
    try {
      const response = await fetch('/api/email_server_configs', {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await response.json();
        setConfigs(data.configs || []);
      } else {
        notify('加载邮件服务器配置失败', 'error');
      }
    } catch (err) {
      notify('加载邮件服务器配置失败: ' + (err as Error).message, 'error');
    }
  }, [notify]);

  useEffect(() => {
    if (user) {
      loadConfigs();
    }
  }, [loadConfigs, user]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    try {
      const url = editingId
        ? `/api/email_server_configs/${editingId}`
        : '/api/email_server_configs';
      const method = editingId ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(formData),
      });

      if (response.ok) {
        notify(editingId ? '更新成功' : '添加成功', 'success');
        closeForm();
        await loadConfigs();
      } else {
        const data = await response.json();
        notify(data.error || '操作失败', 'error');
      }
    } catch (err) {
      notify('操作失败: ' + (err as Error).message, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleEdit = async (config: EmailServerConfig) => {
    setIsLoading(true);
    setShowPassword(false);

    try {
      const response = await fetch(`/api/email_server_configs/${config.id}`, {
        credentials: 'include',
      });

      if (!response.ok) {
        const data = await response.json();
        notify(data.error || '加载配置失败', 'error');
        return;
      }

      const detail: EmailServerConfig = await response.json();

      setFormData({
        server: detail.server,
        smtp_server: detail.smtp_server || '',
        username: detail.username,
        password: detail.password || '',
        mailbox: detail.mailbox || 'INBOX',
        name: detail.name,
        is_default: detail.is_default,
      });

      setEditingId(detail.id);
      setShowForm(true);
    } catch (err) {
      notify('加载配置失败: ' + (err as Error).message, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDelete = (config: EmailServerConfig) => {
    setDeleteTarget(config);
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;

    setDeletingId(deleteTarget.id);
    try {
      const response = await fetch(`/api/email_server_configs/${deleteTarget.id}`, {
        method: 'DELETE',
        credentials: 'include',
      });

      if (response.ok) {
        notify('删除成功', 'success');
        setDeleteTarget(null);
        await loadConfigs();
      } else {
        const data = await response.json();
        notify(data.error || '删除失败', 'error');
      }
    } catch (err) {
      notify('删除失败: ' + (err as Error).message, 'error');
    } finally {
      setDeletingId(null);
    }
  };

  const resetForm = () => {
    setFormData(DEFAULT_FORM);
    setEditingId(null);
    setShowPassword(false);
  };

  const closeForm = () => {
    setShowForm(false);
    resetForm();
  };

  const handleCancel = () => {
    closeForm();
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
      </div>
    );
  }

  if (!user) {
    return null;
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Add / Edit modal */}
      {showForm && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 px-4 backdrop-blur-sm"
          onClick={(e) => {
            if (e.target === e.currentTarget && !isLoading) handleCancel();
          }}
        >
          <div className="w-full max-w-lg overflow-hidden rounded-2xl border bg-background shadow-2xl">
            <div className="flex items-center justify-between border-b px-6 py-4">
              <div>
                <h2 className="text-lg font-semibold">
                  {editingId ? '编辑' : '添加'}邮件服务器
                </h2>
                <p className="text-sm text-muted-foreground mt-0.5">
                  输入 IMAP/SMTP 服务器信息以查询和发送邮件
                </p>
              </div>
              <button
                type="button"
                onClick={handleCancel}
                disabled={isLoading}
                className="rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors disabled:opacity-50"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="px-6 py-5 max-h-[75vh] overflow-y-auto">
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-2">
                    配置名称 *
                  </label>
                  <Input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="例如: 我的 Gmail"
                    required
                    autoFocus
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-2">
                    服务器地址 *
                  </label>
                  <Input
                    type="text"
                    value={formData.server}
                    onChange={(e) => setFormData({ ...formData, server: e.target.value })}
                    placeholder="例如: imap.gmail.com:993"
                    required
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    IMAP 服务器地址和端口，通常是 993
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-2">
                    SMTP 服务器地址
                  </label>
                  <Input
                    type="text"
                    value={formData.smtp_server}
                    onChange={(e) => setFormData({ ...formData, smtp_server: e.target.value })}
                    placeholder="例如: smtp.gmail.com:587"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    可选；配置后 AI 助手可使用该账号发送邮件，通常使用 587 或 465 端口
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-2">
                    邮箱账号 *
                  </label>
                  <Input
                    type="email"
                    value={formData.username}
                    onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                    placeholder="your.email@example.com"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-2">
                    密码 *
                  </label>
                  <div className="relative">
                    <Input
                      type={showPassword ? 'text' : 'password'}
                      value={formData.password}
                      onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                      placeholder={editingId ? '留空表示不修改' : '邮箱密码或应用专用密码'}
                      required={!editingId}
                      className="pr-10"
                    />
                    <button
                      type="button"
                      className="absolute inset-y-0 right-0 px-3 flex items-center text-muted-foreground hover:text-foreground"
                      onClick={() => setShowPassword((v) => !v)}
                    >
                      {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </button>
                  </div>
                  <p className="text-xs text-muted-foreground mt-1">
                    建议使用应用专用密码而不是账户密码
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-2">
                    邮箱文件夹
                  </label>
                  <Input
                    type="text"
                    value={formData.mailbox}
                    onChange={(e) => setFormData({ ...formData, mailbox: e.target.value })}
                    placeholder="INBOX"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    默认为 INBOX (收件箱)
                  </p>
                </div>

                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="is_default"
                    checked={formData.is_default}
                    onChange={(e) => setFormData({ ...formData, is_default: e.target.checked })}
                    className="rounded"
                  />
                  <label htmlFor="is_default" className="text-sm">
                    设为默认邮箱
                  </label>
                </div>

                <div className="flex gap-2 pt-2 border-t">
                  <Button type="submit" disabled={isLoading}>
                    {isLoading ? (
                      <>
                        <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin mr-2"></div>
                        处理中...
                      </>
                    ) : (
                      <>
                        <Check className="h-4 w-4 mr-2" />
                        {editingId ? '更新' : '添加'}
                      </>
                    )}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleCancel}
                    disabled={isLoading}
                  >
                    取消
                  </Button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirmation modal */}
      {deleteTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 px-4 backdrop-blur-sm">
          <div className="w-full max-w-md overflow-hidden rounded-2xl border border-red-100 bg-white shadow-2xl">
            <div className="bg-[linear-gradient(135deg,#fff1f2_0%,#ffffff_65%)] px-6 py-5">
              <div className="flex items-start gap-4">
                <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-red-100 text-red-600 shadow-sm">
                  <AlertTriangle className="h-5 w-5" />
                </div>
                <div className="space-y-1">
                  <h2 className="text-lg font-semibold text-slate-900">
                    删除邮件服务器配置？
                  </h2>
                  <p className="text-sm leading-6 text-slate-600">
                    这会移除 <span className="font-medium text-slate-900">{deleteTarget.name}</span>{' '}
                    的邮箱连接设置，AI 将无法继续使用这个账号查询或发送邮件。
                  </p>
                </div>
              </div>
            </div>

            <div className="space-y-4 px-6 py-5">
              <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600">
                <div className="font-medium text-slate-900">{deleteTarget.username}</div>
                <div className="mt-1">IMAP: {deleteTarget.server}</div>
                <div className="mt-1">SMTP: {deleteTarget.smtp_server || '未配置'}</div>
              </div>

              <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setDeleteTarget(null)}
                  disabled={deletingId !== null}
                >
                  取消
                </Button>
                <Button
                  type="button"
                  variant="destructive"
                  onClick={confirmDelete}
                  disabled={deletingId !== null}
                  className="min-w-28"
                >
                  {deletingId === deleteTarget.id ? '删除中...' : '确认删除'}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="container mx-auto px-4 py-8 max-w-4xl">
        {/* Page header */}
        <div className="mb-6 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => router.push('/chat')}
            >
              <ArrowLeft className="h-5 w-5" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold">邮件服务器配置</h1>
              <p className="text-muted-foreground mt-1">
                管理您的 IMAP 邮件服务器连接
              </p>
            </div>
          </div>
          <Button onClick={() => setShowForm(true)}>
            <Plus className="h-4 w-4 mr-2" />
            添加服务器
          </Button>
        </div>

        <Alert className="mb-4 border-yellow-500 bg-yellow-50 dark:bg-yellow-950">
          <AlertDescription className="text-yellow-800 dark:text-yellow-200">
            <strong>安全提示：</strong>密码当前以明文形式存储。强烈建议使用应用专用密码而不是账户主密码。
          </AlertDescription>
        </Alert>

        {/* Toast notifications */}
        <div className="fixed top-4 right-4 z-50 flex flex-col gap-2 w-80">
          {toasts.map((toast) => {
            const isSuccess = toast.type === 'success';
            const isError = toast.type === 'error';
            const base = 'rounded-md px-4 py-3 shadow-lg text-sm flex items-start gap-2 border';
            const theme = isSuccess
              ? 'bg-green-50 text-green-800 border-green-200'
              : isError
                ? 'bg-red-50 text-red-800 border-red-200'
                : 'bg-blue-50 text-blue-800 border-blue-200';
            return (
              <div key={toast.id} className={`${base} ${theme}`}>
                <span className="font-semibold">
                  {isSuccess ? '成功' : isError ? '错误' : '提示'}
                </span>
                <span className="flex-1">{toast.message}</span>
              </div>
            );
          })}
        </div>

        {/* Config list */}
        <div className="space-y-4">
          {configs.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12 text-center">
                <Mail className="h-12 w-12 text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">
                  还没有配置邮件服务器
                </h3>
                <p className="text-muted-foreground mb-4">
                  添加邮件服务器配置后，AI 助手将能够帮您查询邮件；补充 SMTP 地址后也可发送邮件
                </p>
                <Button onClick={() => setShowForm(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  添加第一个服务器
                </Button>
              </CardContent>
            </Card>
          ) : (
            configs.map((config) => (
              <Card key={config.id}>
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <CardTitle className="text-lg">{config.name}</CardTitle>
                        {config.is_default && (
                          <span className="text-xs px-2 py-1 bg-primary text-primary-foreground rounded">
                            默认
                          </span>
                        )}
                      </div>
                      <CardDescription className="mt-1">{config.username}</CardDescription>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleEdit(config)}
                      >
                        <Edit2 className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDelete(config)}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="text-muted-foreground">服务器：</span>
                      <span className="ml-2">{config.server}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">SMTP：</span>
                      <span className="ml-2">{config.smtp_server || '未配置'}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">邮箱文件夹：</span>
                      <span className="ml-2">{config.mailbox}</span>
                    </div>
                  </div>
                  <div className="mt-2 text-xs text-muted-foreground">
                    创建于 {new Date(config.created_at).toLocaleString('zh-CN')}
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
