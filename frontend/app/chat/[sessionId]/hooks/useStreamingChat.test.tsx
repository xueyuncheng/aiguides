import { act, renderHook } from '@testing-library/react';
import type { SetStateAction } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Session } from '@/app/components/SessionSidebar';
import type { Message } from '../types';
import { useStreamingChat } from './useStreamingChat';

const createResponse = () => {
  const encoder = new TextEncoder();

  return new Response(new ReadableStream<Uint8Array>({
    start(controller) {
      controller.enqueue(encoder.encode('data: {"author":"assistant","content":"ok"}\n'));
      controller.close();
    },
  }), {
    status: 200,
    headers: { 'Content-Type': 'text/event-stream' },
  });
};

describe('useStreamingChat', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('keeps /chat for draft sessions until the first message is sent', async () => {
    const pushStateSpy = vi.spyOn(window.history, 'pushState');
    const authenticatedFetch = vi.fn().mockResolvedValue(createResponse());
    const loadSessions = vi.fn().mockResolvedValue([] as Session[]);
    const setMessages = vi.fn<(updater: SetStateAction<Message[]>) => void>();
    const setInputValue = vi.fn<(value: SetStateAction<string>) => void>();
    const clearImages = vi.fn();

    const { result } = renderHook(() => useStreamingChat({
      agentId: 'assistant',
      sessionId: '',
      setSessionId: vi.fn(),
      currentProjectId: 0,
      sessions: [],
      messages: [],
      userId: 'user-1',
      authenticatedFetch,
      clearImages,
      loadSessions,
      setMessages,
      setInputValue,
    }));

    expect(pushStateSpy).not.toHaveBeenCalled();

    await act(async () => {
      await result.current.sendMessage('hello', []);
    });

    expect(pushStateSpy).toHaveBeenCalledTimes(1);
    expect(pushStateSpy.mock.calls[0][2]).toMatch(/^\/chat\/session-/);
    expect(authenticatedFetch).toHaveBeenCalledTimes(1);
    expect(String(authenticatedFetch.mock.calls[0][0])).toMatch(/\/api\/assistant\/chats\/session-/);
  });
});
