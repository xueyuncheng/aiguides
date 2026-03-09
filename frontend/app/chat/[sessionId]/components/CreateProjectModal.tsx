'use client';

import type { FormEvent } from 'react';
import { useEffect, useRef, useState } from 'react';
import { Loader2, X } from 'lucide-react';
import { Button } from '@/app/components/ui/button';
import { Input } from '@/app/components/ui/input';

interface CreateProjectModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (name: string) => Promise<void>;
  initialValue?: string;
  title?: string;
  description?: string;
  submitLabel?: string;
  submittingLabel?: string;
}

export function CreateProjectModal({
  isOpen,
  onClose,
  onSubmit,
  initialValue = '',
  title = '创建项目',
  description = '输入项目名称，用来归类当前会话。',
  submitLabel = '创建项目',
  submittingLabel = '创建中',
}: CreateProjectModalProps) {
  const [projectName, setProjectName] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!isOpen) {
      setProjectName('');
      setError('');
      setIsSubmitting(false);
      return;
    }

    setProjectName(initialValue);

    const timeoutId = window.setTimeout(() => {
      inputRef.current?.focus();
      inputRef.current?.select();
    }, 0);

    return () => window.clearTimeout(timeoutId);
  }, [initialValue, isOpen]);

  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isSubmitting) {
        onClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, isSubmitting, onClose]);

  if (!isOpen) return null;

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();

    const trimmedName = projectName.trim();
    if (!trimmedName) {
      setError('请输入项目名称');
      return;
    }

    setIsSubmitting(true);
    setError('');

    try {
      await onSubmit(trimmedName);
      setProjectName('');
      onClose();
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : `${submitLabel}失败，请稍后重试`);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4">
      <div
        className="absolute inset-0"
        onClick={() => {
          if (!isSubmitting) {
            onClose();
          }
        }}
      />

      <div className="relative w-full max-w-md rounded-2xl border border-zinc-200 bg-white text-zinc-900 shadow-xl dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
        <div className="px-5 pb-2 pt-5">
          <div className="flex items-start justify-between gap-4">
            <div>
              <h2 className="text-lg font-semibold tracking-tight">{title}</h2>
              <p className="mt-1 text-sm leading-6 text-zinc-500 dark:text-zinc-400">
                {description}
              </p>
            </div>

            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={onClose}
              disabled={isSubmitting}
              className="h-8 w-8 rounded-full text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
              aria-label="关闭创建项目弹框"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4 px-5 pb-5 pt-2">
          <div className="space-y-2">
            <label htmlFor="project-name" className="text-sm font-medium text-zinc-700 dark:text-zinc-200">
              项目名称
            </label>
            <Input
              id="project-name"
              ref={inputRef}
              value={projectName}
              onChange={(event) => {
                setProjectName(event.target.value);
                if (error) {
                  setError('');
                }
              }}
              placeholder="例如：内容策划、招聘流程、产品调研"
              maxLength={50}
              disabled={isSubmitting}
              className="h-11 rounded-xl border-zinc-300 bg-white text-zinc-900 placeholder:text-zinc-400 focus-visible:ring-2 focus-visible:ring-zinc-200 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100 dark:placeholder:text-zinc-500 dark:focus-visible:ring-zinc-700"
            />
            <div className="flex items-center justify-between text-xs text-zinc-500">
              <span>建议简短清晰，方便后续查找</span>
              <span>{projectName.trim().length}/50</span>
            </div>
          </div>

          {error && (
            <div className="rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600 dark:border-red-900/60 dark:bg-red-950/40 dark:text-red-300">
              {error}
            </div>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="ghost"
              onClick={onClose}
              disabled={isSubmitting}
              className="rounded-xl text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
            >
              取消
            </Button>
            <Button
              type="submit"
              disabled={isSubmitting || !projectName.trim()}
              className="min-w-[108px] rounded-xl bg-zinc-900 text-white hover:bg-zinc-800 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  {submittingLabel}
                </>
              ) : (
                submitLabel
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
