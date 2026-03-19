import { memo, type RefObject } from 'react';
import { Check, Copy, Pencil } from 'lucide-react';
import { Button } from '@/app/components/ui/button';
import { cn } from '@/app/lib/utils';
import { ChatSkeleton, AIMessageContent, UserMessage, UserAvatar } from './index';
import type { AgentInfo, Message } from '../types';

interface ChatMessagesPaneProps {
  agentInfo: AgentInfo;
  user: { name: string; picture?: string } | null;
  messages: Message[];
  processedMessages: Message[];
  latestUserMessageId?: string;
  latestUserMessageRef: RefObject<HTMLDivElement | null>;
  messagesEndRef: RefObject<HTMLDivElement | null>;
  isStreamingResponse: boolean;
  isLoadingOlderMessages: boolean;
  hasMoreMessages: boolean;
  totalMessageCount: number;
  isLoadingHistory: boolean;
  isLoading: boolean;
  editingMessageId: string | null;
  editingValue: string;
  isSavingEdit: boolean;
  copiedMessageId: string | null;
  onLoadOlderMessages: () => void;
  onRetry: () => void;
  onEditingValueChange: (value: string) => void;
  onStartEdit: (message: Message) => void;
  onSaveEdit: (message: Message) => void;
  onCancelEdit: () => void;
  onCopyUserMessage: (message: Message) => void;
}

export const ChatMessagesPane = memo(function ChatMessagesPane({
  agentInfo,
  user,
  messages,
  processedMessages,
  latestUserMessageId,
  latestUserMessageRef,
  messagesEndRef,
  isStreamingResponse,
  isLoadingOlderMessages,
  hasMoreMessages,
  totalMessageCount,
  isLoadingHistory,
  isLoading,
  editingMessageId,
  editingValue,
  isSavingEdit,
  copiedMessageId,
  onLoadOlderMessages,
  onRetry,
  onEditingValueChange,
  onStartEdit,
  onSaveEdit,
  onCancelEdit,
  onCopyUserMessage,
}: ChatMessagesPaneProps) {
  if (isLoadingHistory) {
    return (
      <div className="flex justify-center w-full">
        <ChatSkeleton />
      </div>
    );
  }

  if (messages.length === 0) {
    return (
      <div className="text-center py-12 sm:py-16 md:py-20 animate-fade-in px-4">
        <div className="flex justify-center mb-4 sm:mb-6">
          <div className="p-3 sm:p-4 bg-secondary rounded-xl sm:rounded-2xl">
            <span className="text-3xl sm:text-4xl">{agentInfo.icon}</span>
          </div>
        </div>
        <h2 className="text-2xl font-semibold mb-3 tracking-tight">{agentInfo.name}</h2>
        <p className="text-sm text-muted-foreground">{agentInfo.description}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6 sm:space-y-8 animate-fade-in">
      {isLoadingOlderMessages && (
        <div className="flex justify-center py-4">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
            <span>加载更早的消息...</span>
          </div>
        </div>
      )}

      {hasMoreMessages && !isLoadingOlderMessages && messages.length > 0 && (
        <div className="flex justify-center py-2">
          <button
            onClick={onLoadOlderMessages}
            className="text-sm text-muted-foreground hover:text-foreground transition-colors underline"
          >
            还有 {totalMessageCount - messages.length} 条更早的消息，向上滚动或点击加载
          </button>
        </div>
      )}

      {processedMessages.map((message) => (
        <div
          key={message.id}
          ref={message.role === 'user' && message.id === latestUserMessageId ? latestUserMessageRef : undefined}
          className={cn('flex w-full group/message', message.role === 'user' ? 'justify-end' : 'justify-start')}
        >
          {message.role === 'assistant' ? (
            <div className="w-full">
              <div className="relative text-sm w-full leading-relaxed" data-ai-message="">
                <AIMessageContent
                  content={message.content}
                  thought={message.thought}
                  isStreaming={message.isStreaming}
                  images={message.images}
                  isError={message.isError}
                  onRetry={onRetry}
                  toolCalls={message.toolCalls}
                />
              </div>
            </div>
          ) : (
            <div className="flex gap-2 sm:gap-3 md:gap-4 max-w-[95%] sm:max-w-[90%] md:max-w-[85%] flex-row-reverse">
              <UserAvatar user={user} />
              <div className="relative flex flex-col items-end">
                <div className="relative w-full min-w-0 max-w-full overflow-hidden rounded-2xl rounded-tr-sm bg-zinc-100 px-4 py-2.5 text-sm leading-relaxed dark:bg-zinc-800 sm:min-w-[180px] sm:w-fit">
                  {editingMessageId === message.id ? (
                    <div className="space-y-3">
                      <div className="relative">
                        <textarea
                          value={editingValue}
                          onChange={(event) => onEditingValueChange(event.target.value)}
                          onKeyDown={(event) => {
                            if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
                              event.preventDefault();
                              onSaveEdit(message);
                            } else if (event.key === 'Escape') {
                              event.preventDefault();
                              onCancelEdit();
                            }
                          }}
                          autoFocus
                          rows={Math.max(4, Math.min(10, (message.content?.match(/\n/g)?.length || 0) + 2))}
                          className="w-full min-h-[120px] resize-y rounded-lg border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800 px-3 py-3 text-sm leading-relaxed text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-500 dark:placeholder:text-zinc-400 outline-none"
                          placeholder="编辑消息..."
                          disabled={isSavingEdit}
                        />
                        {isSavingEdit && (
                          <div className="absolute inset-0 bg-zinc-50/80 dark:bg-zinc-800/80 rounded-lg flex items-center justify-center">
                            <div className="text-sm text-zinc-600 dark:text-zinc-400">保存中...</div>
                          </div>
                        )}
                      </div>
                    </div>
                  ) : (
                    <UserMessage
                      content={message.content}
                      images={message.images}
                      fileNames={message.fileNames}
                      files={message.files}
                    />
                  )}
                </div>

                <div
                  className={cn(
                    'mt-2 flex justify-end gap-2 transition-opacity duration-200',
                    editingMessageId === message.id
                      ? 'opacity-100'
                      : 'opacity-0 group-hover/message:opacity-100 group-focus-within/message:opacity-100'
                  )}
                >
                  {editingMessageId === message.id ? (
                    <>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => onSaveEdit(message)}
                        className="h-8 px-3 text-xs text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200 border-zinc-200 dark:border-zinc-700 hover:bg-zinc-50 dark:hover:bg-zinc-800"
                        disabled={isSavingEdit}
                      >
                        <Check className="h-3 w-3 mr-1" />
                        保存
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={onCancelEdit}
                        className="h-8 px-3 text-xs text-zinc-500 hover:text-zinc-700 dark:text-zinc-500 dark:hover:text-zinc-300 border-zinc-200 dark:border-zinc-700 hover:bg-zinc-50 dark:hover:bg-zinc-800"
                        disabled={isSavingEdit}
                      >
                        取消
                      </Button>
                    </>
                  ) : (
                    <>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => onStartEdit(message)}
                        className="h-7 w-7 p-0 text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"
                        title="编辑"
                      >
                        <Pencil className="h-3 w-3" />
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => onCopyUserMessage(message)}
                        className="h-7 w-7 p-0 text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"
                        title="复制"
                      >
                        {copiedMessageId === message.id ? (
                          <Check className="h-3 w-3 text-green-600" />
                        ) : (
                          <Copy className="h-3 w-3" />
                        )}
                      </Button>
                    </>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      ))}

      {isLoading && (messages.length === 0 || messages[messages.length - 1].role !== 'assistant') && (
        <div className="flex w-full justify-start">
          <div className="pt-2">
            <div className="flex items-center gap-2 text-sm text-muted-foreground animate-pulse">
              <div className="flex space-x-1">
                <div className="w-1.5 h-1.5 bg-muted-foreground/40 rounded-full animate-bounce [animation-delay:-0.3s]"></div>
                <div className="w-1.5 h-1.5 bg-muted-foreground/40 rounded-full animate-bounce [animation-delay:-0.15s]"></div>
                <div className="w-1.5 h-1.5 bg-muted-foreground/40 rounded-full animate-bounce"></div>
              </div>
              <span>AI 正在思考...</span>
            </div>
          </div>
        </div>
      )}

      <div
        ref={messagesEndRef}
        className={cn(
          'transition-all duration-300',
          messages.length > 0 && (messages[messages.length - 1].role === 'user' || isStreamingResponse)
            ? 'h-36 sm:h-40 md:h-44'
            : 'h-24 sm:h-28'
        )}
      />
    </div>
  );
});

ChatMessagesPane.displayName = 'ChatMessagesPane';
