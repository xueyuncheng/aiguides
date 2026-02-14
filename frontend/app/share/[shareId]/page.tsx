'use client';

import { useState, useEffect, useMemo } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Clock, AlertCircle, Lock } from 'lucide-react';
import { AIAvatar, AIMessageContent, UserMessage, ChatSkeleton } from '@/app/chat/[sessionId]/components';
import { agentInfoMap } from '@/app/chat/[sessionId]/constants';
import type { Message } from '@/app/chat/[sessionId]/types';
import { cn } from '@/app/lib/utils';

interface SharedConversationResponse {
  share_id: string;
  session_id: string;
  app_name: string;
  title?: string;
  messages: Message[];
  expires_at: string;
  is_expired: boolean;
}

export default function SharedConversationPage() {
  const params = useParams();
  const router = useRouter();
  const shareId = params.shareId as string;
  const [data, setData] = useState<SharedConversationResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchSharedConversation = async () => {
      try {
        const response = await fetch(`/api/share/${shareId}`);
        
        if (!response.ok) {
          if (response.status === 404) {
            setError('This shared conversation was not found.');
          } else if (response.status === 410) {
            const errorData = await response.json();
            setError(`This shared link has expired. It expired on ${new Date(errorData.expires_at).toLocaleDateString()}.`);
          } else {
            setError('Failed to load shared conversation.');
          }
          return;
        }

        const result = await response.json();
        setData(result);
      } catch (err) {
        console.error('Error fetching shared conversation:', err);
        setError('An error occurred while loading the conversation.');
      } finally {
        setIsLoading(false);
      }
    };

    if (shareId) {
      fetchSharedConversation();
    }
  }, [shareId]);

  const processedMessages = useMemo(() => {
    if (!data?.messages || data.messages.length === 0) return [];

    const result: Message[] = [];
    data.messages.forEach((msg) => {
      const last = result[result.length - 1];
      if (
        last &&
        last.role === 'assistant' &&
        msg.role === 'assistant' &&
        !last.isError &&
        !msg.isError
      ) {
        const merged = { ...last };
        merged.content = (merged.content || '') + (msg.content || '');
        if (msg.thought) {
          merged.thought = (merged.thought || '') + (merged.thought ? '\n\n' : '') + msg.thought;
        }
        if (msg.images && msg.images.length > 0) {
          merged.images = [...(merged.images || []), ...(msg.images || [])];
        }
        result[result.length - 1] = merged;
      } else {
        result.push(msg);
      }
    });
    return result;
  }, [data?.messages]);
  const conversationTitle = data?.title?.trim() || 'Shared Conversation';

  useEffect(() => {
    if (typeof document === 'undefined') return;

    if (isLoading) {
      document.title = 'Loading shared conversation - AIGuides';
      return;
    }

    if (error || !data) {
      document.title = 'Shared Conversation - AIGuides';
      return;
    }

    document.title = conversationTitle;
  }, [isLoading, error, data, conversationTitle]);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background">
        <div className="flex flex-col h-screen">
          <header className="border-b border-border">
            <div className="max-w-5xl mx-auto px-3 sm:px-4 md:px-6 py-3">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-xl bg-secondary animate-pulse" />
                <div className="space-y-1">
                  <div className="h-4 w-40 rounded bg-secondary/70 animate-pulse" />
                  <div className="h-3 w-24 rounded bg-secondary/50 animate-pulse" />
                </div>
              </div>
            </div>
          </header>
          <div className="flex-1 overflow-y-auto no-scrollbar mobile-scroll">
            <div className="flex flex-col items-center">
              <ChatSkeleton />
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="min-h-screen bg-background">
        <div className="flex flex-col h-screen">
          <header className="border-b border-border">
            <div className="max-w-5xl mx-auto px-3 sm:px-4 md:px-6 py-3">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-xl bg-secondary/80 flex items-center justify-center">
                  <Lock className="h-5 w-5 text-muted-foreground" />
                </div>
                <div>
                  <h1 className="text-base font-semibold text-foreground">{conversationTitle}</h1>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <Lock className="w-3 h-3" />
                    <span>Read-only view</span>
                  </div>
                </div>
              </div>
            </div>
          </header>
          <div className="flex-1 overflow-y-auto no-scrollbar mobile-scroll">
            <div className="flex flex-col items-center">
              <div className="w-full max-w-5xl px-3 sm:px-4 md:px-6 py-10">
                <div className="rounded-xl border border-border bg-background p-6 sm:p-8 text-center">
                  <div className="w-12 h-12 bg-red-100 dark:bg-red-900/20 rounded-full flex items-center justify-center mx-auto mb-4">
                    <AlertCircle className="w-6 h-6 text-red-600 dark:text-red-400" />
                  </div>
                  <h1 className="text-lg font-semibold text-foreground mb-2">
                    Unable to Load Conversation
                  </h1>
                  <p className="text-sm text-muted-foreground mb-6">
                    {error || 'This shared conversation could not be found.'}
                  </p>
                  <button
                    onClick={() => router.push('/')}
                    className="inline-flex items-center justify-center rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/50 transition-colors"
                  >
                    Go to Home
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  const agentInfo = agentInfoMap[data.app_name] || agentInfoMap['assistant'];
  const expiresAt = new Date(data.expires_at);
  return (
    <div className="min-h-screen bg-background">
      <div className="flex flex-col h-screen">
        <header className="border-b border-border">
          <div className="max-w-5xl mx-auto px-3 sm:px-4 md:px-6 py-3">
            <div className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-xl bg-secondary/70 flex items-center justify-center">
                  <span className="text-xl">{agentInfo.icon}</span>
                </div>
                <div>
                  <h1 className="text-base font-semibold text-foreground">{conversationTitle}</h1>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <Lock className="w-3 h-3" />
                    <span>Read-only view</span>
                  </div>
                </div>
              </div>
              <button
                onClick={() => router.push('/')}
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                Visit AIGuides
              </button>
            </div>
          </div>
        </header>

        <div className="border-b border-amber-200/70 dark:border-amber-800/50 bg-amber-50/60 dark:bg-amber-900/10">
          <div className="max-w-5xl mx-auto px-3 sm:px-4 md:px-6 py-2.5">
            <div className="flex items-center gap-2 text-xs sm:text-sm text-amber-800 dark:text-amber-200">
              <Clock className="w-4 h-4" />
              <span>
                This shared link will expire on{' '}
                <strong>{expiresAt.toLocaleDateString()}</strong> at{' '}
                <strong>{expiresAt.toLocaleTimeString()}</strong>
              </span>
            </div>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto no-scrollbar mobile-scroll">
          <div className="flex flex-col items-center">
            <div className="w-full max-w-5xl px-3 sm:px-4 md:px-6 py-6 sm:py-8 md:py-10">
              {processedMessages.length === 0 ? (
                <div className="text-center py-12 sm:py-16 md:py-20 animate-fade-in px-4">
                  <div className="flex justify-center mb-4 sm:mb-6">
                    <div className="p-3 sm:p-4 bg-secondary rounded-xl sm:rounded-2xl">
                      <span className="text-3xl sm:text-4xl">{agentInfo.icon}</span>
                    </div>
                  </div>
                  <h2 className="text-xl sm:text-2xl font-semibold mb-2 tracking-tight">
                    No messages in this conversation
                  </h2>
                  <p className="text-sm text-muted-foreground">
                    This shared conversation does not have any messages yet.
                  </p>
                </div>
              ) : (
                <div className="space-y-6 sm:space-y-8 animate-fade-in">
                  {processedMessages.map((message, index) => (
                    <div
                      key={message.id || index}
                      className={cn(
                        "flex w-full",
                        message.role === 'user' ? "justify-end" : "justify-start"
                      )}
                    >
                      <div
                        className={cn(
                          "flex gap-2 sm:gap-3 md:gap-4 max-w-[95%] sm:max-w-[90%] md:max-w-[85%]",
                          message.role === 'user' ? "flex-row-reverse" : "flex-row"
                        )}
                      >
                        {message.role === 'assistant' ? (
                          <AIAvatar icon={agentInfo.icon} />
                        ) : null}

                        {message.role === 'assistant' ? (
                          <div className="relative text-sm w-full leading-relaxed pt-1 flex-1">
                            <AIMessageContent
                              content={message.content}
                              thought={message.thought}
                              images={message.images}
                              thoughtStorageKey={`share:${data.share_id}:thought:${message.id || index}`}
                            />
                          </div>
                        ) : (
                          <div className="relative flex flex-col items-end">
                            <div className="relative text-sm leading-relaxed bg-zinc-100 dark:bg-zinc-800 px-4 py-2.5 rounded-2xl rounded-tr-sm max-w-fit min-w-[180px]">
                              <UserMessage
                                content={message.content}
                                images={message.images}
                                fileNames={message.fileNames}
                              />
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}

              <div className="mt-8 text-center text-sm text-muted-foreground">
                <p>
                  This is a readonly view of a shared conversation from{' '}
                  <button
                    onClick={() => router.push('/')}
                    className="text-primary hover:underline"
                  >
                    AIGuides
                  </button>
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
