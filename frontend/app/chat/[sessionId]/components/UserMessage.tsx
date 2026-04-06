import { AudioLines, FileText } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import { markdownRemarkPlugins, markdownRehypePlugins, markdownComponents, preprocessMarkdown } from '../utils/markdown';
import type { MessageFile } from '../types';

const getFileBadge = (mimeType: string) => {
  if (mimeType === 'application/pdf') {
    return {
      icon: FileText,
      iconClassName: 'text-red-600 dark:text-red-400',
      badgeClassName: 'bg-red-100 dark:bg-red-900/20',
    };
  }

  if (mimeType.startsWith('audio/')) {
    return {
      icon: AudioLines,
      iconClassName: 'text-blue-600 dark:text-blue-400',
      badgeClassName: 'bg-blue-100 dark:bg-blue-900/20',
    };
  }

  return {
    icon: FileText,
    iconClassName: 'text-zinc-700 dark:text-zinc-300',
    badgeClassName: 'bg-zinc-200 dark:bg-zinc-700',
  };
};

interface UserMessageProps {
  content: string;
  images?: string[];
  fileNames?: string[];
  files?: MessageFile[];
}

export function UserMessage({ content, images, fileNames, files }: UserMessageProps) {
  return (
    <div className="space-y-2">
      {files && files.length > 0 && (
        <div className="space-y-2">
          {files.map((file, index) => {
            const fileName = file.name || file.label || `文件 ${index + 1}`;
            const badge = getFileBadge(file.mime_type);
            const BadgeIcon = badge.icon;
            return (
              <div
                key={`${file.mime_type}-${fileName}-${index}`}
                className="inline-flex items-center gap-2 px-3 py-2 rounded-lg border border-zinc-200 dark:border-zinc-700 bg-zinc-100 dark:bg-zinc-800"
              >
                <div className={`flex items-center justify-center w-8 h-8 rounded ${badge.badgeClassName}`}>
                  <BadgeIcon className={`h-4 w-4 ${badge.iconClassName}`} />
                </div>
                <span className="text-sm text-zinc-700 dark:text-zinc-300">{fileName}</span>
              </div>
            );
          })}
        </div>
      )}
      {images && images.length > 0 && (
        <div className="space-y-2">
          {images.map((imageData, index) => {
            const isPdf = imageData.startsWith('data:application/pdf');
            const isAudio = imageData.startsWith('data:audio/');
            const fileName = fileNames?.[index] || `图片 ${index + 1}`;
            return isPdf || isAudio ? (
              <div
                key={index}
                className="inline-flex items-center gap-2 px-3 py-2 rounded-lg border border-zinc-200 dark:border-zinc-700 bg-zinc-100 dark:bg-zinc-800"
              >
                <div className={`flex items-center justify-center w-8 h-8 rounded ${isPdf ? 'bg-red-100 dark:bg-red-900/20' : 'bg-blue-100 dark:bg-blue-900/20'}`}>
                  {isPdf ? (
                    <FileText className="h-4 w-4 text-red-600 dark:text-red-400" />
                  ) : (
                    <AudioLines className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                  )}
                </div>
                <span className="text-sm text-zinc-700 dark:text-zinc-300">{fileName}</span>
              </div>
            ) : (
              // eslint-disable-next-line @next/next/no-img-element
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
        <div className="prose prose-sm prose-zinc dark:prose-invert max-w-none min-w-0 prose-p:my-3 prose-p:leading-relaxed prose-pre:p-0 prose-pre:rounded-lg prose-headings:my-4 prose-headings:font-semibold break-words break-all [overflow-wrap:anywhere]">
          <ReactMarkdown
            remarkPlugins={markdownRemarkPlugins}
            rehypePlugins={markdownRehypePlugins}
            components={markdownComponents}
          >
            {preprocessMarkdown(content)}
          </ReactMarkdown>
        </div>
      )}
    </div>
  );
}
