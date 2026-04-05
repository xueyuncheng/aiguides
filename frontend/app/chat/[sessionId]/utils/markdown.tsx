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
import SvgBlock from './SvgBlock';

function getCodeLanguage(className?: string): string | undefined {
  return /language-([^\s]+)/.exec(className || '')?.[1]?.toLowerCase();
}

function isSvgLanguage(language?: string): boolean {
  return language === 'svg' || language === 'xml' || language === 'html' || language === 'svg+xml';
}

function looksLikeSvgMarkup(code: string): boolean {
  const normalizedCode = code.trimStart().toLowerCase();
  return normalizedCode.startsWith('<svg') || normalizedCode.startsWith('<?xml');
}

function getCodeChild(children: React.ReactNode): React.ReactElement<{ className?: string }> | null {
  if (!React.isValidElement(children)) {
    return null;
  }

  return children as React.ReactElement<{ className?: string }>;
}

function isSvgCodeElement(element: React.ReactElement<{ className?: string }> | null): boolean {
  const language = getCodeLanguage(element?.props.className);
  const codeString = React.Children.toArray(element?.props.children).join('').replace(/\n$/, '');

  return language === 'svg' || (isSvgLanguage(language) && looksLikeSvgMarkup(codeString));
}

// Markdown plugins
// Enable single-dollar inline math (e.g. `$T$`) for better LaTeX compatibility in chat responses.
// Preprocess markdown to escape currency dollar signs (e.g. `$100`) before remark-math parses them.
export const markdownRemarkPlugins: PluggableList = [remarkGfm, remarkBreaks, remarkMath];

const currencyPattern = /(?<!\\)\$(\d+(?:,\d{3})*(?:\.\d+)?)(?=$|[\s),?!:;%\]]|\.(?!\d))/g;
const fencedCodeBlockPattern = /(```[\s\S]*?```)/g;
const rawSvgPattern = /(?:<\?xml[\s\S]*?\?>\s*)?(?:<!DOCTYPE[\s\S]*?>\s*)?<svg\b[\s\S]*?<\/svg>/gi;

function transformOutsideCodeFences(content: string, transform: (segment: string) => string): string {
  return content
    .split(fencedCodeBlockPattern)
    .map((segment, index) => (index % 2 === 1 ? segment : transform(segment)))
    .join('');
}

function normalizeRawSvgBlocks(content: string): string {
  return transformOutsideCodeFences(content, (segment) => (
    segment.replace(rawSvgPattern, (svg) => `\n\`\`\`svg\n${svg.trim()}\n\`\`\`\n`)
  ));
}

/**
 * Escapes dollar signs that look like standalone currency amounts,
 * preventing remark-math from treating them as LaTeX delimiters.
 * e.g. "$100" -> "\$100", but "$2^{30} \approx 10^9$" remains unchanged.
 */
export function preprocessMarkdown(content: string): string {
  return transformOutsideCodeFences(normalizeRawSvgBlocks(content), (segment) => (
    segment.replace(currencyPattern, (_, amount: string) => `\\$${amount}`)
  ));
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
  const language = getCodeLanguage(className);
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
        <span>{language || 'code'}</span>
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
        language={language || 'text'}
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
  blockquote: ({ children, ...props }) => (
    <blockquote
      {...props}
      className="my-3 border-l-2 border-muted-foreground/30 bg-muted/40 px-3 py-2 text-sm text-muted-foreground dark:bg-muted/20 [&>p]:m-0"
    >
      {children}
    </blockquote>
  ),
  a: (props) => (
    <a {...props} target="_blank" rel="noopener noreferrer" className="text-primary underline underline-offset-4 font-medium" />
  ),
  img: ({ src, alt, ...props }) => {
    if (!src) return null;
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={src} alt={alt || ''} {...props} className="max-w-full h-auto rounded-lg border my-6" loading="lazy" />;
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
  pre: ({ children, ...props }) => {
    const codeChild = getCodeChild(children);

    if (isSvgCodeElement(codeChild)) {
      return <>{children}</>;
    }

    return <pre {...props}>{children}</pre>;
  },
  code: ({ className, children, ...props }) => {
    const language = getCodeLanguage(className);
    const codeString = String(children).replace(/\n$/, '');
    const isSvgBlock = language === 'svg' || (isSvgLanguage(language) && looksLikeSvgMarkup(codeString));
    const isInline = !language;

    if (isSvgBlock) {
      return <SvgBlock>{children}</SvgBlock>;
    }

    if (isInline) {
      return (
        <code className="bg-muted px-1.5 py-0.5 rounded text-[13px] font-mono text-foreground break-all" {...props}>
          {children}
        </code>
      );
    }
    return (
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
