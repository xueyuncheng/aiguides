import ReactMarkdown from 'react-markdown';
import { markdownRemarkPlugins, markdownRehypePlugins, markdownComponents } from '../utils/markdown';

interface UserMessageProps {
  content: string;
  images?: string[];
  fileNames?: string[];
}

export function UserMessage({ content, images, fileNames }: UserMessageProps) {
  return (
    <div className="space-y-2">
      {images && images.length > 0 && (
        <div className="space-y-2">
          {images.map((imageData, index) => {
            const isPdf = imageData.startsWith('data:application/pdf');
            const fileName = fileNames?.[index] || (isPdf ? `PDF 文件 ${index + 1}` : `图片 ${index + 1}`);
            return isPdf ? (
              <div
                key={index}
                className="inline-flex items-center gap-2 px-3 py-2 rounded-lg border border-zinc-200 dark:border-zinc-700 bg-zinc-100 dark:bg-zinc-800"
              >
                <div className="flex items-center justify-center w-8 h-8 rounded bg-red-100 dark:bg-red-900/20">
                  <span className="text-xs font-semibold text-red-600 dark:text-red-400">PDF</span>
                </div>
                <span className="text-sm text-zinc-700 dark:text-zinc-300">{fileName}</span>
              </div>
            ) : (
              <img
                key={index}
                src={imageData}
                alt={fileName}
                className="max-w-full h-auto rounded-lg border shadow-sm"
                loading="lazy"
              />
            );
          })}
        </div>
      )}
      {content && (
        <div className="prose prose-sm prose-zinc dark:prose-invert max-w-none prose-p:leading-relaxed prose-p:my-3 prose-pre:p-0 prose-pre:rounded-lg prose-headings:my-4 prose-headings:font-semibold break-words">
          <ReactMarkdown
            remarkPlugins={markdownRemarkPlugins}
            rehypePlugins={markdownRehypePlugins}
            components={markdownComponents}
          >
            {content}
          </ReactMarkdown>
        </div>
      )}
    </div>
  );
}
