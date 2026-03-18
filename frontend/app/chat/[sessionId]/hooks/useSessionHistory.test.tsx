import { createRef } from 'react';
import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { SCROLL_RESET_DELAY } from '../constants';
import { useSessionHistory } from './useSessionHistory';

const createHistoryResponse = (overrides?: Partial<{ total: number; has_more: boolean }>) => ({
  messages: [
    {
      id: 'message-1',
      role: 'assistant' as const,
      content: 'hello',
      timestamp: '2026-03-19T00:00:00Z',
      images: ['img-1'],
      file_names: ['file-1'],
      tool_calls: [
        {
          tool_name: 'web_search',
          label: 'Search',
          args: { q: 'hello' },
        },
      ],
    },
  ],
  total: overrides?.total ?? 1,
  has_more: overrides?.has_more ?? false,
});

describe('useSessionHistory', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('loads session history and clears instant scroll mode after the reset delay', async () => {
    const authenticatedFetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(createHistoryResponse({ total: 3, has_more: true })), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    );
    const clearImages = vi.fn();
    const onSessionChangeStart = vi.fn();
    const setSessionId = vi.fn();
    const pushStateSpy = vi.spyOn(window.history, 'pushState');

    const { result } = renderHook(() => useSessionHistory({
      agentId: 'assistant',
      userId: 'user-1',
      sessionId: 'session-1',
      urlSessionId: '',
      authenticatedFetch,
      clearImages,
      onSessionChangeStart,
      scrollContainerRef: createRef<HTMLDivElement>(),
      setSessionId,
    }));

    await act(async () => {
      await result.current.loadSessionHistory('session-2');
    });

    expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/chat/session-2');
    expect(setSessionId).toHaveBeenCalledWith('session-2');
    expect(clearImages).toHaveBeenCalled();
    expect(onSessionChangeStart).toHaveBeenCalled();
    expect(result.current.messages).toHaveLength(1);
    expect(result.current.messages[0]).toMatchObject({
      id: 'message-1',
      content: 'hello',
      images: ['img-1'],
      fileNames: ['file-1'],
      toolCalls: [
        {
          toolName: 'web_search',
          label: 'Search',
          args: { q: 'hello' },
        },
      ],
    });
    expect(result.current.hasMoreMessages).toBe(true);
    expect(result.current.totalMessageCount).toBe(3);
    expect(result.current.shouldScrollInstantly).toBe(true);

    await act(async () => {
      vi.advanceTimersByTime(SCROLL_RESET_DELAY);
    });

    expect(result.current.shouldScrollInstantly).toBe(false);
  });

  it('prepends older messages and restores scroll position', async () => {
    let scrollHeight = 200;
    const container = document.createElement('div');
    Object.defineProperty(container, 'scrollHeight', {
      get: () => scrollHeight,
      configurable: true,
    });
    Object.defineProperty(container, 'scrollTop', {
      value: 0,
      writable: true,
      configurable: true,
    });

    const scrollContainerRef = createRef<HTMLDivElement>();
    scrollContainerRef.current = container;

    const authenticatedFetch = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify(createHistoryResponse({ total: 3, has_more: true })), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({
          messages: [
            {
              id: 'message-0',
              role: 'user',
              content: 'older',
              timestamp: '2026-03-18T00:00:00Z',
            },
          ],
          total: 3,
          has_more: false,
        }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      );

    const { result } = renderHook(() => useSessionHistory({
      agentId: 'assistant',
      userId: 'user-1',
      sessionId: 'session-1',
      urlSessionId: '',
      authenticatedFetch,
      clearImages: vi.fn(),
      scrollContainerRef,
      setSessionId: vi.fn(),
    }));

    await act(async () => {
      await result.current.loadSessionHistory('session-1', false);
    });

    await act(async () => {
      await result.current.loadOlderMessages();
      scrollHeight = 320;
      await vi.runAllTimersAsync();
    });

    expect(result.current.messages.map((message) => message.id)).toEqual(['message-0', 'message-1']);
    expect(result.current.hasMoreMessages).toBe(false);
    expect(container.scrollTop).toBe(120);
  });
});
