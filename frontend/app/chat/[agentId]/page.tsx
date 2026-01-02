'use client';

import { useState, useEffect, useRef } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import SessionSidebar, { Session } from '@/app/components/SessionSidebar';
import { Button } from '@/app/components/ui/button';
import { Textarea } from '@/app/components/ui/textarea';
import { Avatar, AvatarFallback, AvatarImage } from '@/app/components/ui/avatar';
import { ArrowUp, Code2, Eye, Copy, Check, X } from 'lucide-react';
import { cn } from '@/app/lib/utils';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  author?: string;
}

interface AgentInfo {
  id: string;
  name: string;
  description: string;
  icon: string;
  color: string;
  examples: string[];
}

const agentInfoMap: Record<string, AgentInfo> = {
  assistant: {
    id: 'assistant',
    name: 'AI Assistant',
    description: '信息检索和事实核查',
    icon: '🔍',
    color: 'bg-blue-500',
    examples: [
      '什么是量子计算？',
      '请帮我查找关于人工智能的最新研究',
      '核查一下这个新闻是否准确...',
    ],
  },
  web_summary: {
    id: 'web_summary',
    name: 'WebSummary Agent',
    description: '网页内容分析',
    icon: '🌐',
    color: 'bg-green-500',
    examples: [
      '请帮我总结这个网页：https://example.com',
      '分析这篇文章的主要内容',
      '提取网页的关键信息',
    ],
  },
  email_summary: {
    id: 'email_summary',
    name: 'EmailSummary Agent',
    description: '邮件智能总结',
    icon: '📧',
    color: 'bg-purple-500',
    examples: [
      '请帮我总结收件箱中的重要邮件',
      '获取最近20封邮件并总结',
      '分析哪些邮件需要优先处理',
    ],
  },
  travel: {
    id: 'travel',
    name: 'Travel Agent',
    description: '旅游规划助手',
    icon: '✈️',
    color: 'bg-orange-500',
    examples: [
      '我计划去日本东京旅游5天，请帮我制定详细的旅游计划',
      '想在泰国曼谷玩3天，预算有限，请推荐经济实惠的行程',
      '帮我规划一个巴黎7日游，我对艺术和美食特别感兴趣',
    ],
  },
};

// Helper component for AI Avatar
const AIAvatar = ({ icon }: { icon: string }) => {
  return (
    <div className="h-8 w-8 rounded-full flex items-center justify-center flex-shrink-0 border border-border/50 bg-background">
      <span className="text-base">{icon}</span>
    </div>
  );
};

// Helper component for User Avatar
const UserAvatar = ({ user }: { user: { name: string; picture?: string } | null }) => {
  if (!user) return null;

  return (
    <Avatar className="h-8 w-8 flex-shrink-0">
      <AvatarImage src={user.picture} alt={user.name} />
      <AvatarFallback className="bg-blue-500 text-white text-sm">
        {user.name.charAt(0).toUpperCase()}
      </AvatarFallback>
    </Avatar>
  );
};

// Feedback timeout duration in milliseconds
const FEEDBACK_TIMEOUT_MS = 2000;

// Helper component for AI Message with raw markdown toggle
const AIMessageContent = ({ content }: { content: string }) => {
  const [showRaw, setShowRaw] = useState(false);
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState(false);
  const copyTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const errorTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Cleanup timeouts on unmount
  useEffect(() => {
    return () => {
      if (copyTimeoutRef.current) {
        clearTimeout(copyTimeoutRef.current);
      }
      if (errorTimeoutRef.current) {
        clearTimeout(errorTimeoutRef.current);
      }
    };
  }, []);

  const handleCopy = async () => {
    // Clear any existing timeouts
    if (copyTimeoutRef.current) {
      clearTimeout(copyTimeoutRef.current);
    }
    if (errorTimeoutRef.current) {
      clearTimeout(errorTimeoutRef.current);
    }

    // Check if clipboard API is available
    if (!navigator.clipboard) {
      console.error('Clipboard API not available');
      setCopyError(true);
      errorTimeoutRef.current = setTimeout(() => setCopyError(false), FEEDBACK_TIMEOUT_MS);
      return;
    }

    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setCopyError(false);
      copyTimeoutRef.current = setTimeout(() => setCopied(false), FEEDBACK_TIMEOUT_MS);
    } catch (err) {
      console.error('Failed to copy:', err);
      setCopyError(true);
      errorTimeoutRef.current = setTimeout(() => setCopyError(false), FEEDBACK_TIMEOUT_MS);
    }
  };

  return (
    <div className="group">
      {/* Content display */}
      {showRaw ? (
        <pre className="whitespace-pre-wrap font-mono text-sm bg-secondary/50 p-4 rounded-lg border overflow-x-auto overflow-y-auto max-h-96">
          {content}
        </pre>
      ) : (
        <div className="prose prose-sm prose-neutral dark:prose-invert max-w-none prose-p:leading-relaxed prose-p:my-2 prose-pre:p-0 prose-pre:rounded-lg prose-headings:my-2">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            components={{
              a: ({ ...props }) => (
                <a {...props} target="_blank" rel="noopener noreferrer" className="text-blue-500 hover:underline" />
              ),
              code: ({ className, children, ...props }) => {
                const match = /language-(\w+)/.exec(className || '')
                const isInline = !match;
                return isInline ? (
                  <code className="bg-secondary px-1.5 py-0.5 rounded text-xs font-mono" {...props}>
                    {children}
                  </code>
                ) : (
                  <div className="my-3 rounded-lg overflow-hidden border bg-zinc-950 dark:bg-zinc-900 text-white">
                    <div className="px-4 py-2 text-xs bg-zinc-800 text-zinc-400 border-b border-zinc-700 flex justify-between">
                      <span>{match?.[1]}</span>
                    </div>
                    <pre className="p-4 overflow-x-auto text-xs">
                      <code className={className} {...props}>
                        {children}
                      </code>
                    </pre>
                  </div>
                )
              },
              ul: ({ ...props }) => (
                <ul {...props} className="list-disc list-inside space-y-0.5 my-3 text-sm" />
              ),
              ol: ({ ...props }) => (
                <ol {...props} className="list-decimal list-inside space-y-0.5 my-3 text-sm" />
              ),
            }}
          >
            {content}
          </ReactMarkdown>
        </div>
      )}

      {/* Toggle and Copy buttons - below content with icons only */}
      <div className="flex gap-1 mt-2 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity duration-200">
        <Button
          size="sm"
          variant="ghost"
          onClick={() => setShowRaw(!showRaw)}
          className="h-6 w-6 p-0 bg-background/80 backdrop-blur-sm border hover:bg-background"
          title={showRaw ? "显示渲染效果" : "显示原始内容"}
          aria-label={showRaw ? "显示渲染效果" : "显示原始内容"}
        >
          {showRaw ? (
            <Eye className="h-3.5 w-3.5" />
          ) : (
            <Code2 className="h-3.5 w-3.5" />
          )}
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onClick={handleCopy}
          className="h-6 w-6 p-0 bg-background/80 backdrop-blur-sm border hover:bg-background"
          title={copyError ? "复制失败" : (copied ? "已复制" : "复制原始内容")}
          aria-label={copyError ? "复制失败" : (copied ? "已复制" : "复制原始内容")}
        >
          {copyError ? (
            <X className="h-3.5 w-3.5 text-red-500" aria-hidden="true" />
          ) : copied ? (
            <Check className="h-3.5 w-3.5 text-green-600" />
          ) : (
            <Copy className="h-3.5 w-3.5" />
          )}
        </Button>
      </div>
    </div>
  );
};

export default function ChatPage() {
  const params = useParams();
  const router = useRouter();
  const { user, loading } = useAuth();
  const agentId = params.agentId as string;
  const agentInfo = agentInfoMap[agentId];

  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string>('');
  const [isLoadingHistory, setIsLoadingHistory] = useState(false);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [isSessionsLoading, setIsSessionsLoading] = useState(false);
  const [isHovering, setIsHovering] = useState(false);
  const [shouldScrollInstantly, setShouldScrollInstantly] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const scrollResetTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  
  // Maximum height for the textarea in pixels
  // Note: This value should match the max-h-[200px] in the Textarea className below
  const MAX_TEXTAREA_HEIGHT = 200;

  // Delay before resetting scroll behavior to smooth after loading history
  // This ensures messages are fully rendered before enabling smooth scroll again
  const SCROLL_RESET_DELAY = 100;

  const loadSessions = async (silent = false) => {
    if (!user?.user_id) return;

    try {
      if (!silent) setIsSessionsLoading(true);
      const response = await fetch(`/api/${agentId}/sessions?user_id=${user.user_id}`);
      if (response.ok) {
        const data = await response.json();
        const sortedSessions = (data || []).sort((a: Session, b: Session) =>
          new Date(b.last_update_time).getTime() - new Date(a.last_update_time).getTime()
        );
        setSessions(sortedSessions);
        return sortedSessions;
      }
    } catch (error) {
      console.error('Error loading sessions:', error);
    } finally {
      if (!silent) setIsSessionsLoading(false);
    }
  };

  useEffect(() => {
    if (user?.user_id) {
      loadSessions();
    }
  }, [agentId, user?.user_id]);

  const handleSessionSelect = async (newSessionId: string) => {
    setSessionId(newSessionId);
    setMessages([]);
    setIsLoadingHistory(true);
    setShouldScrollInstantly(true); // Enable instant scroll for history loading

    // Clear any pending timeout from previous session switch
    if (scrollResetTimeoutRef.current) {
      clearTimeout(scrollResetTimeoutRef.current);
    }

    try {
      const response = await fetch(`/api/${agentId}/sessions/${newSessionId}/history?user_id=${user?.user_id}`);
      if (response.ok) {
        const data = await response.json();
        const historyMessages = data.messages.map((msg: any) => ({
          id: msg.id,
          role: msg.role,
          content: msg.content,
          timestamp: new Date(msg.timestamp),
        }));
        setMessages(historyMessages);
      }
    } catch (error) {
      console.error('Error loading history:', error);
    } finally {
      setIsLoadingHistory(false);
      // Reset to smooth scroll after a short delay to ensure history is rendered
      scrollResetTimeoutRef.current = setTimeout(() => {
        setShouldScrollInstantly(false);
        scrollResetTimeoutRef.current = null;
      }, SCROLL_RESET_DELAY);
    }
  };

  const handleNewSession = async () => {
    const newSessionId = `session-${Date.now()}-${Math.random().toString(36).substring(7)}`;
    setSessionId(newSessionId);
    setMessages([]);
  };

  const handleDeleteSession = async (sessionIdToDelete: string) => {
    try {
      const response = await fetch(`/api/${agentId}/sessions/${sessionIdToDelete}?user_id=${user?.user_id}`, {
        method: 'DELETE',
      });

      if (response.ok) {
        setSessions(prev => prev.filter(s => s.session_id !== sessionIdToDelete));
        if (sessionIdToDelete === sessionId) {
          handleNewSession();
        }
      }
    } catch (error) {
      console.error('Error deleting session:', error);
    }
  };

  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
      return;
    }

    if (!agentInfo) {
      router.push('/');
      return;
    }
    if (!sessionId) {
      handleNewSession();
    }
  }, [agentId, agentInfo, router, user]);

  useEffect(() => {
    if (!isHovering) {
      // Use instant scroll when loading history, smooth scroll for new messages
      messagesEndRef.current?.scrollIntoView({ 
        behavior: shouldScrollInstantly ? 'auto' : 'smooth' 
      });
    }
  }, [messages, shouldScrollInstantly]); // Depend on both messages and scroll behavior flag

  // Cleanup scroll reset timeout on unmount
  useEffect(() => {
    return () => {
      if (scrollResetTimeoutRef.current) {
        clearTimeout(scrollResetTimeoutRef.current);
      }
    };
  }, []);

  // Auto-resize textarea based on content
  useEffect(() => {
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = 'auto';
      textarea.style.height = `${Math.min(textarea.scrollHeight, MAX_TEXTAREA_HEIGHT)}px`;
    }
  }, [inputValue]);

  const handleCancelMessage = () => {
    // Abort the ongoing fetch request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    // Just set loading to false - keep the messages as they are
    setIsLoading(false);
  };

  const sendMessage = async (content: string) => {
    if (!content.trim() || isLoading) return;

    const userMessage: Message = {
      id: `msg-${Date.now()}`,
      role: 'user',
      content: content.trim(),
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInputValue('');
    setIsLoading(true);

    // Create a new AbortController for this request
    abortControllerRef.current = new AbortController();

    try {
      const response = await fetch(`/api/${agentId}/chats/${sessionId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          user_id: user?.user_id,
          session_id: sessionId,
          message: content.trim(),
        }),
        signal: abortControllerRef.current.signal,
      });

      if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);

      const reader = response.body?.getReader();
      const decoder = new TextDecoder();
      let currentAuthor = '';
      let assistantContent = '';

      if (reader) {
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          const chunk = decoder.decode(value, { stream: true });
          buffer += chunk;
          const lines = buffer.split('\n');
          buffer = lines.pop() || '';

          for (const line of lines) {
            const trimmedLine = line.trim();
            if (trimmedLine.startsWith('data:')) {
              try {
                const jsonStr = trimmedLine.substring(5).trim();
                if (!jsonStr) continue;

                const data = JSON.parse(jsonStr);

                if (data.content) {
                  // Check if this is a duplicate complete message (identical to accumulated content)
                  const isCompleteDuplicate = data.content === assistantContent;

                  if (!isCompleteDuplicate) {
                    // Check if author changed - create new message block
                    if (data.author && data.author !== currentAuthor) {
                      currentAuthor = data.author;
                      assistantContent = data.content;

                      // Create new message for new author
                      const newMessage: Message = {
                        id: `msg-${Date.now()}-${currentAuthor}`,
                        role: 'assistant',
                        content: assistantContent,
                        timestamp: new Date(),
                        author: currentAuthor,
                      };
                      setMessages((prev) => [...prev, newMessage]);
                    } else {
                      // Same author - accumulate content
                      assistantContent += data.content;
                      setMessages((prev) => {
                        const newMessages = [...prev];
                        const lastIndex = newMessages.length - 1;
                        if (lastIndex >= 0 && newMessages[lastIndex].role === 'assistant') {
                          newMessages[lastIndex] = {
                            ...newMessages[lastIndex],
                            content: assistantContent,
                          };
                        }
                        return newMessages;
                      });
                    }
                  }
                }
              } catch (e) {
                console.warn('JSON parse error:', e);
              }
            }
          }
        }
        // Poll for session updates to capture title generation
        let pollCount = 0;
        const maxPolls = 5;
        const pollInterval = setInterval(async () => {
          const fetchedSessions = await loadSessions(true);
          const currentSession = fetchedSessions?.find((s: Session) => s.session_id === sessionId);

          if (currentSession?.title) {
            clearInterval(pollInterval);
            return;
          }

          pollCount++;
          if (pollCount >= maxPolls) {
            clearInterval(pollInterval);
          }
        }, 2000); // Poll every 2 seconds
      }
    } catch (error) {
      // Check if the error is due to abort
      if (error instanceof Error && error.name === 'AbortError') {
        // Request was cancelled by user - this is expected, don't show error
        console.log('Request cancelled by user');
      } else {
        console.error('Error sending message:', error);
        const errorMessage: Message = {
          id: `msg-${Date.now()}-error`,
          role: 'assistant',
          content: '抱歉，发生了错误。请确保后端服务正在运行，并稍后重试。\n\n错误详情：' + (error instanceof Error ? error.message : String(error)),
          timestamp: new Date(),
        };
        setMessages((prev) => [...prev, errorMessage]);
      }
    } finally {
      setIsLoading(false);
      abortControllerRef.current = null;
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    sendMessage(inputValue);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Submit on Enter without Shift
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage(inputValue);
    }
    // Allow Shift+Enter for new line (default behavior)
  };

  const handleExampleClick = (example: string) => {
    if (isLoading) return;
    sendMessage(example);
  };

  if (loading || !agentInfo) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-background font-sans text-foreground">
      {/* Session Sidebar */}
      <SessionSidebar
        sessions={sessions}
        isLoading={isSessionsLoading}
        currentSessionId={sessionId}
        onSessionSelect={handleSessionSelect}
        onNewSession={handleNewSession}
        onDeleteSession={handleDeleteSession}
      />

      {/* Main Content */}
      <div className="flex flex-col flex-1 h-full pl-[260px] relative transition-all duration-300">
        {/* Messages Area */}
        <div
          className="flex-1 overflow-y-auto no-scrollbar"
          onMouseEnter={() => setIsHovering(true)}
          onMouseLeave={() => setIsHovering(false)}
        >
          {isLoadingHistory && (
            <div className="absolute inset-0 flex items-center justify-center bg-background/50 z-10">
              <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
            </div>
          )}

          <div className="flex flex-col items-center">
            <div className="w-full max-w-5xl px-6 py-10 space-y-8">
              {messages.length === 0 && !isLoadingHistory ? (
                <div className="text-center py-20">
                  <div className="flex justify-center mb-6">
                    <div className="p-4 bg-secondary rounded-2xl">
                      <span className="text-4xl">{agentInfo.icon}</span>
                    </div>
                  </div>
                  <h2 className="text-2xl font-semibold mb-8">
                    {agentInfo.name} 能够为您做什么？
                  </h2>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-2xl mx-auto">
                    {agentInfo.examples.map((example, index) => (
                      <button
                        key={index}
                        onClick={() => handleExampleClick(example)}
                        className="p-4 text-left border rounded-xl hover:bg-secondary/50 transition-colors text-sm text-balance"
                      >
                        {example}
                      </button>
                    ))}
                  </div>
                </div>
              ) : (
                <>
                  {messages.map((message) => (
                    <div
                      key={message.id}
                      className={cn(
                        "flex w-full",
                        message.role === 'user' ? "justify-end" : "justify-start"
                      )}
                    >
                      <div className={cn(
                        "flex gap-4 max-w-[85%]",
                        message.role === 'user' ? "flex-row-reverse" : "flex-row"
                      )}>
                        {message.role === 'assistant' ? (
                          <AIAvatar icon={agentInfo.icon} />
                        ) : (
                          <UserAvatar user={user} />
                        )}

                        <div className={cn(
                          "relative text-sm",
                          message.role === 'user'
                            ? "bg-secondary px-5 py-3 rounded-2xl rounded-tr-sm"
                            : "leading-6 pt-1"
                        )}>
                          {message.role === 'assistant' ? (
                            <AIMessageContent content={message.content} />
                          ) : (
                            <div className="whitespace-pre-wrap">{message.content}</div>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                  {isLoading && (
                    <div className="flex w-full justify-start">
                      <div className="flex gap-4 max-w-[85%]">
                        <AIAvatar icon={agentInfo.icon} />
                        <div className="pt-2">
                          <div className="flex space-x-1">
                            <div className="w-2 h-2 bg-muted-foreground/40 rounded-full animate-bounce [animation-delay:-0.3s]"></div>
                            <div className="w-2 h-2 bg-muted-foreground/40 rounded-full animate-bounce [animation-delay:-0.15s]"></div>
                            <div className="w-2 h-2 bg-muted-foreground/40 rounded-full animate-bounce"></div>
                          </div>
                        </div>
                      </div>
                    </div>
                  )}
                  <div ref={messagesEndRef} className="h-24" />
                </>
              )}
            </div>
          </div>
        </div>

        {/* Input Area */}
        <div className="absolute bottom-0 left-0 w-full pl-[260px] bg-gradient-to-t from-background via-background to-transparent pt-10 pb-6">
          <div className="max-w-5xl mx-auto px-6">
            <div className="relative flex items-center w-full bg-secondary/50 rounded-3xl border border-input shadow-sm focus-within:ring-1 focus-within:ring-ring focus-within:border-transparent transition-all">
              <form onSubmit={handleSubmit} className="w-full flex items-end p-2 gap-2">
                <Textarea
                  ref={textareaRef}
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder={`给 ${agentInfo.name} 发送消息`}
                  className="flex-1 min-h-[44px] max-h-[200px] border-0 bg-transparent shadow-none focus-visible:ring-0 px-4 py-3 text-base overflow-y-auto"
                  disabled={isLoading}
                  autoComplete="off"
                  rows={1}
                />
                {isLoading ? (
                  <Button
                    type="button"
                    size="icon"
                    onClick={handleCancelMessage}
                    className="h-8 w-8 mb-1 rounded-full transition-all duration-200 bg-gradient-to-br from-orange-500 to-red-500 text-white hover:from-orange-600 hover:to-red-600 shadow-md hover:shadow-lg"
                    title="取消"
                  >
                    <X className="h-4 w-4 stroke-[2.5]" />
                  </Button>
                ) : (
                  <Button
                    type="submit"
                    size="icon"
                    disabled={!inputValue.trim()}
                    className={cn(
                      "h-8 w-8 mb-1 rounded-full transition-all duration-200",
                      inputValue.trim() ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
                    )}
                  >
                    <ArrowUp className="h-4 w-4" />
                  </Button>
                )}
              </form>
            </div>
            <div className="text-center text-xs text-muted-foreground mt-3">
              AI 可能会生成不准确的信息，请核查重要事实。
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
