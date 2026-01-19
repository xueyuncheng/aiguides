'use client';

import { useState, useEffect, useRef, memo, useMemo } from 'react';
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
import { ArrowUp, Code2, Eye, Copy, Check, X, ChevronDown, ChevronRight, Menu, RotateCcw, ImagePlus } from 'lucide-react';
import { cn } from '@/app/lib/utils';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  thought?: string;
  timestamp: Date;
  author?: string;
  isStreaming?: boolean;
  images?: string[];
  isError?: boolean;
}

interface SelectedImage {
  id: string;
  dataUrl: string;
  name: string;
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
};

// Helper component for AI Avatar
const AIAvatar = memo(({ icon }: { icon: string }) => {
  return (
    <div className="h-8 w-8 rounded-full flex items-center justify-center flex-shrink-0 border border-border bg-background shadow-sm">
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
    <div className="my-6 rounded-lg overflow-hidden border bg-zinc-950 dark:bg-zinc-900 text-zinc-50 relative group shadow-sm">
      <div className="px-4 py-2 text-xs bg-zinc-900 border-b border-zinc-800 flex justify-between items-center">
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

const markdownRemarkPlugins = [remarkGfm, remarkMath];
const markdownRehypePlugins = [rehypeKatex];
const markdownComponents = {
  a: ({ ...props }) => (
    <a {...props} target="_blank" rel="noopener noreferrer" className="text-primary underline underline-offset-4 font-medium" />
  ),
  img: ({ src, ...props }) => {
    if (!src) return null;
    return <img src={src} {...props} className="max-w-full h-auto rounded-lg border my-6" loading="lazy" />;
  },
  code: ({ className, children, ...props }) => {
    const match = /language-(\w+)/.exec(className || '');
    const isInline = !match;
    return isInline ? (
      <code className="bg-muted px-1.5 py-0.5 rounded text-[13px] font-mono text-foreground" {...props}>
        {children}
      </code>
    ) : (
      <CodeBlock className={className}>
        {children}
      </CodeBlock>
    );
  },
  ul: ({ ...props }) => (
    <ul {...props} className="list-disc pl-6 space-y-1 my-4 text-sm" />
  ),
  ol: ({ ...props }) => (
    <ol {...props} className="list-decimal pl-6 space-y-1 my-4 text-sm" />
  ),
};

// Feedback timeout duration in milliseconds
const FEEDBACK_TIMEOUT_MS = 2000;

// Helper component for AI Message with raw markdown toggle
const AIMessageContent = memo(({
  content,
  thought,
  isStreaming,
  images,
  isError,
  onRetry
}: {
  content: string;
  thought?: string;
  isStreaming?: boolean;
  images?: string[];
  isError?: boolean;
  onRetry?: () => void;
}) => {
  const [showRaw, setShowRaw] = useState(false);
  const [isThoughtExpanded, setIsThoughtExpanded] = useState(false);

  // Keep thought process collapsed by default, even during streaming
  // Users can manually expand it if they want to see the thinking process

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
            className="flex items-center gap-2 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors py-1.5 px-3 rounded-md border bg-muted/50"
            aria-expanded={isThoughtExpanded}
          >
            {isThoughtExpanded ? (
              <ChevronDown className="h-3 w-3" />
            ) : (
              <ChevronRight className="h-3 w-3" />
            )}
            <span>{isStreaming && !content ? '思考中' : '查看思考过程'}</span>
            {isStreaming && !content && (
              <div className="flex space-x-0.5 ml-1">
                <div className="w-1 h-1 bg-primary rounded-full animate-bounce [animation-delay:-0.3s]"></div>
                <div className="w-1 h-1 bg-primary rounded-full animate-bounce [animation-delay:-0.15s]"></div>
                <div className="w-1 h-1 bg-primary rounded-full animate-bounce"></div>
              </div>
            )}
          </button>

          <div className={cn(
            "mt-3 overflow-hidden transition-all duration-300 ease-in-out",
            isThoughtExpanded ? "max-h-[2000px] opacity-100" : "max-h-0 opacity-0"
          )}>
            <div className="text-xs text-muted-foreground/90 leading-relaxed pl-4 border-l-2 border-primary/20 py-1 whitespace-pre-wrap bg-muted/20 rounded-r-md">
              {thought}
              {isStreaming && (
                <span className="inline-block w-1 h-3 ml-1 bg-primary animate-pulse align-middle" />
              )}
            </div>
          </div>
        </div>
      )}

      {/* Content display */}
      <div className="relative">
        {/* Display images if present */}
        {images && images.length > 0 && (
          <div className="mb-4 space-y-3">
            {images.map((imageData, index) => (
              <img
                key={index}
                src={imageData}
                alt={`Generated image ${index + 1}`}
                className="max-w-full h-auto rounded-lg border shadow-sm"
                loading="lazy"
              />
            ))}
          </div>
        )}
        {!content && isStreaming && !thought && (
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
          <pre className="whitespace-pre-wrap font-mono text-sm bg-muted p-4 rounded-lg border overflow-x-auto overflow-y-auto max-h-96">
            {content}
          </pre>
        ) : (
          <div className="prose prose-sm prose-zinc dark:prose-invert max-w-none prose-p:leading-relaxed prose-p:my-3 prose-pre:p-0 prose-pre:rounded-lg prose-headings:my-4 prose-headings:font-semibold">
            <ReactMarkdown
              remarkPlugins={markdownRemarkPlugins}
              rehypePlugins={markdownRehypePlugins}
              components={markdownComponents}
            >
              {content}
            </ReactMarkdown>
          </div>
        )}

        {/* Action buttons - below content */}
        {!isStreaming && (
          <div className={cn(
            "flex gap-1 mt-2 transition-opacity duration-200",
            isError ? "opacity-100" : "opacity-0 group-hover:opacity-100 group-focus-within:opacity-100"
          )}>
            {isError ? (
              onRetry && (
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={onRetry}
                  className="h-6 w-6 p-0 hover:bg-secondary text-muted-foreground hover:text-foreground"
                  title="重试"
                  aria-label="重试"
                >
                  <RotateCcw className="h-3.5 w-3.5" />
                </Button>
              )
            ) : (
              <>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => setShowRaw(!showRaw)}
                  className="h-6 w-6 p-0 hover:bg-secondary text-muted-foreground hover:text-foreground"
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
                  className="h-6 w-6 p-0 hover:bg-secondary text-muted-foreground hover:text-foreground"
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
              </>
            )}
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
  const urlSessionId = params.sessionId as string;
  // 固定使用 assistant agent
  const agentId = 'assistant';
  const agentInfo = agentInfoMap[agentId];

  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [selectedImages, setSelectedImages] = useState<SelectedImage[]>([]);
  const [imageError, setImageError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string>(urlSessionId || '');
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
  const imageInputRef = useRef<HTMLInputElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const titlePollIntervalRef = useRef<NodeJS.Timeout | null>(null);
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

  // Keep these limits aligned with backend validation in assistant/sse.go.
  const MAX_IMAGE_COUNT = 4;
  const MAX_IMAGE_SIZE_BYTES = 5 * 1024 * 1024;
  const MAX_IMAGE_SIZE_MB = Math.round(MAX_IMAGE_SIZE_BYTES / (1024 * 1024));
  const IMAGE_COUNT_ERROR = `最多只能上传 ${MAX_IMAGE_COUNT} 张图片`;
  const IMAGE_SIZE_ERROR = `图片大小不能超过 ${MAX_IMAGE_SIZE_MB}MB`;
  const IMAGE_TYPE_ERROR = '仅支持图片文件';
  const IMAGE_READ_ERROR = '读取图片失败';

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
  }, [user?.user_id]);

  const loadSessionHistory = async (targetSessionId: string, updateUrl: boolean = true) => {
    // Update URL if needed (for session switching)
    if (updateUrl && targetSessionId !== sessionId) {
      window.history.pushState(null, '', `/chat/${targetSessionId}`);
      setSessionId(targetSessionId);
    }

    // Clear messages immediately to show skeleton and avoid layout jumps
    setMessages([]);
    setHasMoreMessages(false);
    setTotalMessageCount(0);
    setIsLoadingHistory(true);
    setShouldScrollInstantly(true); // Enable instant scroll for history loading
    setIsInputVisible(true); // Always show input when switching sessions
    setSelectedImages([]);
    setImageError(null);

    // Clear any pending timeout from previous session switch
    if (scrollResetTimeoutRef.current) {
      clearTimeout(scrollResetTimeoutRef.current);
    }

    try {
      // Load only the most recent messages (pagination)
      const response = await authenticatedFetch(`/api/${agentId}/sessions/${targetSessionId}/history?user_id=${user?.user_id}&limit=${MESSAGES_PER_PAGE}&offset=0`);
      if (response.ok) {
        const data = await response.json();
        const historyMessages = data.messages.map((msg: any) => ({
          id: msg.id,
          role: msg.role,
          content: msg.content,
          thought: msg.thought,
          timestamp: new Date(msg.timestamp),
          images: msg.images || [],
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

  const handleSessionSelect = async (newSessionId: string) => {
    if (newSessionId === sessionId) return;
    await loadSessionHistory(newSessionId, true);
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
          images: msg.images || [],
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
    // Generate a new session ID
    const newSessionId = `session-${Date.now()}-${Math.random().toString(36).substring(7)}`;

    // Update URL without causing a full page reload
    window.history.pushState(null, '', `/chat/${newSessionId}`);

    // Update state to show empty chat
    setSessionId(newSessionId);
    setMessages([]);
    setHasMoreMessages(false);
    setTotalMessageCount(0);
    setIsInputVisible(true);

    // Clear input
    setInputValue('');
    setSelectedImages([]);
    setImageError(null);

    // Focus input
    setTimeout(() => {
      textareaRef.current?.focus();
    }, 0);

    // Refresh sessions list to include the new session (will be created when first message is sent)
    // No need to reload sessions here as the session doesn't exist yet
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

  // Handle authentication and routing
  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
      return;
    }

    if (!agentInfo) {
      router.push('/');
      return;
    }
  }, [agentInfo, router, user, loading]);

  // Load session history when component mounts or URL session changes
  useEffect(() => {
    if (!user?.user_id || !urlSessionId) return;

    // Load history if messages are empty (initial load or page refresh)
    if (messages.length === 0 && !isLoadingHistory) {
      loadSessionHistory(urlSessionId, false);
    }
  }, [urlSessionId, user?.user_id]);

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

  // Always show input when there are no messages (new/empty session)
  useEffect(() => {
    if (messages.length === 0 && !isLoadingHistory) {
      if (!isInputVisible) {
        setIsInputVisible(true);
      }
      // Auto-focus on new session
      textareaRef.current?.focus();
    }
  }, [messages.length, isLoadingHistory, isInputVisible]);

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

  // Merge consecutive assistant messages for display
  const processedMessages = useMemo(() => {
    if (messages.length === 0) return [];

    const result: Message[] = [];
    messages.forEach((msg) => {
      const last = result[result.length - 1];
      if (
        last &&
        last.role === 'assistant' &&
        msg.role === 'assistant' &&
        !last.isError &&
        !msg.isError
      ) {
        // Create a new object for the merged message to avoid mutating state
        const merged = { ...last };
        merged.content = (merged.content || '') + (msg.content || '');
        if (msg.thought) {
          merged.thought = (merged.thought || '') + (merged.thought ? '\n\n' : '') + msg.thought;
        }
        if (msg.images && msg.images.length > 0) {
          merged.images = [...(merged.images || []), ...(msg.images || [])];
        }
        // If either is streaming, the merged one is streaming
        merged.isStreaming = last.isStreaming || msg.isStreaming;

        result[result.length - 1] = merged;
      } else {
        result.push(msg);
      }
    });
    return result;
  }, [messages]);

  const handleCancelMessage = () => {
    // Abort the ongoing fetch request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    // Clear title polling interval
    if (titlePollIntervalRef.current) {
      clearInterval(titlePollIntervalRef.current);
      titlePollIntervalRef.current = null;
    }

    // Just set loading to false - keep the messages as they are
    setIsLoading(false);
  };

  const readFileAsDataUrl = (file: File) => new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });

  const createImageId = () => {
    if (typeof crypto !== 'undefined') {
      if ('randomUUID' in crypto) {
        return (crypto as Crypto).randomUUID();
      }
      if ('getRandomValues' in crypto) {
        const bytes = new Uint8Array(16);
        (crypto as Crypto).getRandomValues(bytes);
        const hex = Array.from(bytes)
          .map((value) => value.toString(16).padStart(2, '0'))
          .join('');
        return `img-${hex}`;
      }
    }
    return `img-${Date.now()}-${Math.random().toString(36).slice(2)}`;
  };

  const addImagesFromFiles = async (files: File[]) => {
    if (files.length === 0) return;

    setImageError(null);
    const remainingSlots = MAX_IMAGE_COUNT - selectedImages.length;
    if (remainingSlots <= 0) {
      setImageError(IMAGE_COUNT_ERROR);
      return;
    }

    const nextImages: SelectedImage[] = [];
    let errorMessage: string | null = null;
    const limitedFiles = files.slice(0, remainingSlots);

    for (const [index, file] of limitedFiles.entries()) {
      if (!file.type.startsWith('image/')) {
        if (!errorMessage) {
          errorMessage = IMAGE_TYPE_ERROR;
        }
        continue;
      }
      if (file.size > MAX_IMAGE_SIZE_BYTES) {
        if (!errorMessage) {
          errorMessage = IMAGE_SIZE_ERROR;
        }
        continue;
      }

      try {
        const dataUrl = await readFileAsDataUrl(file);
        const imageId = createImageId();
        const fallbackName = `clipboard-image-${index + 1}`;
        nextImages.push({
          id: imageId,
          dataUrl,
          name: file.name || fallbackName,
        });
      } catch (error) {
        console.error('Error reading image file:', error);
        if (!errorMessage) {
          errorMessage = IMAGE_READ_ERROR;
        }
      }
    }

    if (files.length > remainingSlots) {
      if (!errorMessage) {
        errorMessage = IMAGE_COUNT_ERROR;
      }
    }

    if (nextImages.length > 0) {
      setSelectedImages((prev) => [...prev, ...nextImages]);
    }
    if (errorMessage) {
      setImageError(errorMessage);
    }
  };

  const handleImageSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []);
    await addImagesFromFiles(files);
    event.target.value = '';
  };

  const handlePaste = async (event: React.ClipboardEvent<HTMLTextAreaElement>) => {
    const items = event.clipboardData?.items;
    if (!items || items.length === 0) return;

    const files: File[] = [];
    for (const item of Array.from(items)) {
      if (item.type.startsWith('image/')) {
        const file = item.getAsFile();
        if (file) {
          files.push(file);
        }
      }
    }

    if (files.length > 0) {
      event.preventDefault();
      await addImagesFromFiles(files);
    }
  };

  const handleRemoveImage = (imageId: string) => {
    setSelectedImages((prev) => prev.filter((image) => image.id !== imageId));
  };

  const sendMessage = async (content: string, images: SelectedImage[]) => {
    if (isLoading) return;

    const trimmedContent = content.trim();
    const hasImages = images.length > 0;
    const isRetry = !trimmedContent && !hasImages;
    if (isRetry && messages.length === 0) return;

    if (isRetry) {
      // 如果是重试且最后一条是错误消息，先将其移除，避免界面堆积错误信息
      setMessages((prev) => {
        if (prev.length > 0 && prev[prev.length - 1].isError) {
          return prev.slice(0, -1);
        }
        return prev;
      });
    } else {
      // 正常发送新消息
      const imageData = images.map((image) => image.dataUrl);
      const userMessage: Message = {
        id: `msg-${Date.now()}`,
        role: 'user',
        content: trimmedContent,
        images: imageData,
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, userMessage]);
      setInputValue('');
      setSelectedImages([]);
      setImageError(null);
    }

    setIsLoading(true);

    // Create a new AbortController for this request
    abortControllerRef.current = new AbortController();

    // Only poll for session title on the first message in this session
    // Check if this is the first user message (before adding the new one, we had 0 user messages)
    const isFirstMessage = messages.filter(m => m.role === 'user').length === 0;
    if (isFirstMessage) {
      // Clear any existing title polling interval
      if (titlePollIntervalRef.current) {
        clearInterval(titlePollIntervalRef.current);
      }

      let pollCount = 0;
      const maxPolls = 30; // Poll up to 30 times
      titlePollIntervalRef.current = setInterval(async () => {
        const fetchedSessions = await loadSessions(true);
        const currentSession = fetchedSessions?.find((s: Session) => s.session_id === sessionId);

        if (currentSession?.title) {
          if (titlePollIntervalRef.current) {
            clearInterval(titlePollIntervalRef.current);
            titlePollIntervalRef.current = null;
          }
          return;
        }

        pollCount++;
        if (pollCount >= maxPolls) {
          if (titlePollIntervalRef.current) {
            clearInterval(titlePollIntervalRef.current);
            titlePollIntervalRef.current = null;
          }
        }
      }, 1000); // Poll every 1 second
    }

    try {
      const imageData = isRetry ? [] : images.map((image) => image.dataUrl);
      const response = await authenticatedFetch(`/api/${agentId}/chats/${sessionId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          user_id: user?.user_id,
          session_id: sessionId,
          message: trimmedContent,
          images: imageData,
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
      let assistantImages: string[] = [];

      if (reader) {
        let buffer = '';
        let currentEventType = 'data'; // Track the current SSE event type

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          const chunk = decoder.decode(value, { stream: true });
          buffer += chunk;
          const lines = buffer.split('\n');
          buffer = lines.pop() || '';

          for (const line of lines) {
            const trimmedLine = line.trim();

            // Handle SSE event type line (e.g., "event: error", "event: data", "event: stop", "event: heartbeat")
            if (trimmedLine.startsWith('event:')) {
              currentEventType = trimmedLine.substring(6).trim();
              continue;
            }

            if (trimmedLine.startsWith('data:')) {
              try {
                const jsonStr = trimmedLine.substring(5).trim();
                if (!jsonStr) continue;

                const data = JSON.parse(jsonStr);

                // Ignore heartbeat events (used to keep connection alive)
                if (currentEventType === 'heartbeat') {
                  currentEventType = 'data';
                  continue;
                }

                // Handle error events
                if (currentEventType === 'error') {
                  const errorMessage: Message = {
                    id: `msg-${Date.now()}-error`,
                    role: 'assistant',
                    content: `❌ **错误**: ${data.error || '发生未知错误，请稍后重试。'}`,
                    timestamp: new Date(),
                    isError: true,
                  };
                  setMessages((prev) => [...prev, errorMessage]);
                  setIsLoading(false);
                  // Reset event type back to data for next event
                  currentEventType = 'data';
                  continue;
                }

                // Handle stop events
                if (currentEventType === 'stop') {
                  // Reset event type back to data for next event
                  currentEventType = 'data';
                  continue;
                }

                // Handle images data
                if (data.images && Array.isArray(data.images)) {
                  assistantImages = [...assistantImages, ...data.images];

                  // Update last message with images
                  setMessages((prev) => {
                    const newMessages = [...prev];
                    const lastIndex = newMessages.length - 1;
                    if (lastIndex >= 0 && newMessages[lastIndex].role === 'assistant') {
                      newMessages[lastIndex] = {
                        ...newMessages[lastIndex],
                        images: assistantImages,
                        isStreaming: true,
                      };
                    }
                    return newMessages;
                  });
                }

                if (data.content) {
                  // Check if this is a duplicate complete message (identical to accumulated content)
                  const isCompleteDuplicate = !data.is_thought && data.content === assistantContent;

                  if (!isCompleteDuplicate) {
                    // Check if author changed - create new message block
                    if (data.author && data.author !== currentAuthor) {
                      currentAuthor = data.author;
                      assistantContent = data.is_thought ? '' : data.content;
                      assistantThought = data.is_thought ? data.content : '';
                      assistantImages = [];

                      // Create new message for new author
                      const newMessage: Message = {
                        id: `msg-${Date.now()}-${currentAuthor}`,
                        role: 'assistant',
                        content: assistantContent,
                        thought: assistantThought,
                        timestamp: new Date(),
                        author: currentAuthor,
                        isStreaming: true,
                        images: [],
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
                            images: assistantImages,
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
      // Clear title polling on error
      if (titlePollIntervalRef.current) {
        clearInterval(titlePollIntervalRef.current);
        titlePollIntervalRef.current = null;
      }

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
          isError: true,
        };
        setMessages((prev) => [...prev, errorMessage]);
      }
    } finally {
      setIsLoading(false);
      abortControllerRef.current = null;
    }
  };

  const canSend = inputValue.trim().length > 0 || selectedImages.length > 0;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    sendMessage(inputValue, selectedImages);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Submit on Enter without Shift, but NOT while composing (IME)
    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
      if (!canSend) return;
      e.preventDefault();
      sendMessage(inputValue, selectedImages);
    }
    // Allow Shift+Enter for new line (default behavior)
  };

  const handleExampleClick = (example: string) => {
    if (isLoading) return;
    sendMessage(example, []);
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
        <div className="md:hidden fixed top-3 left-3 z-30">
          <Button
            onClick={() => setIsMobileSidebarOpen(true)}
            size="icon"
            variant="outline"
            className="h-10 w-10 rounded-full bg-background shadow-lg tap-highlight-transparent min-h-[44px] min-w-[44px]"
            aria-label="打开菜单"
          >
            <Menu className="h-5 w-5" />
          </Button>
        </div>

        {/* Messages Area */}
        <div
          ref={scrollContainerRef}
          className="flex-1 overflow-y-auto no-scrollbar mobile-scroll"
          onScroll={handleScroll}
        >
          <div className="flex flex-col items-center">
            <div className="w-full max-w-5xl px-3 sm:px-4 md:px-6 py-6 sm:py-8 md:py-10 space-y-6 sm:space-y-8">{/* Loading older messages indicator */}
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
                <div className="text-center py-12 sm:py-16 md:py-20 animate-fade-in px-4">
                  <div className="flex justify-center mb-4 sm:mb-6">
                    <div className="p-3 sm:p-4 bg-secondary rounded-xl sm:rounded-2xl">
                      <span className="text-3xl sm:text-4xl">{agentInfo.icon}</span>
                    </div>
                  </div>
                  <h2 className="text-2xl font-semibold mb-8 tracking-tight">
                    {agentInfo.name} 能够为您做什么？
                  </h2>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 max-w-2xl mx-auto">
                    {agentInfo.examples.map((example, index) => (
                      <button
                        key={index}
                        onClick={() => handleExampleClick(example)}
                        className="p-3 sm:p-4 text-left border border-zinc-200 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700 rounded-lg sm:rounded-xl bg-zinc-50/50 dark:bg-zinc-900/30 hover:bg-white dark:hover:bg-zinc-900 active:bg-zinc-100 dark:active:bg-zinc-800 shadow-sm hover:shadow-md transition-all duration-200 text-xs sm:text-sm text-balance tap-highlight-transparent min-h-[44px]"
                      >
                        {example}
                      </button>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="space-y-6 sm:space-y-8 animate-fade-in">
                  {processedMessages.map((message) => (
                    <div
                      key={message.id}
                      className={cn(
                        "flex w-full",
                        message.role === 'user' ? "justify-end" : "justify-start"
                      )}
                    >
                      <div className={cn(
                        "flex gap-2 sm:gap-3 md:gap-4 max-w-[95%] sm:max-w-[90%] md:max-w-[85%]",
                        message.role === 'user' ? "flex-row-reverse" : "flex-row"
                      )}>
                        {message.role === 'assistant' ? (
                          <AIAvatar icon={agentInfo.icon} />
                        ) : (
                          <UserAvatar user={user} />
                        )}

                        <div className={cn(
                          "relative text-sm w-full leading-relaxed",
                          message.role === 'user'
                            ? "bg-zinc-100 dark:bg-zinc-800 px-4 py-2.5 rounded-2xl rounded-tr-sm self-end max-w-fit"
                            : "pt-1 flex-1"
                        )}>
                          {message.role === 'assistant' ? (
                            <AIMessageContent
                              content={message.content}
                              thought={message.thought}
                              isStreaming={message.isStreaming}
                              images={message.images}
                              isError={message.isError}
                              onRetry={() => sendMessage("", [])}
                            />
                          ) : (
                            <div className="space-y-2">
                              {message.images && message.images.length > 0 && (
                                <div className="space-y-2">
                                  {message.images.map((imageData, index) => (
                                    <img
                                      key={index}
                                      src={imageData}
                                      alt={`用户上传图片 ${index + 1}`}
                                      className="max-w-full h-auto rounded-lg border shadow-sm"
                                      loading="lazy"
                                    />
                                  ))}
                                </div>
                              )}
                              {message.content && (
                                <div className="prose prose-sm prose-zinc dark:prose-invert max-w-none prose-p:leading-relaxed prose-p:my-3 prose-pre:p-0 prose-pre:rounded-lg prose-headings:my-4 prose-headings:font-semibold break-words">
                                  <ReactMarkdown
                                    remarkPlugins={markdownRemarkPlugins}
                                    rehypePlugins={markdownRehypePlugins}
                                    components={markdownComponents}
                                  >
                                    {message.content}
                                  </ReactMarkdown>
                                </div>
                              )}
                            </div>
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
          "absolute bottom-0 left-0 w-full md:pl-[260px] bg-gradient-to-t from-background via-background to-transparent pt-4 sm:pt-6 pb-3 sm:pb-4 transition-transform duration-300 ease-in-out",
          !isInputVisible && "translate-y-full"
        )}>
          <div className="max-w-4xl mx-auto px-3 sm:px-4 md:px-6">
            <div className="relative flex flex-col w-full bg-zinc-50/50 dark:bg-zinc-900/30 backdrop-blur-xl rounded-2xl border border-zinc-200 dark:border-zinc-800 shadow-sm hover:border-zinc-300 dark:hover:border-zinc-700 focus-within:border-zinc-300 dark:focus-within:border-zinc-600 focus-within:ring-4 focus-within:ring-zinc-900/5 dark:focus-within:ring-zinc-100/5 transition-all duration-300 overflow-hidden">
              {selectedImages.length > 0 && (
                <div className="flex flex-wrap gap-2 px-3 pt-3">
                  {selectedImages.map((image, index) => (
                    <div key={image.id} className="relative group">
                      <img
                        src={image.dataUrl}
                        alt={image.name || `已选图片 ${index + 1}`}
                        className="h-16 w-16 object-cover rounded-lg border border-zinc-200 dark:border-zinc-700"
                      />
                      <button
                        type="button"
                        onClick={() => handleRemoveImage(image.id)}
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
              <form onSubmit={handleSubmit} className="w-full flex items-center p-2 gap-2">
                <input
                  ref={imageInputRef}
                  type="file"
                  accept="image/*"
                  multiple
                  className="hidden"
                  onChange={handleImageSelect}
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
                  onChange={(e) => setInputValue(e.target.value)}
                  onKeyDown={handleKeyDown}
                  onPaste={handlePaste}
                  onFocus={() => setIsInputVisible(true)}
                  placeholder={isLoadingHistory ? "正在加载历史记录..." : `给 ${agentInfo.name} 发送消息`}
                  className="flex-1 min-h-[44px] max-h-[160px] border-0 bg-transparent shadow-none focus-visible:ring-0 px-3.5 py-3 text-base sm:text-sm overflow-y-auto resize-none placeholder:text-zinc-500/60 dark:placeholder:text-zinc-400/50 no-scrollbar leading-relaxed transition-colors"
                  disabled={isLoading || isLoadingHistory}
                  autoComplete="off"
                  rows={1}
                />
                {isLoading ? (
                  <Button
                    type="button"
                    size="icon"
                    onClick={handleCancelMessage}
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
      </div>
    </div>
  );
}
