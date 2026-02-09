import { useState, useEffect, useRef, memo } from 'react';
import ReactMarkdown from 'react-markdown';
import { Button } from '@/app/components/ui/button';
import { Code2, Eye, Copy, Check, X, ChevronDown, ChevronRight, RotateCcw } from 'lucide-react';
import { cn } from '@/app/lib/utils';
import { markdownRemarkPlugins, markdownRehypePlugins, markdownComponents } from '../utils/markdown';
import { FEEDBACK_TIMEOUT_MS } from '../constants';

interface AIMessageContentProps {
  content: string;
  thought?: string;
  isStreaming?: boolean;
  images?: string[];
  isError?: boolean;
  onRetry?: () => void;
}

export const AIMessageContent = memo(({
  content,
  thought,
  isStreaming,
  images,
  isError,
  onRetry
}: AIMessageContentProps) => {
  const [showRaw, setShowRaw] = useState(false);
  const [isThoughtExpanded, setIsThoughtExpanded] = useState(false);
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
          <div className="prose prose-sm prose-zinc dark:prose-invert max-w-none prose-p:leading-relaxed prose-p:my-3 prose-pre:p-0 prose-pre:rounded-lg prose-headings:my-4 prose-headings:font-semibold prose-table:border-collapse prose-table:w-full prose-th:border prose-td:border prose-th:border-border prose-td:border-border prose-th:bg-muted/60 prose-th:px-3 prose-th:py-2 prose-td:px-3 prose-td:py-2">
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
