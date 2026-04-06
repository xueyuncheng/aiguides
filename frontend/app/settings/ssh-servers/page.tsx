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
  Terminal,
  ArrowLeft,
  Eye,
  EyeOff,
} from 'lucide-react';

type AuthMethod = 'password' | 'key';

interface SSHServerConfig {
  id: number;
  name: string;
  host: string;
  port: number;
  username: string;
  auth_method: AuthMethod;
  password?: string;
  private_key?: string;
  passphrase?: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

interface FormData {
  name: string;
  host: string;
  port: number;
  username: string;
  auth_method: AuthMethod;
  password: string;
  private_key: string;
  passphrase: string;
  is_default: boolean;
}

type ToastType = 'success' | 'error' | 'info';

interface ToastMessage {
  id: number;
  message: string;
  type: ToastType;
}

const DEFAULT_FORM: FormData = {
  name: '',
  host: '',
  port: 22,
  username: '',
  auth_method: 'password',
  password: '',
  private_key: '',
  passphrase: '',
  is_default: false,
};

export default function SSHServersPage() {
  const router = useRouter();
  const { user, loading } = useAuth();
  const [configs, setConfigs] = useState<SSHServerConfig[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<SSHServerConfig | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [showPassphrase, setShowPassphrase] = useState(false);
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

  useEffect(() => {
    if (!deleteTarget) {
      return;
    }

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
      const response = await fetch('/api/ssh_server_configs', {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await response.json();
        setConfigs(data.configs || []);
      } else {
        notify('Failed to load SSH server configs', 'error');
      }
    } catch (err) {
      notify('Failed to load SSH server configs: ' + (err as Error).message, 'error');
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
        ? `/api/ssh_server_configs/${editingId}`
        : '/api/ssh_server_configs';
      const method = editingId ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(formData),
      });

      if (response.ok) {
        notify(editingId ? 'Updated successfully' : 'Added successfully', 'success');
        setShowForm(false);
        setEditingId(null);
        resetForm();
        await loadConfigs();
      } else {
        const data = await response.json();
        notify(data.error || 'Operation failed', 'error');
      }
    } catch (err) {
      notify('Operation failed: ' + (err as Error).message, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleEdit = async (config: SSHServerConfig) => {
    setIsLoading(true);
    setShowPassword(false);
    setShowPassphrase(false);

    try {
      const response = await fetch(`/api/ssh_server_configs/${config.id}`, {
        credentials: 'include',
      });

      if (!response.ok) {
        const data = await response.json();
        notify(data.error || 'Failed to load config', 'error');
        return;
      }

      const detail: SSHServerConfig = await response.json();

      setFormData({
        name: detail.name,
        host: detail.host,
        port: detail.port || 22,
        username: detail.username,
        auth_method: detail.auth_method || 'password',
        password: detail.password || '',
        private_key: detail.private_key || '',
        passphrase: detail.passphrase || '',
        is_default: detail.is_default,
      });

      setEditingId(detail.id);
      setShowForm(true);
    } catch (err) {
      notify('Failed to load config: ' + (err as Error).message, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDelete = (config: SSHServerConfig) => {
    setDeleteTarget(config);
  };

  const confirmDelete = async () => {
    if (!deleteTarget) {
      return;
    }

    setDeletingId(deleteTarget.id);
    try {
      const response = await fetch(`/api/ssh_server_configs/${deleteTarget.id}`, {
        method: 'DELETE',
        credentials: 'include',
      });

      if (response.ok) {
        notify('Deleted successfully', 'success');
        setDeleteTarget(null);
        await loadConfigs();
      } else {
        const data = await response.json();
        notify(data.error || 'Delete failed', 'error');
      }
    } catch (err) {
      notify('Delete failed: ' + (err as Error).message, 'error');
    } finally {
      setDeletingId(null);
    }
  };

  const resetForm = () => {
    setFormData(DEFAULT_FORM);
    setEditingId(null);
    setShowPassword(false);
    setShowPassphrase(false);
  };

  const handleCancel = () => {
    setShowForm(false);
    resetForm();
  };

  const handleAuthMethodChange = (method: AuthMethod) => {
    setFormData((prev) => ({ ...prev, auth_method: method, password: '', private_key: '', passphrase: '' }));
    setShowPassword(false);
    setShowPassphrase(false);
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
                    Delete SSH server config?
                  </h2>
                  <p className="text-sm leading-6 text-slate-600">
                    This removes the SSH connection for{' '}
                    <span className="font-medium text-slate-900">{deleteTarget.name}</span>.
                    The agent will no longer be able to run commands on this server.
                  </p>
                </div>
              </div>
            </div>

            <div className="space-y-4 px-6 py-5">
              <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600">
                <div className="font-medium text-slate-900">{deleteTarget.username}@{deleteTarget.host}</div>
                <div className="mt-1">Port: {deleteTarget.port}</div>
              </div>

              <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setDeleteTarget(null)}
                  disabled={deletingId !== null}
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  variant="destructive"
                  onClick={confirmDelete}
                  disabled={deletingId !== null}
                  className="min-w-28"
                >
                  {deletingId === deleteTarget.id ? 'Deleting...' : 'Confirm delete'}
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
              <h1 className="text-3xl font-bold">SSH Server Configs</h1>
              <p className="text-muted-foreground mt-1">
                Manage SSH connections the agent can use to run commands
              </p>
            </div>
          </div>
          {!showForm && (
            <Button onClick={() => setShowForm(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Add server
            </Button>
          )}
        </div>

        <Alert className="mb-4 border-yellow-500 bg-yellow-50 dark:bg-yellow-950">
          <AlertDescription className="text-yellow-800 dark:text-yellow-200">
            <strong>Security notice:</strong> Credentials are stored in plain text. Use a
            dedicated low-privilege account and avoid reusing important passwords or keys.
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
                  {isSuccess ? 'Success' : isError ? 'Error' : 'Info'}
                </span>
                <span className="flex-1">{toast.message}</span>
              </div>
            );
          })}
        </div>

        {/* Add / Edit form */}
        {showForm && (
          <Card className="mb-6">
            <CardHeader>
              <CardTitle>{editingId ? 'Edit' : 'Add'} SSH server</CardTitle>
              <CardDescription>
                Enter the connection details. The agent will use these credentials to run shell commands.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-2">
                    Config name *
                  </label>
                  <Input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="e.g. My dev server"
                    required
                  />
                </div>

                <div className="grid grid-cols-3 gap-4">
                  <div className="col-span-2">
                    <label className="block text-sm font-medium mb-2">
                      Host *
                    </label>
                    <Input
                      type="text"
                      value={formData.host}
                      onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                      placeholder="e.g. 192.168.1.10 or example.com"
                      required
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                      Hostname or IP address of the remote machine
                    </p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      Port *
                    </label>
                    <Input
                      type="number"
                      min={1}
                      max={65535}
                      value={formData.port}
                      onChange={(e) =>
                        setFormData({ ...formData, port: parseInt(e.target.value, 10) || 22 })
                      }
                      placeholder="22"
                      required
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-2">
                    Username *
                  </label>
                  <Input
                    type="text"
                    value={formData.username}
                    onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                    placeholder="e.g. ubuntu"
                    required
                  />
                </div>

                {/* Auth method toggle */}
                <div>
                  <label className="block text-sm font-medium mb-2">
                    Authentication method
                  </label>
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={() => handleAuthMethodChange('password')}
                      className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium transition-colors ${
                        formData.auth_method === 'password'
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-input bg-background hover:bg-accent hover:text-accent-foreground'
                      }`}
                    >
                      Password
                    </button>
                    <button
                      type="button"
                      onClick={() => handleAuthMethodChange('key')}
                      className={`flex-1 rounded-md border px-3 py-2 text-sm font-medium transition-colors ${
                        formData.auth_method === 'key'
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-input bg-background hover:bg-accent hover:text-accent-foreground'
                      }`}
                    >
                      Public key
                    </button>
                  </div>
                </div>

                {/* Password field */}
                {formData.auth_method === 'password' && (
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      Password{!editingId && ' *'}
                    </label>
                    <div className="relative">
                      <Input
                        type={showPassword ? 'text' : 'password'}
                        value={formData.password}
                        onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                        placeholder={editingId ? 'Leave blank to keep current password' : 'SSH password'}
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
                  </div>
                )}

                {/* Private key + passphrase fields */}
                {formData.auth_method === 'key' && (
                  <div className="space-y-4">
                    <div>
                      <label className="block text-sm font-medium mb-2">
                        Private key (PEM){!editingId && ' *'}
                      </label>
                      <textarea
                        value={formData.private_key}
                        onChange={(e) => setFormData({ ...formData, private_key: e.target.value })}
                        placeholder={
                          editingId
                            ? 'Leave blank to keep current key'
                            : '-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----'
                        }
                        required={!editingId}
                        rows={6}
                        className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring resize-y"
                      />
                      <p className="text-xs text-muted-foreground mt-1">
                        Paste your PEM-encoded private key (RSA, ECDSA, or Ed25519).
                      </p>
                    </div>

                    <div>
                      <label className="block text-sm font-medium mb-2">
                        Passphrase <span className="text-muted-foreground font-normal">(optional)</span>
                      </label>
                      <div className="relative">
                        <Input
                          type={showPassphrase ? 'text' : 'password'}
                          value={formData.passphrase}
                          onChange={(e) => setFormData({ ...formData, passphrase: e.target.value })}
                          placeholder={editingId ? 'Leave blank to keep current passphrase' : 'Key passphrase (if encrypted)'}
                          className="pr-10"
                        />
                        <button
                          type="button"
                          className="absolute inset-y-0 right-0 px-3 flex items-center text-muted-foreground hover:text-foreground"
                          onClick={() => setShowPassphrase((v) => !v)}
                        >
                          {showPassphrase ? (
                            <EyeOff className="h-4 w-4" />
                          ) : (
                            <Eye className="h-4 w-4" />
                          )}
                        </button>
                      </div>
                    </div>
                  </div>
                )}

                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="is_default"
                    checked={formData.is_default}
                    onChange={(e) => setFormData({ ...formData, is_default: e.target.checked })}
                    className="rounded"
                  />
                  <label htmlFor="is_default" className="text-sm">
                    Set as default server
                  </label>
                </div>

                <div className="flex gap-2 pt-4">
                  <Button type="submit" disabled={isLoading}>
                    {isLoading ? (
                      <>
                        <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin mr-2"></div>
                        Saving...
                      </>
                    ) : (
                      <>
                        <Check className="h-4 w-4 mr-2" />
                        {editingId ? 'Update' : 'Add'}
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
                    Cancel
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        )}

        {/* Config list */}
        <div className="space-y-4">
          {configs.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12 text-center">
                <Terminal className="h-12 w-12 text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">No SSH servers configured</h3>
                <p className="text-muted-foreground mb-4">
                  Add an SSH server so the agent can run shell commands on your machines.
                </p>
                {!showForm && (
                  <Button onClick={() => setShowForm(true)}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add your first server
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
                        <CardTitle className="text-lg">{config.name}</CardTitle>
                        {config.is_default && (
                          <span className="text-xs px-2 py-1 bg-primary text-primary-foreground rounded">
                            Default
                          </span>
                        )}
                      </div>
                      <CardDescription className="mt-1">
                        {config.username}@{config.host}
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
                      <span className="text-muted-foreground">Host:</span>
                      <span className="ml-2">{config.host}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Port:</span>
                      <span className="ml-2">{config.port}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Username:</span>
                      <span className="ml-2">{config.username}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Auth:</span>
                      <span className="ml-2 capitalize">
                        {config.auth_method === 'key' ? 'Public key' : 'Password'}
                      </span>
                    </div>
                  </div>
                  <div className="mt-2 text-xs text-muted-foreground">
                    Added {new Date(config.created_at).toLocaleString()}
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
