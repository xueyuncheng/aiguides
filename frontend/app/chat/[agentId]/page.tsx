'use client';

import { useState, useEffect, useRef, memo } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import SessionSidebar, { Session } from '@/app/components/SessionSidebar';
import { Button } from '@/app/components/ui/button';
import { Textarea } from '@/app/components/ui/textarea';
import { Avatar, AvatarFallback, AvatarImage } from '@/app/components/ui/avatar';
import { ArrowUp, Code2, Eye, Copy, Check, X, ChevronDown, ChevronRight, Menu } from 'lucide-react';
import { cn } from '@/app/lib/utils';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  thought?: string;
  timestamp: Date;
  author?: string;
  isStreaming?: boolean;
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
const AIAvatar = memo(({ icon }: { icon: string }) => {
  return (
    <div className="h-8 w-8 rounded-full flex items-center justify-center flex-shrink-0 border border-border/50 bg-background">
      <span className="text-base">{icon}</span>
    </div>
  );
});

AIAvatar.displayName = 'AIAvatar';

// Helper component for User Avatar
const UserAvatar = memo(({ user }: { user: { name: string; picture?: string } | null }) => {
  if (!user) return null;

  return (
    <Avatar className="h-8 w-8 flex-shrink-0">
      <AvatarImage src={user.picture} alt={user.name} />
      <AvatarFallback className="bg-blue-500 text-white text-sm">
        {user.name.charAt(0).toUpperCase()}
      </AvatarFallback>
    </Avatar>
  );
});

UserAvatar.displayName = 'UserAvatar';

// Helper component for Code Block with syntax highlighting and copy button
const CodeBlock = memo(({ className, children }: { className?: string; children: React.ReactNode }) => {
  const match = /language-(\w+)/.exec(className || '');
  const [codeCopied, setCodeCopied] = useState(false);
  const codeString = String(children).replace(/\n$/, '');
  
  const handleCodeCopy = async () => {
    try {
      await navigator.clipboard.writeText(codeString);
      setCodeCopied(true);
      setTimeout(() => setCodeCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy code:', err);
    }
  };

  return (
    <div className="my-3 rounded-lg overflow-hidden border bg-zinc-950 dark:bg-zinc-900 text-white relative group">
      <div className="px-4 py-2 text-xs bg-zinc-800 text-zinc-400 border-b border-zinc-700 flex justify-between items-center">
        <span>{match?.[1] || 'code'}</span>
        <button
          onClick={handleCodeCopy}
          className="opacity-0 group-hover:opacity-100 transition-opacity duration-200 flex items-center gap-1.5 px-2 py-1 rounded hover:bg-zinc-700 text-zinc-300 hover:text-white"
          title={codeCopied ? "已复制" : "复制代码"}
          aria-label={codeCopied ? "已复制" : "复制代码"}
        >
          {codeCopied ? (
            <>
              <Check className="h-3.5 w-3.5" />
              <span className="text-xs">已复制</span>
            </>
          ) : (
            <>
              <Copy className="h-3.5 w-3.5" />
              <span className="text-xs">复制</span>
            </>
          )}
        </button>
      </div>
      <SyntaxHighlighter
        language={match?.[1] || 'text'}
        style={vscDarkPlus}
        customStyle={{
          margin: 0,
          padding: '1rem',
          background: 'transparent',
          fontSize: '0.75rem',
          lineHeight: '1.5',
        }}
        codeTagProps={{
          style: {
            fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
          }
        }}
      >
        {codeString}
      </SyntaxHighlighter>
    </div>
  );
});

CodeBlock.displayName = 'CodeBlock';

// Feedback timeout duration in milliseconds
const FEEDBACK_TIMEOUT_MS = 2000;

// Helper component for AI Message with raw markdown toggle
const AIMessageContent = memo(({ content, thought, isStreaming }: { content: string; thought?: string; isStreaming?: boolean }) => {
  const [showRaw, setShowRaw] = useState(false);
  const [isThoughtExpanded, setIsThoughtExpanded] = useState(false);

  // Handle auto-expand/collapse of thought process during streaming
  useEffect(() => {
    if (isStreaming) {
      if (thought && !content) {
        // Expand when thought is streaming but content hasn't started
        setIsThoughtExpanded(true);
      } else if (content) {
        // Collapse when main content starts appearing
        setIsThoughtExpanded(false);
      }
    }
  }, [isStreaming, !!thought, !!content]);

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
      {/* Thought Process section */}
      {thought && (
        <div className="mb-4">
          <button
            onClick={() => setIsThoughtExpanded(!isThoughtExpanded)}
            className="flex items-center gap-2 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors py-1 px-2 rounded-lg border bg-secondary/30"
            aria-expanded={isThoughtExpanded}
          >
            {isThoughtExpanded ? (
              <ChevronDown className="h-3 w-3" />
            ) : (
              <ChevronRight className="h-3 w-3" />
            )}
            <span>思考过程</span>
          </button>

          <div className={cn(
            "mt-2 overflow-hidden transition-all duration-300 ease-in-out",
            isThoughtExpanded ? "max-h-[2000px] opacity-100" : "max-h-0 opacity-0"
          )}>
            <div className="text-xs text-muted-foreground/80 leading-relaxed pl-4 border-l-2 border-muted py-1 italic whitespace-pre-wrap">
              {thought}
              {isStreaming && (
                <span className="inline-block w-1 h-3 ml-1 bg-muted-foreground/40 animate-pulse align-middle" />
              )}
            </div>
          </div>
        </div>
      )}

      {/* Content display */}
      <div className="relative">
        {!content && isStreaming && thought && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground py-2 animate-pulse">
            <div className="flex space-x-1">
              <div className="w-1.5 h-1.5 bg-muted-foreground/40 rounded-full animate-bounce [animation-delay:-0.3s]"></div>
              <div className="w-1.5 h-1.5 bg-muted-foreground/40 rounded-full animate-bounce [animation-delay:-0.15s]"></div>
              <div className="w-1.5 h-1.5 bg-muted-foreground/40 rounded-full animate-bounce"></div>
            </div>
            <span>准备回答...</span>
          </div>
        )}
        {showRaw ? (
          <pre className="whitespace-pre-wrap font-mono text-sm bg-secondary/50 p-4 rounded-lg border overflow-x-auto overflow-y-auto max-h-96">
            {content}
          </pre>
        ) : (
          <div className="prose prose-sm prose-neutral dark:prose-invert max-w-none prose-p:leading-relaxed prose-p:my-2 prose-pre:p-0 prose-pre:rounded-lg prose-headings:my-2">
            <ReactMarkdown
              remarkPlugins={[remarkGfm, remarkMath]}
              rehypePlugins={[rehypeKatex]}
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
                    <CodeBlock className={className}>
                      {children}
                    </CodeBlock>
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
        {!isStreaming && (
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
        )}
      </div>
    </div>
  );
});

AIMessageContent.displayName = 'AIMessageContent';

// Helper component for Chat Skeleton
const ChatSkeleton = memo(() => {
  return (
    <div className="w-full max-w-5xl px-6 py-10 space-y-12 animate-skeleton">
      {[1, 2, 3].map((i) => (
        <div key={i} className="flex flex-col space-y-8">
          {/* User message skeleton */}
          <div className="flex justify-end">
            <div className="flex gap-4 max-w-[85%] flex-row-reverse items-start">
              <div className="h-8 w-8 rounded-full bg-secondary shrink-0" />
              <div className="bg-secondary/50 h-10 w-48 rounded-2xl rounded-tr-sm" />
            </div>
          </div>
          {/* Assistant message skeleton */}
          <div className="flex justify-start">
            <div className="flex gap-4 max-w-[85%] items-start">
              <div className="h-8 w-8 rounded-full bg-secondary shrink-0" />
              <div className="space-y-3 pt-1">
                <div className="h-4 bg-secondary/50 w-[300px] md:w-[500px] rounded" />
                <div className="h-4 bg-secondary/50 w-[200px] md:w-[400px] rounded" />
                <div className="h-4 bg-secondary/50 w-[250px] md:w-[450px] rounded" />
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
});

ChatSkeleton.displayName = 'ChatSkeleton';

export default function ChatPage() {
  const params = useParams();
  const router = useRouter();
  const { user, loading, authenticatedFetch } = useAuth();
  const agentId = params.agentId as string;
  const agentInfo = agentInfoMap[agentId];

  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string>('');
  const [isLoadingHistory, setIsLoadingHistory] = useState(false);
  const [isLoadingOlderMessages, setIsLoadingOlderMessages] = useState(false);
  const [hasMoreMessages, setHasMoreMessages] = useState(false);
  const [totalMessageCount, setTotalMessageCount] = useState(0);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [isSessionsLoading, setIsSessionsLoading] = useState(false);
  const [shouldScrollInstantly, setShouldScrollInstantly] = useState(false);
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesStartRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const scrollResetTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previousScrollHeightRef = useRef<number>(0);
  const isAtBottomRef = useRef(true);
  const lastScrollTopRef = useRef(0);
  const scrollDirectionTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [isInputVisible, setIsInputVisible] = useState(true);
  // Maximum height for the textarea in pixels
  // Note: This value should match the max-h-[160px] in the Textarea className below
  const MAX_TEXTAREA_HEIGHT = 160;

  // Delay before resetting scroll behavior to smooth after loading history
  // This ensures messages are fully rendered before enabling smooth scroll again
  const SCROLL_RESET_DELAY = 100;

  // Number of messages to load per request
  const MESSAGES_PER_PAGE = 50;

  // Scroll threshold (in pixels) to trigger loading older messages
  const LOAD_MORE_THRESHOLD = 100;

  // Minimum scroll delta (in pixels) to detect scroll direction
  const MIN_SCROLL_THRESHOLD = 5;

  // Minimum scroll distance from top (in pixels) before hiding input
  const MIN_SCROLL_DISTANCE = 100;

  // Debounce delay (in milliseconds) for scroll direction detection
  const SCROLL_DEBOUNCE_DELAY = 50;

  const loadSessions = async (silent = false) => {
    if (!user?.user_id) return;

    try {
      if (!silent) setIsSessionsLoading(true);
      const response = await authenticatedFetch(`/api/${agentId}/sessions?user_id=${user.user_id}`);
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
    if (newSessionId === sessionId) return;

    setSessionId(newSessionId);
    // Clear messages immediately to show skeleton and avoid layout jumps
    setMessages([]);
    setHasMoreMessages(false);
    setTotalMessageCount(0);
    setIsLoadingHistory(true);
    setShouldScrollInstantly(true); // Enable instant scroll for history loading

    // Clear any pending timeout from previous session switch
    if (scrollResetTimeoutRef.current) {
      clearTimeout(scrollResetTimeoutRef.current);
    }

    try {
      // Load only the most recent messages (pagination)
      const response = await authenticatedFetch(`/api/${agentId}/sessions/${newSessionId}/history?user_id=${user?.user_id}&limit=${MESSAGES_PER_PAGE}&offset=0`);
      if (response.ok) {
        const data = await response.json();
        const historyMessages = data.messages.map((msg: any) => ({
          id: msg.id,
          role: msg.role,
          content: msg.content,
          thought: msg.thought,
          timestamp: new Date(msg.timestamp),
        }));
        setMessages(historyMessages);
        setHasMoreMessages(data.has_more || false);
        setTotalMessageCount(data.total || 0);
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

  const loadOlderMessages = async () => {
    if (isLoadingOlderMessages || !hasMoreMessages || !sessionId) return;

    setIsLoadingOlderMessages(true);

    // Store current scroll position before loading
    const container = scrollContainerRef.current;
    if (container) {
      previousScrollHeightRef.current = container.scrollHeight;
    }

    try {
      const currentOffset = messages.length;
      const response = await authenticatedFetch(`/api/${agentId}/sessions/${sessionId}/history?user_id=${user?.user_id}&limit=${MESSAGES_PER_PAGE}&offset=${currentOffset}`);
      if (response.ok) {
        const data = await response.json();
        const olderMessages = data.messages.map((msg: any) => ({
          id: msg.id,
          role: msg.role,
          content: msg.content,
          thought: msg.thought,
          timestamp: new Date(msg.timestamp),
        }));

        // Prepend older messages to the beginning
        setMessages(prev => [...olderMessages, ...prev]);
        setHasMoreMessages(data.has_more || false);

        // Restore scroll position after messages are added
        setTimeout(() => {
          const container = scrollContainerRef.current;
          if (container && previousScrollHeightRef.current) {
            const newScrollHeight = container.scrollHeight;
            const scrollDiff = newScrollHeight - previousScrollHeightRef.current;
            container.scrollTop = scrollDiff;
          }
        }, 0);
      }
    } catch (error) {
      console.error('Error loading older messages:', error);
    } finally {
      setIsLoadingOlderMessages(false);
    }
  };

  const handleNewSession = async () => {
    const newSessionId = `session-${Date.now()}-${Math.random().toString(36).substring(7)}`;
    setSessionId(newSessionId);
    setMessages([]);
    setHasMoreMessages(false);
    setTotalMessageCount(0);
  };

  const handleDeleteSession = async (sessionIdToDelete: string) => {
    try {
      const response = await authenticatedFetch(`/api/${agentId}/sessions/${sessionIdToDelete}?user_id=${user?.user_id}`, {
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

  const handleScroll = () => {
    if (scrollContainerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
      // Smart Scroll: check if at bottom
      const atBottom = scrollHeight - scrollTop - clientHeight < 10;
      isAtBottomRef.current = atBottom;

      // Pagination: check if scrolled to top to load more
      if (scrollTop < LOAD_MORE_THRESHOLD && hasMoreMessages && !isLoadingOlderMessages && !isLoadingHistory) {
        loadOlderMessages();
      }

      // Detect scroll direction for input box visibility
      // 向上滚动（查看历史消息）时隐藏输入框，向下滚动时显示
      const scrollDelta = scrollTop - lastScrollTopRef.current;

      // 如果在底部，始终立即显示输入框
      if (atBottom) {
        setIsInputVisible(true);
        lastScrollTopRef.current = scrollTop;
        return;
      }

      // 只在有明显滚动时才处理
      if (Math.abs(scrollDelta) > MIN_SCROLL_THRESHOLD && scrollTop > MIN_SCROLL_DISTANCE) {
        // Clear existing timeout
        if (scrollDirectionTimeoutRef.current) {
          clearTimeout(scrollDirectionTimeoutRef.current);
        }

        // 立即响应滚动方向
        if (scrollDelta < 0) {
          // 向上滚动 - 立即隐藏输入框
          setIsInputVisible(false);
        } else if (scrollDelta > 0) {
          // 向下滚动 - 短暂延迟后显示输入框（避免快速滚动时闪烁）
          scrollDirectionTimeoutRef.current = setTimeout(() => {
            setIsInputVisible(true);
          }, 100);
        }
      }

      // Update scroll position for next calculation
      lastScrollTopRef.current = scrollTop;
    }
  };

  useEffect(() => {
    // Scroll to bottom on EVERY new message added (especially user message)
    // Or if we are already at the bottom while streaming (to follow content)
    // Use behavior: 'auto' (instant) when loading history, 'smooth' for new messages
    const isNewUserMessage = messages.length > 0 && messages[messages.length - 1].role === 'user';

    if (isNewUserMessage || isAtBottomRef.current) {
      messagesEndRef.current?.scrollIntoView({
        behavior: shouldScrollInstantly ? 'auto' : 'smooth'
      });
    }

    // Show input when a new user message is added (not during streaming updates)
    if (isNewUserMessage) {
      setIsInputVisible(true);
    }
  }, [messages, shouldScrollInstantly]); // Only scroll when messages update

  // Cleanup scroll reset timeout on unmount
  useEffect(() => {
    return () => {
      if (scrollResetTimeoutRef.current) {
        clearTimeout(scrollResetTimeoutRef.current);
      }
      if (scrollDirectionTimeoutRef.current) {
        clearTimeout(scrollDirectionTimeoutRef.current);
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

    // Only poll for session title on the first message in this session
    // Check if this is the first user message (before adding the new one, we had 0 user messages)
    const isFirstMessage = messages.filter(m => m.role === 'user').length === 0;
    if (isFirstMessage) {
      let pollCount = 0;
      const maxPolls = 30; // Poll up to 30 times
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
      }, 1000); // Poll every 1 second
    }

    try {
      const response = await authenticatedFetch(`/api/${agentId}/chats/${sessionId}`, {
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

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const reader = response.body?.getReader();
      const decoder = new TextDecoder();
      let currentAuthor = '';
      let assistantContent = '';
      let assistantThought = '';

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
                  const isCompleteDuplicate = !data.is_thought && data.content === assistantContent;

                  if (!isCompleteDuplicate) {
                    // Check if author changed - create new message block
                    if (data.author && data.author !== currentAuthor) {
                      currentAuthor = data.author;
                      assistantContent = data.is_thought ? '' : data.content;
                      assistantThought = data.is_thought ? data.content : '';

                      // Create new message for new author
                      const newMessage: Message = {
                        id: `msg-${Date.now()}-${currentAuthor}`,
                        role: 'assistant',
                        content: assistantContent,
                        thought: assistantThought,
                        timestamp: new Date(),
                        author: currentAuthor,
                        isStreaming: true,
                      };
                      setMessages((prev) => [...prev, newMessage]);
                    } else {
                      // Same author - accumulate content or thought
                      if (data.is_thought) {
                        assistantThought += data.content;
                      } else {
                        assistantContent += data.content;
                      }

                      setMessages((prev) => {
                        const newMessages = [...prev];
                        const lastIndex = newMessages.length - 1;
                        if (lastIndex >= 0 && newMessages[lastIndex].role === 'assistant') {
                          newMessages[lastIndex] = {
                            ...newMessages[lastIndex],
                            content: assistantContent,
                            thought: assistantThought,
                            isStreaming: true,
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

        // Finalize streaming state for all messages in this session
        setMessages((prev) => prev.map(msg => ({ ...msg, isStreaming: false })));
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
        isMobileOpen={isMobileSidebarOpen}
        onMobileToggle={() => setIsMobileSidebarOpen(!isMobileSidebarOpen)}
      />

      {/* Main Content */}
      <div className="flex flex-col flex-1 h-full md:pl-[260px] relative transition-all duration-300">
        {/* Mobile Menu Button */}
        <div className="md:hidden fixed top-4 left-4 z-30">
          <Button
            onClick={() => setIsMobileSidebarOpen(true)}
            size="icon"
            variant="outline"
            className="h-10 w-10 rounded-full bg-background shadow-md"
            aria-label="打开菜单"
          >
            <Menu className="h-5 w-5" />
          </Button>
        </div>

        {/* Messages Area */}
        <div
          ref={scrollContainerRef}
          className="flex-1 overflow-y-auto no-scrollbar"
          onScroll={handleScroll}
        >
          <div className="flex flex-col items-center">
            <div className="w-full max-w-5xl px-4 sm:px-6 py-10 space-y-8">
              {/* Loading older messages indicator */}
              {isLoadingOlderMessages && (
                <div className="flex justify-center py-4">
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
                    <span>加载更早的消息...</span>
                  </div>
                </div>
              )}

              {/* Show info about available older messages */}
              {hasMoreMessages && !isLoadingOlderMessages && messages.length > 0 && (
                <div className="flex justify-center py-2">
                  <button
                    onClick={loadOlderMessages}
                    className="text-sm text-muted-foreground hover:text-foreground transition-colors underline"
                  >
                    还有 {totalMessageCount - messages.length} 条更早的消息，向上滚动或点击加载
                  </button>
                </div>
              )}

              {/* Hidden ref for tracking start of messages */}
              <div ref={messagesStartRef} />

              {isLoadingHistory ? (
                <div className="flex justify-center w-full">
                  <ChatSkeleton />
                </div>
              ) : messages.length === 0 ? (
                <div className="text-center py-20 animate-fade-in">
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
                <div className="space-y-8 animate-fade-in">
                  {messages.map((message) => (
                    <div
                      key={message.id}
                      className={cn(
                        "flex w-full",
                        message.role === 'user' ? "justify-end" : "justify-start"
                      )}
                    >
                      <div className={cn(
                        "flex gap-2 sm:gap-4 max-w-[95%] sm:max-w-[85%]",
                        message.role === 'user' ? "flex-row-reverse" : "flex-row"
                      )}>
                        {message.role === 'assistant' ? (
                          <AIAvatar icon={agentInfo.icon} />
                        ) : (
                          <UserAvatar user={user} />
                        )}

                        <div className={cn(
                          "relative text-sm w-full",
                          message.role === 'user'
                            ? "bg-secondary px-3 sm:px-5 py-2.5 sm:py-3 rounded-2xl rounded-tr-sm self-end max-w-[95%] sm:max-w-[85%]"
                            : "leading-6 pt-1 flex-1"
                        )}>
                          {message.role === 'assistant' ? (
                            <AIMessageContent
                              content={message.content}
                              thought={message.thought}
                              isStreaming={message.isStreaming}
                            />
                          ) : (
                            <div className="whitespace-pre-wrap">{message.content}</div>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                  {isLoading && (messages.length === 0 || messages[messages.length - 1].role !== 'assistant') && (
                    <div className="flex w-full justify-start">
                      <div className="flex gap-4 max-w-[85%]">
                        <AIAvatar icon={agentInfo.icon} />
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
                    </div>
                  )}
                  <div ref={messagesEndRef} className="h-24" />
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Input Area */}
        <div className={cn(
          "absolute bottom-0 left-0 w-full md:pl-[260px] bg-gradient-to-t from-background via-background to-transparent pt-6 pb-4 transition-transform duration-300 ease-in-out",
          !isInputVisible && "translate-y-full"
        )}>
          <div className="max-w-4xl mx-auto px-4 sm:px-6">
            <div className="relative flex items-center w-full bg-secondary/50 rounded-2xl border border-input shadow-sm focus-within:ring-1 focus-within:ring-ring focus-within:border-transparent transition-all">
              <form onSubmit={handleSubmit} className="w-full flex items-end p-1.5 gap-1.5">
                <Textarea
                  ref={textareaRef}
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                  onKeyDown={handleKeyDown}
                  onFocus={() => setIsInputVisible(true)}
                  placeholder={isLoadingHistory ? "正在加载历史记录..." : `给 ${agentInfo.name} 发送消息`}
                  className="flex-1 min-h-[36px] max-h-[160px] border-0 bg-transparent shadow-none focus-visible:ring-0 px-3 py-2 text-sm overflow-y-auto"
                  disabled={isLoading || isLoadingHistory}
                  autoComplete="off"
                  rows={1}
                />
                {isLoading ? (
                  <Button
                    type="button"
                    size="icon"
                    onClick={handleCancelMessage}
                    className="h-7 w-7 mb-0.5 rounded-full transition-all duration-200 bg-gradient-to-br from-orange-500 to-red-500 text-white hover:from-orange-600 hover:to-red-600 shadow-md hover:shadow-lg"
                    title="取消"
                  >
                    <X className="h-3.5 w-3.5 stroke-[2.5]" />
                  </Button>
                ) : (
                  <Button
                    type="submit"
                    size="icon"
                    disabled={!inputValue.trim()}
                    className={cn(
                      "h-7 w-7 mb-0.5 rounded-full transition-all duration-200",
                      inputValue.trim() ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
                    )}
                  >
                    <ArrowUp className="h-3.5 w-3.5" />
                  </Button>
                )}
              </form>
            </div>
            <div className="text-center text-xs text-muted-foreground mt-2">
              AI 可能会生成不准确的信息，请核查重要事实。
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
