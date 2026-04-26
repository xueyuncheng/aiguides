import { AudioLines, FileText } from 'lucide-react';
import { CodeBlock, fencedCodeBlockPattern } from '../utils/markdown';
import { VoiceAudioPlayer, resolveVoiceAudioUrl } from './VoiceAudioPlayer';
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

/**
 * Renders user message content as plain text with fenced code block support.
 * Plain text is rendered with `white-space: pre-wrap` to preserve spaces,
 * indentation, and newlines exactly as typed. Fenced code blocks (```) are
 * rendered with syntax highlighting via CodeBlock.
 */
function renderUserContent(content: string) {
  const parts = content.split(fencedCodeBlockPattern);
  return parts.map((part, i) => {
    if (i % 2 === 1) {
      // Fenced code block — extract language and code, render with CodeBlock
      const match = part.match(/^```(\w*)\n?([\s\S]*?)```$/);
      const language = match?.[1] || '';
      const code = match?.[2]?.replace(/\n$/, '') || part;
      return (
        <CodeBlock key={i} className={language ? `language-${language}` : undefined}>
          {code}
        </CodeBlock>
      );
    }
    // Plain text — preserve whitespace exactly as typed
    if (!part) return null;
    return (
      <span key={i} className="whitespace-pre-wrap break-words [overflow-wrap:anywhere]">
        {part}
      </span>
    );
  });
}

interface UserMessageProps {
  content: string;
  images?: string[];
  fileNames?: string[];
  files?: MessageFile[];
  voiceAudioFileId?: number;
  voiceAudioUrl?: string;
}

export function UserMessage({ content, images, fileNames, files, voiceAudioFileId, voiceAudioUrl }: UserMessageProps) {
  return (
    <div className="space-y-2">
      {resolveVoiceAudioUrl(voiceAudioFileId, voiceAudioUrl) && (
        <VoiceAudioPlayer
          audioUrl={resolveVoiceAudioUrl(voiceAudioFileId, voiceAudioUrl)!}
          className="max-w-sm"
        />
      )}
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
        <div className="text-sm leading-relaxed min-w-0 break-words [overflow-wrap:anywhere]">
          {renderUserContent(content)}
        </div>
      )}
    </div>
  );
}
