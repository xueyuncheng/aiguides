import { useState, useEffect, useRef, memo } from 'react';
import ReactMarkdown from 'react-markdown';
import { Button } from '@/app/components/ui/button';
import { Code2, Eye, Copy, Check, X, ChevronDown, ChevronRight, RotateCcw } from 'lucide-react';
import { cn } from '@/app/lib/utils';
import { markdownRemarkPlugins, markdownRehypePlugins, markdownComponents, preprocessMarkdown } from '../utils/markdown';
import { FEEDBACK_TIMEOUT_MS } from '../constants';
import type { ToolCallItem } from '../types';

interface AIMessageContentProps {
  content: string;
  thought?: string;
  isStreaming?: boolean;
  images?: string[];
  isError?: boolean;
  onRetry?: () => void;
  thoughtStorageKey?: string;
  toolCalls?: ToolCallItem[];
}

export const AIMessageContent = memo(({
  content,
  thought,
  isStreaming,
  images,
  isError,
  onRetry,
  thoughtStorageKey,
  toolCalls,
}: AIMessageContentProps) => {
  const [showRaw, setShowRaw] = useState(false);
  const [expandedToolCallIndexes, setExpandedToolCallIndexes] = useState<number[]>([]);
  const [isThoughtExpanded, setIsThoughtExpanded] = useState(() => {
    if (!thought || !thoughtStorageKey || typeof window === 'undefined') return false;

    try {
      return window.localStorage.getItem(thoughtStorageKey) === 'true';
    } catch (err) {
      console.error('Failed to read thought expansion state:', err);
      return false;
    }
  });
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

  useEffect(() => {
    if (!thought || !thoughtStorageKey || typeof window === 'undefined') return;

    try {
      window.localStorage.setItem(thoughtStorageKey, String(isThoughtExpanded));
    } catch (err) {
      console.error('Failed to persist thought expansion state:', err);
    }
  }, [isThoughtExpanded, thought, thoughtStorageKey]);

  const handleCopy = async () => {
    if (copyTimeoutRef.current) {
      clearTimeout(copyTimeoutRef.current);
    }
    if (errorTimeoutRef.current) {
      clearTimeout(errorTimeoutRef.current);
    }

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

  const toggleToolArgs = (index: number) => {
    setExpandedToolCallIndexes((prev) => (
      prev.includes(index) ? prev.filter((item) => item !== index) : [...prev, index]
    ));
  };

  const formatToolArgs = (args?: Record<string, unknown>) => {
    if (!args || Object.keys(args).length === 0) {
      return '';
    }

    return JSON.stringify(args, null, 2);
  };

  const resolvedContent = content.replaceAll(
    '(download_path)',
    (() => {
      const fileGetCall = [...(toolCalls || [])].reverse().find((tc) => (
        tc.toolName === 'file_get' && typeof tc.result?.download_path === 'string'
      ));
      const downloadPath = fileGetCall?.result?.download_path;
      return typeof downloadPath === 'string' && downloadPath.trim() !== ''
        ? `(${downloadPath})`
        : '(download_path)';
    })()
  );

  return (
    <div className="group">
      {/* Thought Process section */}
      {thought && (
        <div className="mb-4">
          <button
            onClick={() => setIsThoughtExpanded(!isThoughtExpanded)}
            className="flex items-center gap-2 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors py-1.5 px-3 rounded-md border border-muted-foreground/20 bg-muted/50"
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
            isThoughtExpanded ? "max-h-[99999px] opacity-100" : "max-h-0 opacity-0"
          )}>
            <div className="text-xs text-muted-foreground/90 leading-relaxed pl-4 border-l-2 border-primary/20 py-1 whitespace-pre-wrap bg-muted/20 rounded-r-md max-h-[600px] overflow-y-auto">
              {thought}
              {isStreaming && (
                <span className="inline-block w-1 h-3 ml-1 bg-primary animate-pulse align-middle" />
              )}
            </div>
          </div>
        </div>
      )}

      {/* Tool calls section */}
      {toolCalls && toolCalls.length > 0 && (
        <div className="mb-3 flex flex-col gap-1.5">
          {toolCalls.map((tc, i) => (
            <div
              key={i}
              className="text-xs text-muted-foreground bg-muted/40 border border-muted-foreground/10 rounded-md overflow-hidden max-w-full"
            >
              <div className="flex items-center gap-2 px-3 py-1.5">
                {isStreaming && i === toolCalls.length - 1 ? (
                  <div className="flex space-x-0.5 shrink-0">
                    <div className="w-1 h-1 bg-primary/60 rounded-full animate-bounce [animation-delay:-0.3s]" />
                    <div className="w-1 h-1 bg-primary/60 rounded-full animate-bounce [animation-delay:-0.15s]" />
                    <div className="w-1 h-1 bg-primary/60 rounded-full animate-bounce" />
                  </div>
                ) : (
                  <div className="w-1 h-1 bg-muted-foreground/40 rounded-full shrink-0" />
                )}
                <span className="min-w-0 flex-1 break-all">{tc.label}</span>
                {tc.args && Object.keys(tc.args).length > 0 && (
                  <button
                    type="button"
                    onClick={() => toggleToolArgs(i)}
                    className="inline-flex items-center gap-1 rounded border border-muted-foreground/15 px-2 py-0.5 text-[11px] hover:bg-muted/60 transition-colors shrink-0"
                    aria-expanded={expandedToolCallIndexes.includes(i)}
                  >
                    {expandedToolCallIndexes.includes(i) ? (
                      <ChevronDown className="h-3 w-3" />
                    ) : (
                      <ChevronRight className="h-3 w-3" />
                    )}
                    <span>参数</span>
                  </button>
                )}
              </div>
              {tc.args && Object.keys(tc.args).length > 0 && expandedToolCallIndexes.includes(i) && (
                <pre className="border-t border-muted-foreground/10 bg-background/70 px-3 py-2 overflow-x-auto whitespace-pre-wrap break-all text-[11px] leading-relaxed">
                  {formatToolArgs(tc.args)}
                </pre>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Content display */}
      <div className="relative">
        {/* Display images if present */}
        {images && images.length > 0 && (
          <div className="mb-4 space-y-3">
            {images.map((imageData, index) => (
              // eslint-disable-next-line @next/next/no-img-element
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
          <div className="prose prose-sm prose-zinc dark:prose-invert max-w-none prose-p:leading-relaxed prose-p:my-3 prose-pre:p-0 prose-pre:rounded-lg prose-headings:my-4 prose-headings:font-semibold prose-table:border-collapse prose-table:w-full prose-th:border prose-td:border prose-th:border-border prose-td:border-border prose-th:bg-muted/60 prose-th:px-3 prose-th:py-2 prose-td:px-3 prose-td:py-2 break-words [overflow-wrap:anywhere]">
            <ReactMarkdown
              remarkPlugins={markdownRemarkPlugins}
              rehypePlugins={markdownRehypePlugins}
              components={markdownComponents}
            >
              {preprocessMarkdown(resolvedContent)}
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
