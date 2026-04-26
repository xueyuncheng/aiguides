'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

export type VoiceCallStatus = 'idle' | 'connecting' | 'connected' | 'error';

export interface VoiceMessage {
  id: string;
  role: 'user' | 'assistant';
  audioUrl: string | null;
  transcript: string;
  isComplete: boolean;
}

interface UseVoiceCallResult {
  status: VoiceCallStatus;
  voiceMessages: VoiceMessage[];
  startCall: (overrideSessionId?: string) => Promise<void>;
  endCall: () => void;
  error: string | null;
}

const INPUT_SAMPLE_RATE = 16000;
const OUTPUT_SAMPLE_RATE = 24000;

let nextMsgId = 0;
function genId(): string {
  return `vm-${Date.now()}-${nextMsgId++}`;
}

function getBackendWsUrl(): string {
  if (process.env.NEXT_PUBLIC_BACKEND_WS_URL) {
    return process.env.NEXT_PUBLIC_BACKEND_WS_URL;
  }
  if (typeof window !== 'undefined') {
    const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
    return `${proto}://${window.location.hostname}:8080`;
  }
  return 'ws://localhost:8080';
}

function pcmToWavBlob(int16Chunks: Int16Array[], sampleRate: number): Blob {
  let totalLen = 0;
  for (const chunk of int16Chunks) totalLen += chunk.length;

  const pcm = new Int16Array(totalLen);
  let offset = 0;
  for (const chunk of int16Chunks) {
    pcm.set(chunk, offset);
    offset += chunk.length;
  }

  const buffer = new ArrayBuffer(44 + pcm.length * 2);
  const view = new DataView(buffer);
  const numChannels = 1;
  const bitsPerSample = 16;
  const byteRate = sampleRate * numChannels * (bitsPerSample / 8);
  const blockAlign = numChannels * (bitsPerSample / 8);
  const dataSize = pcm.length * 2;

  writeString(view, 0, 'RIFF');
  view.setUint32(4, 36 + dataSize, true);
  writeString(view, 8, 'WAVE');
  writeString(view, 12, 'fmt ');
  view.setUint32(16, 16, true);
  view.setUint16(20, 1, true);
  view.setUint16(22, numChannels, true);
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, byteRate, true);
  view.setUint16(32, blockAlign, true);
  view.setUint16(34, bitsPerSample, true);
  writeString(view, 36, 'data');
  view.setUint32(40, dataSize, true);

  const pcmBytes = new Uint8Array(pcm.buffer);
  new Uint8Array(buffer).set(pcmBytes, 44);

  return new Blob([buffer], { type: 'audio/wav' });
}

function writeString(view: DataView, offset: number, str: string) {
  for (let i = 0; i < str.length; i++) {
    view.setUint8(offset + i, str.charCodeAt(i));
  }
}

function base64ToPcmInt16(b64: string): Int16Array {
  const binary = atob(b64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return new Int16Array(bytes.buffer);
}

interface TurnRefs {
  turnPhase: React.MutableRefObject<'waiting' | 'user' | 'model'>;
  userAudioChunks: React.MutableRefObject<Int16Array[]>;
  modelAudioChunks: React.MutableRefObject<Int16Array[]>;
  currentUserMsgId: React.MutableRefObject<string | null>;
  currentModelMsgId: React.MutableRefObject<string | null>;
}

export function useVoiceCall(sessionId?: string): UseVoiceCallResult {
  const [status, setStatus] = useState<VoiceCallStatus>('idle');
  const [voiceMessages, setVoiceMessages] = useState<VoiceMessage[]>([]);
  const [error, setError] = useState<string | null>(null);

  const prevSessionIdRef = useRef(sessionId);
  useEffect(() => {
    if (prevSessionIdRef.current !== sessionId) {
      prevSessionIdRef.current = sessionId;
      setVoiceMessages([]);
    }
  }, [sessionId]);

  const wsRef = useRef<WebSocket | null>(null);
  const audioCtxRef = useRef<AudioContext | null>(null);
  const sourceNodeRef = useRef<MediaStreamAudioSourceNode | null>(null);
  const processorRef = useRef<ScriptProcessorNode | null>(null);
  const streamRef = useRef<MediaStream | null>(null);

  const playbackCtxRef = useRef<AudioContext | null>(null);
  const nextPlayTimeRef = useRef(0);

  const turnPhaseRef = useRef<'waiting' | 'user' | 'model'>('waiting');
  const userAudioChunksRef = useRef<Int16Array[]>([]);
  const modelAudioChunksRef = useRef<Int16Array[]>([]);
  const currentUserMsgIdRef = useRef<string | null>(null);
  const currentModelMsgIdRef = useRef<string | null>(null);

  const turnRefs: TurnRefs = {
    turnPhase: turnPhaseRef,
    userAudioChunks: userAudioChunksRef,
    modelAudioChunks: modelAudioChunksRef,
    currentUserMsgId: currentUserMsgIdRef,
    currentModelMsgId: currentModelMsgIdRef,
  };

  const cleanup = useCallback(() => {
    processorRef.current?.disconnect();
    processorRef.current = null;
    sourceNodeRef.current?.disconnect();
    sourceNodeRef.current = null;
    streamRef.current?.getTracks().forEach((t) => t.stop());
    streamRef.current = null;
    audioCtxRef.current?.close();
    audioCtxRef.current = null;

    if (wsRef.current) {
      wsRef.current.onclose = null;
      wsRef.current.onerror = null;
      wsRef.current.onmessage = null;
      wsRef.current.close();
      wsRef.current = null;
    }

    playbackCtxRef.current?.close();
    playbackCtxRef.current = null;
    nextPlayTimeRef.current = 0;

    turnPhaseRef.current = 'waiting';
    userAudioChunksRef.current = [];
    modelAudioChunksRef.current = [];
    currentUserMsgIdRef.current = null;
    currentModelMsgIdRef.current = null;

    setStatus('idle');
  }, []);

  useEffect(() => () => cleanup(), [cleanup]);

  const finalizeTurn = useCallback((
    msgIdRef: React.MutableRefObject<string | null>,
    chunksRef: React.MutableRefObject<Int16Array[]>,
    sampleRate: number,
  ) => {
    const msgId = msgIdRef.current;
    const chunks = chunksRef.current;
    if (msgId) {
      if (chunks.length > 0) {
        const blob = pcmToWavBlob(chunks, sampleRate);
        const url = URL.createObjectURL(blob);
        setVoiceMessages((prev) =>
          prev.map((m) => (m.id === msgId ? { ...m, audioUrl: url, isComplete: true } : m))
        );
      } else {
        setVoiceMessages((prev) =>
          prev.map((m) => (m.id === msgId ? { ...m, isComplete: true } : m))
        );
      }
    }
    chunksRef.current = [];
    msgIdRef.current = null;
  }, []);

  const finalizeUserTurn = useCallback(() => {
    finalizeTurn(currentUserMsgIdRef, userAudioChunksRef, INPUT_SAMPLE_RATE);
  }, [finalizeTurn]);

  const finalizeModelTurn = useCallback(() => {
    finalizeTurn(currentModelMsgIdRef, modelAudioChunksRef, OUTPUT_SAMPLE_RATE);
  }, [finalizeTurn]);

  const playPCMChunk = useCallback((pcmData: Int16Array) => {
    if (!playbackCtxRef.current) {
      playbackCtxRef.current = new AudioContext({ sampleRate: OUTPUT_SAMPLE_RATE });
      nextPlayTimeRef.current = playbackCtxRef.current.currentTime;
    }
    const ctx = playbackCtxRef.current;

    const float32 = new Float32Array(pcmData.length);
    for (let i = 0; i < pcmData.length; i++) {
      float32[i] = pcmData[i] / 32768.0;
    }

    const audioBuffer = ctx.createBuffer(1, float32.length, OUTPUT_SAMPLE_RATE);
    audioBuffer.copyToChannel(float32, 0);

    const source = ctx.createBufferSource();
    source.buffer = audioBuffer;
    source.connect(ctx.destination);

    const startTime = Math.max(nextPlayTimeRef.current, ctx.currentTime);
    source.start(startTime);
    nextPlayTimeRef.current = startTime + audioBuffer.duration;
  }, []);

  const resetPlayback = useCallback(() => {
    playbackCtxRef.current?.close();
    playbackCtxRef.current = null;
    nextPlayTimeRef.current = 0;
  }, []);

  const startCapture = useCallback((stream: MediaStream, ws: WebSocket) => {
    const audioCtx = new AudioContext({ sampleRate: INPUT_SAMPLE_RATE });
    audioCtxRef.current = audioCtx;

    const source = audioCtx.createMediaStreamSource(stream);
    sourceNodeRef.current = source;

    const processor = audioCtx.createScriptProcessor(4096, 1, 1);
    processorRef.current = processor;

    processor.onaudioprocess = (e) => {
      if (ws.readyState !== WebSocket.OPEN) return;

      const float32 = e.inputBuffer.getChannelData(0);
      const int16 = new Int16Array(float32.length);
      for (let i = 0; i < float32.length; i++) {
        const s = Math.max(-1, Math.min(1, float32[i]));
        int16[i] = s < 0 ? s * 32768 : s * 32767;
      }

      userAudioChunksRef.current.push(int16.slice());

      const bytes = new Uint8Array(int16.buffer);
      const chars = new Array<string>(bytes.length);
      for (let i = 0; i < bytes.length; i++) chars[i] = String.fromCharCode(bytes[i]);
      ws.send(JSON.stringify({ type: 'audio', data: btoa(chars.join('')) }));
    };

    source.connect(processor);
    processor.connect(audioCtx.destination);
  }, []);

  const handleWsMessage = useCallback((event: MessageEvent, stream: MediaStream, ws: WebSocket) => {
    let msg: { type: string; data?: string; finished?: boolean };
    try {
      msg = JSON.parse(event.data as string);
    } catch {
      return;
    }

    switch (msg.type) {
      case 'setup_ok':
        setStatus('connected');
        startCapture(stream, ws);
        break;

      case 'input_transcript':
        if (turnPhaseRef.current === 'model') {
          finalizeModelTurn();
        }
        handleInputTranscript(msg.data, turnRefs, setVoiceMessages);
        break;

      case 'audio':
        if (turnPhaseRef.current === 'user') {
          finalizeUserTurn();
        }
        handleAudioMessage(msg.data, turnRefs, setVoiceMessages, playPCMChunk);
        break;

      case 'output_transcript':
        handleOutputTranscript(msg.data, turnRefs, setVoiceMessages);
        break;

      case 'turn_complete':
        finalizeModelTurn();
        turnPhaseRef.current = 'waiting';
        userAudioChunksRef.current = [];
        break;

      case 'interrupted':
        resetPlayback();
        finalizeModelTurn();
        turnPhaseRef.current = 'waiting';
        userAudioChunksRef.current = [];
        break;

      case 'go_away':
        cleanup();
        break;

      case 'error':
        setError(msg.data ?? 'Voice call error');
        setStatus('error');
        cleanup();
        break;
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cleanup, finalizeModelTurn, finalizeUserTurn, playPCMChunk, resetPlayback, startCapture]);

  const startCall = useCallback(async (overrideSessionId?: string) => {
    if (status !== 'idle') return;
    setError(null);
    setVoiceMessages([]);
    setStatus('connecting');

    let stream: MediaStream;
    try {
      stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          channelCount: 1,
          sampleRate: INPUT_SAMPLE_RATE,
          echoCancellation: true,
          noiseSuppression: true,
        },
      });
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Microphone access denied';
      setError(msg);
      setStatus('error');
      return;
    }
    streamRef.current = stream;

    const resolvedSessionId = overrideSessionId || sessionId;
    const wsUrl = `${getBackendWsUrl()}/api/assistant/live${resolvedSessionId ? `?session_id=${encodeURIComponent(resolvedSessionId)}` : ''}`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onmessage = (ev) => handleWsMessage(ev, stream, ws);

    ws.onerror = () => {
      setError('WebSocket connection error');
      setStatus('error');
      cleanup();
    };

    ws.onclose = () => {
      cleanup();
    };
  }, [cleanup, handleWsMessage, sessionId, status]);

  const endCall = useCallback(() => {
    if (userAudioChunksRef.current.length > 0 || currentUserMsgIdRef.current) {
      finalizeUserTurn();
    }
    if (turnPhaseRef.current === 'model') finalizeModelTurn();

    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'end' }));
    }
    cleanup();
  }, [cleanup, finalizeModelTurn, finalizeUserTurn]);

  return { status, voiceMessages, startCall, endCall, error };
}

function handleInputTranscript(
  data: string | undefined,
  refs: TurnRefs,
  setMessages: React.Dispatch<React.SetStateAction<VoiceMessage[]>>,
) {
  if (!data) return;
  if (refs.turnPhase.current !== 'user') {
    refs.turnPhase.current = 'user';
    const id = genId();
    refs.currentUserMsgId.current = id;
    setMessages((prev) => [
      ...prev,
      { id, role: 'user', audioUrl: null, transcript: '', isComplete: false },
    ]);
  }
  const userMsgId = refs.currentUserMsgId.current;
  if (userMsgId) {
    setMessages((prev) =>
      prev.map((m) =>
        m.id === userMsgId ? { ...m, transcript: m.transcript + data } : m
      )
    );
  }
}

function handleAudioMessage(
  data: string | undefined,
  refs: TurnRefs,
  setMessages: React.Dispatch<React.SetStateAction<VoiceMessage[]>>,
  playPCMChunk: (pcm: Int16Array) => void,
) {
  if (!data) return;
  if (refs.turnPhase.current !== 'model') {
    refs.turnPhase.current = 'model';
    const id = genId();
    refs.currentModelMsgId.current = id;
    refs.modelAudioChunks.current = [];
    setMessages((prev) => [
      ...prev,
      { id, role: 'assistant', audioUrl: null, transcript: '', isComplete: false },
    ]);
  }
  const pcm = base64ToPcmInt16(data);
  refs.modelAudioChunks.current.push(pcm);
  playPCMChunk(pcm);
}

function handleOutputTranscript(
  data: string | undefined,
  refs: TurnRefs,
  setMessages: React.Dispatch<React.SetStateAction<VoiceMessage[]>>,
) {
  if (!data) return;
  const modelMsgId = refs.currentModelMsgId.current;
  if (modelMsgId) {
    setMessages((prev) =>
      prev.map((m) =>
        m.id === modelMsgId ? { ...m, transcript: m.transcript + data } : m
      )
    );
  }
}
