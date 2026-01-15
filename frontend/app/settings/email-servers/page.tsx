'use client';

import { useState, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import { Button } from '@/app/components/ui/button';
import { Input } from '@/app/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/app/components/ui/card';
import { Alert, AlertDescription } from '@/app/components/ui/alert';
import {
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

export default function EmailServersPage() {
  const router = useRouter();
  const { user, loading } = useAuth();
  const [configs, setConfigs] = useState<EmailServerConfig[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [formData, setFormData] = useState<FormData>({
    server: '',
    username: '',
    password: '',
    mailbox: 'INBOX',
    name: '',
    is_default: false,
  });
  const [isLoading, setIsLoading] = useState(false);
  const [toasts, setToasts] = useState<ToastMessage[]>([]);
  const toastId = useRef(0);

  const notify = (message: string, type: ToastType = 'info') => {
    const id = toastId.current++;
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 3200);
  };

  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
    }
  }, [user, loading, router]);

  useEffect(() => {
    if (user) {
      loadConfigs();
    }
  }, [user]);

  const loadConfigs = async () => {
    try {
      const response = await fetch('/api/email_server_configs', {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await response.json();
        setConfigs(data.configs || []);
      } else {
        const msg = '加载邮件服务器配置失败';
        notify(msg, 'error');
      }
    } catch (err) {
      const msg = '加载邮件服务器配置失败: ' + (err as Error).message;
      notify(msg, 'error');
    }
  };

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
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(formData),
      });

      if (response.ok) {
        const msg = editingId ? '更新成功' : '添加成功';
        notify(msg, 'success');
        setShowForm(false);
        setEditingId(null);
        resetForm();
        await loadConfigs();
      } else {
        const data = await response.json();
        const msg = data.error || '操作失败';
        notify(msg, 'error');
      }
    } catch (err) {
      const msg = '操作失败: ' + (err as Error).message;
      notify(msg, 'error');
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
        const msg = data.error || '加载配置失败';
        notify(msg, 'error');
        return;
      }

      const detail: EmailServerConfig = await response.json();

      setFormData({
        server: detail.server,
        username: detail.username,
        password: detail.password || '',
        mailbox: detail.mailbox || 'INBOX',
        name: detail.name,
        is_default: detail.is_default,
      });

      setEditingId(detail.id);
      setShowForm(true);
    } catch (err) {
      const msg = '加载配置失败: ' + (err as Error).message;
      notify(msg, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个邮件服务器配置吗？')) {
      return;
    }

    try {
      const response = await fetch(`/api/email_server_configs/${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });

      if (response.ok) {
        const msg = '删除成功';
        notify(msg, 'success');
        await loadConfigs();
      } else {
        const data = await response.json();
        const msg = data.error || '删除失败';
        notify(msg, 'error');
      }
    } catch (err) {
      const msg = '删除失败: ' + (err as Error).message;
      notify(msg, 'error');
    }
  };

  const resetForm = () => {
    setFormData({
      server: '',
      username: '',
      password: '',
      mailbox: 'INBOX',
      name: '',
      is_default: false,
    });
    setEditingId(null);
    setShowPassword(false);
  };

  const handleCancel = () => {
    setShowForm(false);
    resetForm();
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
      <div className="container mx-auto px-4 py-8 max-w-4xl">
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
          {!showForm && (
            <Button onClick={() => setShowForm(true)}>
              <Plus className="h-4 w-4 mr-2" />
              添加服务器
            </Button>
          )}
        </div>

        <Alert className="mb-4 border-yellow-500 bg-yellow-50 dark:bg-yellow-950">
          <AlertDescription className="text-yellow-800 dark:text-yellow-200">
            <strong>安全提示：</strong>密码当前以明文形式存储。强烈建议使用应用专用密码而不是账户主密码。
          </AlertDescription>
        </Alert>

        {/* Toasts */}
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

        {showForm && (
          <Card className="mb-6">
            <CardHeader>
              <CardTitle>{editingId ? '编辑' : '添加'}邮件服务器</CardTitle>
              <CardDescription>
                输入 IMAP 服务器信息以连接您的邮箱
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-2">
                    配置名称 *
                  </label>
                  <Input
                    type="text"
                    value={formData.name}
                    onChange={(e) =>
                      setFormData({ ...formData, name: e.target.value })
                    }
                    placeholder="例如: 我的 Gmail"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-2">
                    服务器地址 *
                  </label>
                  <Input
                    type="text"
                    value={formData.server}
                    onChange={(e) =>
                      setFormData({ ...formData, server: e.target.value })
                    }
                    placeholder="例如: imap.gmail.com:993"
                    required
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    IMAP 服务器地址和端口，通常是 993
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-2">
                    邮箱账号 *
                  </label>
                  <Input
                    type="email"
                    value={formData.username}
                    onChange={(e) =>
                      setFormData({ ...formData, username: e.target.value })
                    }
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
                      onChange={(e) =>
                        setFormData({ ...formData, password: e.target.value })
                      }
                      placeholder={editingId ? '留空表示不修改' : '邮箱密码或应用专用密码'}
                      required={!editingId}
                      className="pr-10"
                    />
                    <button
                      type="button"
                      className="absolute inset-y-0 right-0 px-3 flex items-center text-muted-foreground hover:text-foreground"
                      onClick={() => setShowPassword((v) => !v)}
                    >
                      {showPassword ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
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
                    onChange={(e) =>
                      setFormData({ ...formData, mailbox: e.target.value })
                    }
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
                    onChange={(e) =>
                      setFormData({ ...formData, is_default: e.target.checked })
                    }
                    className="rounded"
                  />
                  <label htmlFor="is_default" className="text-sm">
                    设为默认邮箱
                  </label>
                </div>

                <div className="flex gap-2 pt-4">
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
                    <X className="h-4 w-4 mr-2" />
                    取消
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        )}

        <div className="space-y-4">
          {configs.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12 text-center">
                <Mail className="h-12 w-12 text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">
                  还没有配置邮件服务器
                </h3>
                <p className="text-muted-foreground mb-4">
                  添加邮件服务器配置后，AI 助手将能够帮您查询邮件
                </p>
                {!showForm && (
                  <Button onClick={() => setShowForm(true)}>
                    <Plus className="h-4 w-4 mr-2" />
                    添加第一个服务器
                  </Button>
                )}
              </CardContent>
            </Card>
          ) : (
            configs.map((config) => (
              <Card key={config.id}>
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <CardTitle className="text-lg">
                          {config.name}
                        </CardTitle>
                        {config.is_default && (
                          <span className="text-xs px-2 py-1 bg-primary text-primary-foreground rounded">
                            默认
                          </span>
                        )}
                      </div>
                      <CardDescription className="mt-1">
                        {config.username}
                      </CardDescription>
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
                        onClick={() => handleDelete(config.id)}
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
