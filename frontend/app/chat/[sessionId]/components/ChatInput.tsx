import { forwardRef, useRef } from 'react';
import { Button } from '@/app/components/ui/button';
import { Textarea } from '@/app/components/ui/textarea';
import { ArrowUp, X, ImagePlus } from 'lucide-react';
import { cn } from '@/app/lib/utils';
import type { SelectedImage } from '../types';

interface ChatInputProps {
  inputValue: string;
  onInputChange: (value: string) => void;
  onKeyDown: (e: React.KeyboardEvent<HTMLTextAreaElement>) => void;
  onPaste: (e: React.ClipboardEvent<HTMLTextAreaElement>) => void;
  onSubmit: (e: React.FormEvent) => void;
  onCancel: () => void;
  onFocus: () => void;
  selectedImages: SelectedImage[];
  onRemoveImage: (imageId: string) => void;
  onImageSelect: (e: React.ChangeEvent<HTMLInputElement>) => void;
  imageError: string | null;
  isLoading: boolean;
  isLoadingHistory: boolean;
  canSend: boolean;
  agentName: string;
}

export const ChatInput = forwardRef<HTMLTextAreaElement, ChatInputProps>(({
  inputValue,
  onInputChange,
  onKeyDown,
  onPaste,
  onSubmit,
  onCancel,
  onFocus,
  selectedImages,
  onRemoveImage,
  onImageSelect,
  imageError,
  isLoading,
  isLoadingHistory,
  canSend,
  agentName,
}, textareaRef) => {
  const imageInputRef = useRef<HTMLInputElement>(null);

  return (
      <div className="absolute bottom-0 left-0 w-full md:pl-[260px] bg-gradient-to-t from-background via-background/95 to-transparent pt-3 sm:pt-4 pb-2 sm:pb-3">
      <div className="max-w-5xl mx-auto px-3 sm:px-4 md:px-6">
        <div className="relative flex flex-col w-full bg-white/95 dark:bg-zinc-950/70 backdrop-blur-xl rounded-2xl border border-zinc-200/80 dark:border-zinc-800/80 shadow-[0_4px_20px_rgba(15,23,42,0.06)] dark:shadow-[0_4px_20px_rgba(0,0,0,0.35)] transition-all duration-300 overflow-hidden">
          {selectedImages.length > 0 && (
            <div className="flex flex-wrap gap-2 px-3 pt-3">
              {selectedImages.map((image, index) => (
                <div key={image.id} className="relative group">
                  {image.isPdf ? (
                    <div className="h-16 w-16 rounded-lg border border-zinc-200 dark:border-zinc-700 bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center text-[10px] font-medium text-zinc-600 dark:text-zinc-300">
                      PDF
                    </div>
                  ) : (
                    <img
                      src={image.dataUrl}
                      alt={image.name || `已选图片 ${index + 1}`}
                      className="h-16 w-16 object-cover rounded-lg border border-zinc-200 dark:border-zinc-700"
                    />
                  )}
                  <button
                    type="button"
                    onClick={() => onRemoveImage(image.id)}
                    className="absolute -top-1.5 -right-1.5 h-5 w-5 rounded-full bg-zinc-900/80 text-white flex items-center justify-center shadow-sm opacity-0 group-hover:opacity-100 transition-opacity"
                    aria-label="移除图片"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              ))}
            </div>
          )}
          {imageError && (
            <div className="px-3 pt-2 text-xs text-red-500" role="alert" aria-live="polite">
              {imageError}
            </div>
          )}
          <form onSubmit={onSubmit} className="w-full flex items-center p-2 gap-2">
            <input
              ref={imageInputRef}
              type="file"
              accept="image/*,.pdf"
              multiple
              className="hidden"
              onChange={onImageSelect}
            />
            <Button
              type="button"
              size="icon"
              variant="ghost"
              onClick={() => imageInputRef.current?.click()}
              disabled={isLoading || isLoadingHistory}
              className="h-8 w-8 sm:h-7 sm:w-7 rounded-full text-muted-foreground hover:text-foreground"
              title="添加图片"
              aria-label="添加图片"
            >
              <ImagePlus className="h-4 w-4 sm:h-3.5 sm:w-3.5" />
            </Button>
            <Textarea
              ref={textareaRef}
              value={inputValue}
              onChange={(e) => onInputChange(e.target.value)}
              onKeyDown={onKeyDown}
              onPaste={onPaste}
              onFocus={onFocus}
              placeholder={isLoadingHistory ? "正在加载历史记录..." : `给 ${agentName} 发送消息`}
              className="chat-input-textarea flex-1 min-h-[44px] max-h-[160px] rounded-xl border-0 bg-transparent shadow-none focus-visible:ring-0 focus-visible:ring-offset-0 px-2.5 py-2 text-sm overflow-y-auto resize-none placeholder:text-zinc-400/70 dark:placeholder:text-zinc-500/60 no-scrollbar leading-normal transition-colors"
              disabled={isLoading || isLoadingHistory}
              autoComplete="off"
              rows={1}
            />
            {isLoading ? (
              <Button
                type="button"
                size="icon"
                onClick={onCancel}
                className="h-8 w-8 sm:h-7 sm:w-7 rounded-full transition-all duration-200 bg-gradient-to-br from-orange-500 to-red-500 text-white hover:from-orange-600 hover:to-red-600 shadow-md hover:shadow-lg tap-highlight-transparent min-h-[36px] min-w-[36px] sm:min-h-[28px] sm:min-w-[28px]"
                title="取消"
              >
                <X className="h-3.5 w-3.5 sm:h-3.5 sm:w-3.5 stroke-[2.5]" />
              </Button>
            ) : (
              <Button
                type="submit"
                size="icon"
                disabled={!canSend}
                className={cn(
                  "h-8 w-8 sm:h-7 sm:w-7 rounded-full transition-all duration-300 tap-highlight-transparent min-h-[36px] min-w-[36px] sm:min-h-[28px] sm:min-w-[28px] flex items-center justify-center",
                  canSend
                    ? "bg-zinc-900 dark:bg-zinc-100 text-zinc-100 dark:text-zinc-900 hover:bg-zinc-800 dark:hover:bg-zinc-200 hover:scale-105 active:scale-95 shadow-sm"
                    : "bg-zinc-100 dark:bg-zinc-800 text-zinc-400 dark:text-zinc-500 opacity-50"
                )}
              >
                <ArrowUp className="h-4 w-4 sm:h-3.5 sm:w-3.5 stroke-[2.5]" />
              </Button>
            )}
          </form>
        </div>
      </div>
    </div>
  );
});

ChatInput.displayName = 'ChatInput';
