'use client';

import { useState, useRef, useEffect, useCallback, memo } from 'react';
import { Button } from '@/app/components/ui/button';
import { Volume2, Square } from 'lucide-react';
import { useAuth } from '@/app/contexts/AuthContext';

interface TTSButtonProps {
  text: string;
  messageId: string;
  isStreaming?: boolean;
  /** Called with true when a TTS job starts (loading or playing), false when it ends. */
  onActiveChange?: (active: boolean) => void;
}

export const TTSButton = memo(({
  text,
  messageId,
  isStreaming,
  onActiveChange,
}: TTSButtonProps) => {
  const [isPlaying, setIsPlaying] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const audioRef = useRef<HTMLAudioElement>(null);
  // Queue of blob URLs waiting to be played.
  const queueRef = useRef<string[]>([]);
  // Whether the stream is still producing chunks.
  const streamingRef = useRef(false);
  // Object URLs that need to be revoked on cleanup.
  const allObjectURLs = useRef<string[]>([]);
  // AbortController to cancel the fetch when the user stops.
  const abortRef = useRef<AbortController | null>(null);

  const { authenticatedFetch } = useAuth();

  // Revoke all object URLs on unmount.
  useEffect(() => {
    return () => {
      allObjectURLs.current.forEach((url) => URL.revokeObjectURL(url));
    };
  }, []);

  // Play the next chunk in the queue, if any.
  const playNext = useCallback(() => {
    if (!audioRef.current) return;

    const next = queueRef.current.shift();
    if (next) {
      audioRef.current.src = next;
      audioRef.current.play().catch((err) => {
        console.error('TTS playback error:', err);
      });
    } else if (!streamingRef.current) {
      // Queue is empty and stream is done — playback complete.
      setIsPlaying(false);
      onActiveChange?.(false);
    }
    // If queue is empty but stream is still going, `handleEnded` will be called
    // again once more chunks arrive and get pushed to the queue.
  }, [onActiveChange]);

  const stop = useCallback(() => {
    // Cancel the in-flight SSE request.
    abortRef.current?.abort();
    abortRef.current = null;
    streamingRef.current = false;

    // Revoke queued URLs.
    queueRef.current.forEach((url) => URL.revokeObjectURL(url));
    queueRef.current = [];

    if (audioRef.current) {
      audioRef.current.pause();
      audioRef.current.src = '';
    }

    setIsPlaying(false);
    setIsLoading(false);
    onActiveChange?.(false);
  }, [onActiveChange]);

  const handlePlayAudio = async () => {
    if (isPlaying || isLoading) {
      stop();
      return;
    }

    // Unlock autoplay within the user-gesture window before any async work.
    if (audioRef.current) {
      audioRef.current.play().catch(() => {});
      audioRef.current.pause();
    }

    setIsLoading(true);
    setError(null);
    onActiveChange?.(true);

    const controller = new AbortController();
    abortRef.current = controller;
    streamingRef.current = true;
    queueRef.current = [];

    try {
      const response = await authenticatedFetch('/api/assistant/tts/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text, voice_name: 'Kore' }),
        signal: controller.signal,
      } as RequestInit);

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `TTS request failed (${response.status})`);
      }

      if (!response.body) {
        throw new Error('No response body');
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      let firstChunk = true;

      // eslint-disable-next-line no-constant-condition
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });

        // SSE events are separated by double newlines.
        const parts = buffer.split('\n\n');
        buffer = parts.pop() ?? '';

        for (const part of parts) {
          const lines = part.split('\n');
          let eventType = 'message';
          let dataLine = '';

          for (const line of lines) {
            if (line.startsWith('event:')) {
              eventType = line.slice(6).trim();
            } else if (line.startsWith('data:')) {
              dataLine = line.slice(5).trim();
            }
          }

          if (!dataLine) continue;

          let payload: Record<string, unknown>;
          try {
            payload = JSON.parse(dataLine);
          } catch {
            continue;
          }

          if (eventType === 'error') {
            throw new Error(String(payload.error ?? 'TTS stream error'));
          }

          if (eventType === 'done') {
            streamingRef.current = false;
            // If the audio element has already finished the last queued chunk,
            // it will be idle now — trigger a final check.
            if (!audioRef.current?.src || audioRef.current.ended || audioRef.current.paused) {
              playNext();
            }
            break;
          }

          if (eventType === 'chunk' && typeof payload.data === 'string') {
            // Decode base64 WAV.
            const binary = atob(payload.data);
            const bytes = new Uint8Array(binary.length);
            for (let i = 0; i < binary.length; i++) {
              bytes[i] = binary.charCodeAt(i);
            }
            const blob = new Blob([bytes], { type: 'audio/wav' });
            const objectURL = URL.createObjectURL(blob);
            allObjectURLs.current.push(objectURL);

            if (firstChunk) {
              firstChunk = false;
              setIsLoading(false);
              setIsPlaying(true);
              // Play immediately.
              if (audioRef.current) {
                audioRef.current.src = objectURL;
                audioRef.current.play().catch((err) => {
                  console.error('TTS first-chunk playback error:', err);
                });
              }
            } else {
              queueRef.current.push(objectURL);
              // If the audio element finished a previous chunk while the queue
              // was empty (slow network), it will be paused/ended with a stale
              // src. Resume immediately now that a new chunk has arrived.
              if (audioRef.current && (audioRef.current.ended || audioRef.current.paused)) {
                playNext();
              }
            }
          }
        }
      }
    } catch (err) {
      if ((err as Error).name === 'AbortError') return; // user stopped
      const errorMessage = err instanceof Error ? err.message : 'Unknown error';
      console.error('TTS error:', errorMessage);
      setError(errorMessage);
      setIsPlaying(false);
      setIsLoading(false);
      onActiveChange?.(false);
    } finally {
      streamingRef.current = false;
    }
  };

  // When a chunk finishes playing, move to the next one.
  const handleEnded = useCallback(() => {
    if (queueRef.current.length > 0) {
      playNext();
    } else if (!streamingRef.current) {
      // No more chunks and stream is done.
      setIsPlaying(false);
      onActiveChange?.(false);
    }
    // Otherwise the stream is still running; wait for next chunk to be pushed.
  }, [playNext, onActiveChange]);

  const isDisabled = isStreaming || !text?.trim();

  return (
    <>
      <Button
        size="sm"
        variant="ghost"
        onClick={handlePlayAudio}
        disabled={isDisabled}
        className="h-6 w-6 p-0 hover:bg-secondary text-muted-foreground hover:text-foreground"
        title={isPlaying || isLoading ? 'Stop audio' : error ? 'TTS error' : 'Play audio'}
        aria-label={isPlaying || isLoading ? 'Stop audio' : 'Play audio'}
      >
        {isPlaying || isLoading ? (
          <Square className="h-3.5 w-3.5" />
        ) : (
          <Volume2 className={`h-3.5 w-3.5 ${error ? 'text-red-500' : ''}`} />
        )}
      </Button>
      <audio ref={audioRef} onEnded={handleEnded} />
    </>
  );
});

TTSButton.displayName = 'TTSButton';
