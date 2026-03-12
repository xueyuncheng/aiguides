import React, { useState } from 'react';
import type { Components } from 'react-markdown';
import type { PluggableList } from 'unified';
import remarkGfm from 'remark-gfm';
import remarkBreaks from 'remark-breaks';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { Copy, Check } from 'lucide-react';
import { cn } from '@/app/lib/utils';

// Markdown plugins
// Enable single-dollar inline math (e.g. `$T$`) for better LaTeX compatibility in chat responses.
// Preprocess markdown to escape currency dollar signs (e.g. `$100`) before remark-math parses them.
export const markdownRemarkPlugins: PluggableList = [remarkGfm, remarkBreaks, remarkMath];

const currencyPattern = /(?<!\\)\$(\d+(?:,\d{3})*(?:\.\d+)?)(?=$|[\s),.?!:;%\]])/g;

/**
 * Escapes dollar signs that look like standalone currency amounts,
 * preventing remark-math from treating them as LaTeX delimiters.
 * e.g. "$100" -> "\$100", but "$2^{30} \approx 10^9$" remains unchanged.
 */
export function preprocessMarkdown(content: string): string {
  return content.replace(currencyPattern, '\\$$1');
}
export const markdownRehypePlugins: PluggableList = [rehypeKatex];

// Table styles
const markdownTableStyles = {
  wrapper: "my-4 w-full overflow-x-auto",
  table: "w-full border-collapse border border-border text-sm",
  th: "border border-border bg-muted/60 px-3 py-2 text-left font-semibold",
  td: "border border-border px-3 py-2 align-top",
};

// Code Block component with syntax highlighting and copy button
const CodeBlock = ({ className, children }: { className?: string; children: React.ReactNode }) => {
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
    <div className="my-6 rounded-lg overflow-x-auto border bg-zinc-950 dark:bg-zinc-900 text-zinc-50 relative group shadow-sm">
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
};

// Markdown components configuration
export const markdownComponents: Components = {
  a: (props) => (
    <a {...props} target="_blank" rel="noopener noreferrer" className="text-primary underline underline-offset-4 font-medium" />
  ),
  img: ({ src, ...props }) => {
    if (!src) return null;
    return <img src={src} {...props} className="max-w-full h-auto rounded-lg border my-6" loading="lazy" />;
  },
  table: ({ className, ...props }: React.TableHTMLAttributes<HTMLTableElement>) => (
    <div className={markdownTableStyles.wrapper}>
      <table
        {...props}
        className={cn(markdownTableStyles.table, className)}
      />
    </div>
  ),
  th: ({ className, ...props }: React.ThHTMLAttributes<HTMLTableCellElement>) => (
    <th
      {...props}
      className={cn(markdownTableStyles.th, className)}
    />
  ),
  td: ({ className, ...props }: React.TdHTMLAttributes<HTMLTableCellElement>) => (
    <td
      {...props}
      className={cn(markdownTableStyles.td, className)}
    />
  ),
  code: ({ className, children, ...props }) => {
    const match = /language-(\w+)/.exec(className || '');
    const isInline = !match;
    return isInline ? (
      <code className="bg-muted px-1.5 py-0.5 rounded text-[13px] font-mono text-foreground break-all" {...props}>
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
