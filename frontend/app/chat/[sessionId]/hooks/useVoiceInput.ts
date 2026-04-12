import { useCallback, useEffect, useRef, useState } from 'react';

const hasSpeechRecognition =
  typeof window !== 'undefined' &&
  ('SpeechRecognition' in window || 'webkitSpeechRecognition' in window);

function resolveLanguage(): string {
  const lang = (navigator.language ?? '').toLowerCase();
  if (lang.startsWith('zh')) return 'zh-CN';
  if (lang.startsWith('en')) return 'en-US';
  return 'zh-CN';
}

// Local type definitions for the Web Speech API (not present in all TS DOM libs).

interface SpeechRecognitionAlternative {
  readonly transcript: string;
  readonly confidence: number;
}

interface SpeechRecognitionResultItem {
  readonly isFinal: boolean;
  readonly [index: number]: SpeechRecognitionAlternative;
}

interface SpeechRecognitionResultList {
  readonly length: number;
  item(index: number): SpeechRecognitionResultItem;
  [index: number]: SpeechRecognitionResultItem;
}

interface SpeechRecognitionEvent extends Event {
  readonly resultIndex: number;
  readonly results: SpeechRecognitionResultList;
}

interface SpeechRecognitionErrorEvent extends Event {
  readonly error: string;
  readonly message: string;
}

interface SpeechRecognitionInstance {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  onresult: ((event: SpeechRecognitionEvent) => void) | null;
  onerror: ((event: SpeechRecognitionErrorEvent) => void) | null;
  onend: (() => void) | null;
  start(): void;
  stop(): void;
  abort(): void;
}

interface SpeechRecognitionConstructor {
  new (): SpeechRecognitionInstance;
}

interface UseVoiceInputOptions {
  /** Called with each confirmed transcript chunk. Appended to the current input value. */
  onTranscript: (text: string) => void;
  /** Block recording while a chat response is streaming. */
  disabled?: boolean;
}

interface UseVoiceInputResult {
  isRecording: boolean;
  isSupported: boolean;
  toggleRecording: () => void;
  voiceError: string | null;
}

export function useVoiceInput({
  onTranscript,
  disabled = false,
}: UseVoiceInputOptions): UseVoiceInputResult {
  const [isRecording, setIsRecording] = useState(false);
  const [voiceError, setVoiceError] = useState<string | null>(null);
  const recognitionRef = useRef<SpeechRecognitionInstance | null>(null);

  // Tear down recognition on unmount.
  useEffect(() => {
    return () => {
      recognitionRef.current?.abort();
    };
  }, []);

  const startRecording = useCallback(() => {
    if (!hasSpeechRecognition || disabled) return;
    setVoiceError(null);

    if (!recognitionRef.current) {
      const win = window as unknown as {
        SpeechRecognition?: SpeechRecognitionConstructor;
        webkitSpeechRecognition?: SpeechRecognitionConstructor;
      };
      const SpeechRecognitionImpl = win.SpeechRecognition ?? win.webkitSpeechRecognition;
      if (!SpeechRecognitionImpl) return;

      const rec = new SpeechRecognitionImpl();
      rec.continuous = true;      // Keep listening until explicitly stopped.
      rec.interimResults = false; // Only fire onresult for final segments.
      rec.lang = resolveLanguage();

      rec.onresult = (event: SpeechRecognitionEvent) => {
        const results = Array.from({ length: event.results.length }, (_, i) => event.results[i]);
        const transcript = results
          .slice(event.resultIndex)
          .filter((r) => r.isFinal)
          .map((r) => r[0].transcript)
          .join('');
        if (transcript) onTranscript(transcript);
      };

      rec.onerror = (event: SpeechRecognitionErrorEvent) => {
        // 'aborted' fires when we call stop() ourselves — not a real error.
        if (event.error === 'aborted') return;
        setIsRecording(false);
        const messages: Record<string, string> = {
          'not-allowed': '麦克风权限被拒绝，请在浏览器设置中允许访问。',
          'no-speech': '未检测到语音，请重试。',
          'network': '语音识别网络错误，请检查网络连接。',
        };
        setVoiceError(messages[event.error] ?? `语音识别错误：${event.error}`);
      };

      rec.onend = () => {
        // Fires after stop() or a silence timeout.
        setIsRecording(false);
      };

      recognitionRef.current = rec;
    }

    recognitionRef.current.start();
    setIsRecording(true);
  }, [disabled, onTranscript]);

  const stopRecording = useCallback(() => {
    recognitionRef.current?.stop();
    // isRecording is updated in onend to avoid races.
  }, []);

  const toggleRecording = useCallback(() => {
    if (isRecording) {
      stopRecording();
    } else {
      startRecording();
    }
  }, [isRecording, startRecording, stopRecording]);

  return {
    isRecording,
    isSupported: hasSpeechRecognition,
    toggleRecording,
    voiceError,
  };
}
