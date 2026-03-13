'use client';

import type { FormEvent } from 'react';
import { useEffect } from 'react';
import { Plus, X } from 'lucide-react';
import { Button } from '@/app/components/ui/button';
import { Textarea } from '@/app/components/ui/textarea';
import { cn } from '@/app/lib/utils';

type MemoryType = 'fact' | 'preference' | 'context';

interface MemoryFormData {
  memory_type: MemoryType;
  content: string;
  importance: number;
}

interface MemoryEditorModalProps {
  isOpen: boolean;
  editingMemoryId: number | null;
  formData: MemoryFormData;
  isSubmitting: boolean;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onChange: (next: MemoryFormData) => void;
}

const filters: Array<{ value: MemoryType; label: string }> = [
  { value: 'fact', label: '事实' },
  { value: 'preference', label: '偏好' },
  { value: 'context', label: '上下文' },
];

export function MemoryEditorModal({
  isOpen,
  editingMemoryId,
  formData,
  isSubmitting,
  onClose,
  onSubmit,
  onChange,
}: MemoryEditorModalProps) {
  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isSubmitting) {
        onClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, isSubmitting, onClose]);

  if (!isOpen) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4">
      <div
        className="absolute inset-0"
        onClick={() => {
          if (!isSubmitting) {
            onClose();
          }
        }}
      />

      <div className="relative w-full max-w-xl rounded-2xl border border-zinc-800 bg-zinc-900 text-zinc-100 shadow-2xl">
        <div className="flex items-start justify-between gap-4 border-b border-zinc-800 px-5 py-4">
          <div>
            <h2 className="text-lg font-semibold text-white">{editingMemoryId ? '编辑记忆' : '新增记忆'}</h2>
            <p className="mt-1 text-sm text-zinc-400">保留会长期影响回答质量的信息，尽量简洁明确。</p>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={onClose}
            disabled={isSubmitting}
            className="h-8 w-8 rounded-full text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
            aria-label="关闭记忆编辑弹窗"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        <form onSubmit={onSubmit} className="space-y-5 px-5 py-5">
          <div className="grid gap-2">
            <div className="text-sm font-medium text-zinc-200">记忆类型</div>
            <div className="grid grid-cols-3 gap-2">
              {filters.map((filter) => {
                const active = formData.memory_type === filter.value;
                return (
                  <button
                    key={filter.value}
                    type="button"
                    onClick={() => onChange({ ...formData, memory_type: filter.value })}
                    className={cn(
                      'rounded-lg border px-3 py-2 text-sm transition',
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
          </div>

          <div className="grid gap-2">
            <label htmlFor="memory-content" className="text-sm font-medium text-zinc-200">记忆内容</label>
            <Textarea
              id="memory-content"
              value={formData.content}
              onChange={(event) => onChange({ ...formData, content: event.target.value })}
              placeholder="例如：我更喜欢回答直接一点，代码示例尽量用 Go。"
              maxLength={2000}
              className="min-h-[180px] rounded-xl border border-zinc-800 bg-zinc-950 px-3 py-2 text-zinc-100 placeholder:text-zinc-500"
            />
            <div className="text-right text-xs text-zinc-500">{formData.content.length}/2000</div>
          </div>

          <div className="grid gap-2">
            <label htmlFor="memory-importance" className="text-sm font-medium text-zinc-200">重要度 {formData.importance}/10</label>
            <input
              id="memory-importance"
              type="range"
              min={1}
              max={10}
              value={formData.importance}
              onChange={(event) => onChange({ ...formData, importance: Number(event.target.value) })}
              className="accent-zinc-100"
            />
          </div>

          <div className="flex justify-end gap-2 border-t border-zinc-800 pt-4">
            <Button
              type="button"
              variant="ghost"
              onClick={onClose}
              disabled={isSubmitting}
              className="text-zinc-300 hover:bg-zinc-800 hover:text-zinc-100"
            >
              取消
            </Button>
            <Button type="submit" disabled={isSubmitting} className="bg-zinc-100 text-zinc-950 hover:bg-zinc-200 disabled:opacity-60">
              <Plus className="mr-2 h-4 w-4" />
              {isSubmitting ? '保存中...' : editingMemoryId ? '保存修改' : '添加记忆'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
