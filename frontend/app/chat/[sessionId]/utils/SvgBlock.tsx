import React, { useState } from 'react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { Copy, Check, Code, Image as ImageIcon } from 'lucide-react';
import DOMPurify from 'dompurify';

// SvgBlock renders a sanitized SVG inline with a code/preview toggle and a copy button.
const SvgBlock = ({ children }: { children: React.ReactNode }) => {
  const [showCode, setShowCode] = useState(false);
  const [codeCopied, setCodeCopied] = useState(false);
  const svgString = String(children).replace(/\n$/, '');

  const sanitizedSvg = typeof window !== 'undefined'
    ? DOMPurify.sanitize(svgString, { USE_PROFILES: { svg: true, svgFilters: true } })
    : '';

  const handleCodeCopy = async () => {
    try {
      await navigator.clipboard.writeText(svgString);
      setCodeCopied(true);
      setTimeout(() => setCodeCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy SVG code:', err);
    }
  };

  return (
    <div className="my-6 rounded-lg overflow-hidden border shadow-sm">
      <div className="px-4 py-2 text-xs bg-zinc-900 border-b border-zinc-800 flex justify-between items-center text-zinc-300">
        <span>svg</span>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowCode((v) => !v)}
            className="flex items-center gap-1.5 px-2 py-1 rounded hover:bg-zinc-700 hover:text-white transition-colors duration-200"
            title={showCode ? '显示图形' : '查看代码'}
            aria-label={showCode ? '显示图形' : '查看代码'}
          >
            {showCode ? (
              <>
                <ImageIcon className="h-3.5 w-3.5" />
                <span>图形</span>
              </>
            ) : (
              <>
                <Code className="h-3.5 w-3.5" />
                <span>代码</span>
              </>
            )}
          </button>
          <button
            onClick={handleCodeCopy}
            className="flex items-center gap-1.5 px-2 py-1 rounded hover:bg-zinc-700 hover:text-white transition-colors duration-200"
            title={codeCopied ? '已复制' : '复制代码'}
            aria-label={codeCopied ? '已复制' : '复制代码'}
          >
            {codeCopied ? (
              <>
                <Check className="h-3.5 w-3.5" />
                <span>已复制</span>
              </>
            ) : (
              <>
                <Copy className="h-3.5 w-3.5" />
                <span>复制</span>
              </>
            )}
          </button>
        </div>
      </div>
      {showCode ? (
        <div className="bg-zinc-950 dark:bg-zinc-900 text-zinc-50 overflow-x-auto">
          <SyntaxHighlighter
            language="xml"
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
            {svgString}
          </SyntaxHighlighter>
        </div>
      ) : (
        <div
          className="flex items-center justify-center bg-white dark:bg-zinc-950 p-4 overflow-x-auto [&>svg]:max-w-full [&>svg]:h-auto"
          dangerouslySetInnerHTML={{ __html: sanitizedSvg }}
        />
      )}
    </div>
  );
};

export default SvgBlock;
