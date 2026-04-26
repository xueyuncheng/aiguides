'use client';

import { memo, useCallback, useEffect, useRef, useState } from 'react';
import { Play, Pause, Volume2 } from 'lucide-react';
import { cn } from '@/app/lib/utils';

export function resolveVoiceAudioUrl(fileId?: number, blobUrl?: string): string | undefined {
  if (blobUrl) return blobUrl;
  if (fileId) return `/api/assistant/files/${fileId}/download`;
  return undefined;
}

interface VoiceAudioPlayerProps {
  audioUrl: string;
  className?: string;
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s.toString().padStart(2, '0')}`;
}

export const VoiceAudioPlayer = memo(function VoiceAudioPlayer({
  audioUrl,
  className,
}: VoiceAudioPlayerProps) {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);

  useEffect(() => {
    const audio = new Audio(audioUrl);
    audioRef.current = audio;

    const onLoadedMetadata = () => setDuration(audio.duration);
    const onTimeUpdate = () => setCurrentTime(audio.currentTime);
    const onEnded = () => { setIsPlaying(false); setCurrentTime(0); };

    audio.addEventListener('loadedmetadata', onLoadedMetadata);
    audio.addEventListener('timeupdate', onTimeUpdate);
    audio.addEventListener('ended', onEnded);

    return () => {
      audio.removeEventListener('loadedmetadata', onLoadedMetadata);
      audio.removeEventListener('timeupdate', onTimeUpdate);
      audio.removeEventListener('ended', onEnded);
      audio.pause();
      audio.src = '';
    };
  }, [audioUrl]);

  const togglePlay = useCallback(() => {
    const audio = audioRef.current;
    if (!audio) return;
    if (isPlaying) {
      audio.pause();
      setIsPlaying(false);
    } else {
      audio.play();
      setIsPlaying(true);
    }
  }, [isPlaying]);

  const handleSeek = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const audio = audioRef.current;
    if (!audio) return;
    const time = parseFloat(e.target.value);
    audio.currentTime = time;
    setCurrentTime(time);
  }, []);

  const progress = duration > 0 ? (currentTime / duration) * 100 : 0;

  return (
    <div className={cn(
      "flex items-center gap-2.5 rounded-full bg-zinc-100 dark:bg-zinc-800 pl-1 pr-3 py-1",
      className,
    )}>
      <button
        type="button"
        onClick={togglePlay}
        className="flex items-center justify-center h-8 w-8 rounded-full bg-white dark:bg-zinc-700 shadow-sm hover:shadow transition-shadow"
        aria-label={isPlaying ? "Pause" : "Play"}
      >
        {isPlaying ? (
          <Pause className="h-3.5 w-3.5 text-zinc-700 dark:text-zinc-200" />
        ) : (
          <Play className="h-3.5 w-3.5 text-zinc-700 dark:text-zinc-200 ml-0.5" />
        )}
      </button>

      <span className="text-xs text-zinc-500 dark:text-zinc-400 tabular-nums min-w-[72px]">
        {formatTime(currentTime)} / {formatTime(duration)}
      </span>

      <div className="relative flex-1 min-w-[80px]">
        <div className="h-1 rounded-full bg-zinc-300 dark:bg-zinc-600">
          <div
            className="h-1 rounded-full bg-zinc-500 dark:bg-zinc-400 transition-all duration-100"
            style={{ width: `${progress}%` }}
          />
        </div>
        <input
          type="range"
          min={0}
          max={duration || 0}
          step={0.1}
          value={currentTime}
          onChange={handleSeek}
          className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
          aria-label="Seek"
        />
      </div>

      <Volume2 className="h-3.5 w-3.5 text-zinc-400 dark:text-zinc-500 flex-shrink-0" />
    </div>
  );
});
