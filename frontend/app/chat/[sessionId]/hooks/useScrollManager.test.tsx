import { createRef } from 'react';
import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Message } from '../types';
import { useScrollManager } from './useScrollManager';

const createMessage = (overrides: Partial<Message>): Message => ({
  id: overrides.id ?? 'message-1',
  role: overrides.role ?? 'assistant',
  content: overrides.content ?? '',
  timestamp: overrides.timestamp ?? new Date('2026-04-05T00:00:00Z'),
  ...overrides,
});

describe('useScrollManager', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-04-05T00:00:00Z'));

    class ResizeObserverMock {
      callback: ResizeObserverCallback;

      constructor(callback: ResizeObserverCallback) {
        this.callback = callback;
      }

      observe = vi.fn();
      disconnect = vi.fn();
      unobserve = vi.fn();
    }

    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('scrolls to the loading area when a user message is waiting for the assistant response', () => {
    const scrollTo = vi.fn();
    const scrollContainer = document.createElement('div');
    scrollContainer.scrollTo = scrollTo;

    Object.defineProperty(scrollContainer, 'clientHeight', {
      value: 600,
      configurable: true,
    });
    Object.defineProperty(scrollContainer, 'scrollHeight', {
      value: 1200,
      configurable: true,
    });
    Object.defineProperty(scrollContainer, 'scrollTop', {
      value: 0,
      writable: true,
      configurable: true,
    });

    scrollContainer.getBoundingClientRect = vi.fn(() => ({
      top: 0,
      bottom: 600,
      left: 0,
      right: 0,
      width: 0,
      height: 600,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    }));

    const latestUserElement = document.createElement('div');
    latestUserElement.getBoundingClientRect = vi.fn(() => ({
      top: 640,
      bottom: 700,
      left: 0,
      right: 0,
      width: 0,
      height: 60,
      x: 0,
      y: 640,
      toJSON: () => ({}),
    }));

    const messagesEndElement = document.createElement('div');
    messagesEndElement.getBoundingClientRect = vi.fn(() => ({
      top: 700,
      bottom: 720,
      left: 0,
      right: 0,
      width: 0,
      height: 20,
      x: 0,
      y: 700,
      toJSON: () => ({}),
    }));

    const scrollContainerRef = createRef<HTMLDivElement>();
    scrollContainerRef.current = scrollContainer;

    const { result, rerender } = renderHook(
      ({ messages, isLoading, isStreamingResponse }: { messages: Message[]; isLoading: boolean; isStreamingResponse: boolean }) =>
        useScrollManager({
          messages,
          isLoading,
          isStreamingResponse,
          latestUserMessageId: 'user-1',
          isLoadingHistory: false,
          inputValue: '',
          shouldScrollInstantly: false,
          shouldLoadOlderMessages: () => false,
          loadOlderMessages: vi.fn(),
          textareaRef: createRef<HTMLTextAreaElement>(),
          scrollContainerRef,
        }),
      {
        initialProps: {
          messages: [],
          isLoading: false,
          isStreamingResponse: false,
        },
      }
    );

    result.current.latestUserMessageRef.current = latestUserElement;
    result.current.messagesEndRef.current = messagesEndElement;

    rerender({
      messages: [createMessage({ id: 'user-1', role: 'user', content: 'hello' })],
      isLoading: false,
      isStreamingResponse: false,
    });

    expect(scrollTo).toHaveBeenLastCalledWith({
      top: 292,
      behavior: 'smooth',
    });

    rerender({
      messages: [createMessage({ id: 'user-1', role: 'user', content: 'hello' })],
      isLoading: true,
      isStreamingResponse: false,
    });

    expect(scrollTo).toHaveBeenLastCalledWith({
      top: 312,
      behavior: 'smooth',
    });
    expect(scrollTo).toHaveBeenCalledTimes(2);
  });

  it('keeps auto-scrolling when streaming starts after the loading area is shown', () => {
    const scrollTo = vi.fn();
    const scrollContainer = document.createElement('div');
    scrollContainer.scrollTo = scrollTo;

    Object.defineProperty(scrollContainer, 'clientHeight', {
      value: 600,
      configurable: true,
    });
    Object.defineProperty(scrollContainer, 'scrollHeight', {
      value: 1200,
      configurable: true,
    });
    Object.defineProperty(scrollContainer, 'scrollTop', {
      value: 0,
      writable: true,
      configurable: true,
    });

    scrollContainer.getBoundingClientRect = vi.fn(() => ({
      top: 0,
      bottom: 600,
      left: 0,
      right: 0,
      width: 0,
      height: 600,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    }));

    const messagesEndElement = document.createElement('div');
    messagesEndElement.getBoundingClientRect = vi.fn(() => ({
      top: 700,
      bottom: 720,
      left: 0,
      right: 0,
      width: 0,
      height: 20,
      x: 0,
      y: 700,
      toJSON: () => ({}),
    }));

    const scrollContainerRef = createRef<HTMLDivElement>();
    scrollContainerRef.current = scrollContainer;

    const { result, rerender } = renderHook(
      ({ messages, isLoading, isStreamingResponse }: { messages: Message[]; isLoading: boolean; isStreamingResponse: boolean }) =>
        useScrollManager({
          messages,
          isLoading,
          isStreamingResponse,
          latestUserMessageId: 'user-1',
          isLoadingHistory: false,
          inputValue: '',
          shouldScrollInstantly: false,
          shouldLoadOlderMessages: () => false,
          loadOlderMessages: vi.fn(),
          textareaRef: createRef<HTMLTextAreaElement>(),
          scrollContainerRef,
        }),
      {
        initialProps: {
          messages: [createMessage({ id: 'user-1', role: 'user', content: 'hello' })],
          isLoading: true,
          isStreamingResponse: false,
        },
      }
    );

    result.current.messagesEndRef.current = messagesEndElement;

    rerender({
      messages: [
        createMessage({ id: 'user-1', role: 'user', content: 'hello' }),
        createMessage({ id: 'assistant-1', role: 'assistant', content: 'hi', isStreaming: true }),
      ],
      isLoading: true,
      isStreamingResponse: true,
    });

    expect(scrollTo).toHaveBeenLastCalledWith({
      top: 312,
      behavior: 'auto',
    });
  });

  it('does not treat instant auto-scroll after history load as manual scrolling', () => {
    const scrollTo = vi.fn();
    const scrollContainer = document.createElement('div');
    scrollContainer.scrollTo = scrollTo;

    Object.defineProperty(scrollContainer, 'clientHeight', {
      value: 600,
      configurable: true,
    });
    Object.defineProperty(scrollContainer, 'scrollHeight', {
      value: 1200,
      configurable: true,
    });
    Object.defineProperty(scrollContainer, 'scrollTop', {
      value: 0,
      writable: true,
      configurable: true,
    });

    scrollContainer.getBoundingClientRect = vi.fn(() => ({
      top: 0,
      bottom: 600,
      left: 0,
      right: 0,
      width: 0,
      height: 600,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    }));

    const messagesEndElement = document.createElement('div');
    messagesEndElement.getBoundingClientRect = vi.fn(() => ({
      top: 900,
      bottom: 901,
      left: 0,
      right: 0,
      width: 0,
      height: 1,
      x: 0,
      y: 900,
      toJSON: () => ({}),
    }));

    const scrollContainerRef = createRef<HTMLDivElement>();
    scrollContainerRef.current = scrollContainer;

    const { result, rerender } = renderHook(() =>
      useScrollManager({
        messages: [createMessage({ id: 'assistant-1', role: 'assistant', content: 'done' })],
        isLoading: false,
        isStreamingResponse: false,
        latestUserMessageId: undefined,
        isLoadingHistory: false,
        inputValue: '',
        shouldScrollInstantly: true,
        shouldLoadOlderMessages: () => false,
        loadOlderMessages: vi.fn(),
        textareaRef: createRef<HTMLTextAreaElement>(),
        scrollContainerRef,
      })
    );

    result.current.messagesEndRef.current = messagesEndElement;

    rerender();

    expect(scrollTo).toHaveBeenLastCalledWith({
      top: 493,
      behavior: 'auto',
    });

    act(() => {
      scrollContainer.scrollTop = 493;
      result.current.handleScroll();
    });

    expect(result.current.isAtBottomRef.current).toBe(true);
  });

  it('scrolls after history loading finishes and the message end marker is mounted', () => {
    const scrollTo = vi.fn();
    const scrollContainer = document.createElement('div');
    scrollContainer.scrollTo = scrollTo;

    Object.defineProperty(scrollContainer, 'clientHeight', {
      value: 600,
      configurable: true,
    });
    Object.defineProperty(scrollContainer, 'scrollHeight', {
      value: 1200,
      configurable: true,
    });
    Object.defineProperty(scrollContainer, 'scrollTop', {
      value: 0,
      writable: true,
      configurable: true,
    });

    scrollContainer.getBoundingClientRect = vi.fn(() => ({
      top: 0,
      bottom: 600,
      left: 0,
      right: 0,
      width: 0,
      height: 600,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    }));

    const messagesEndElement = document.createElement('div');
    messagesEndElement.getBoundingClientRect = vi.fn(() => ({
      top: 900,
      bottom: 901,
      left: 0,
      right: 0,
      width: 0,
      height: 1,
      x: 0,
      y: 900,
      toJSON: () => ({}),
    }));

    const scrollContainerRef = createRef<HTMLDivElement>();
    scrollContainerRef.current = scrollContainer;

    const { result, rerender } = renderHook(
      ({ isLoadingHistory }: { isLoadingHistory: boolean }) =>
        useScrollManager({
          messages: [createMessage({ id: 'assistant-1', role: 'assistant', content: 'done' })],
          isLoading: false,
          isStreamingResponse: false,
          latestUserMessageId: undefined,
          isLoadingHistory,
          inputValue: '',
          shouldScrollInstantly: true,
          shouldLoadOlderMessages: () => false,
          loadOlderMessages: vi.fn(),
          textareaRef: createRef<HTMLTextAreaElement>(),
          scrollContainerRef,
        }),
      {
        initialProps: {
          isLoadingHistory: true,
        },
      }
    );

    expect(scrollTo).not.toHaveBeenCalled();

    result.current.messagesEndRef.current = messagesEndElement;

    rerender({ isLoadingHistory: false });

    expect(scrollTo).toHaveBeenLastCalledWith({
      top: 493,
      behavior: 'auto',
    });
  });

  it('re-observes the active chat input container after the layout switches', () => {
    const firstInput = document.createElement('div');
    const secondInput = document.createElement('div');

    firstInput.getBoundingClientRect = vi.fn(() => ({
      top: 0,
      bottom: 100,
      left: 0,
      right: 0,
      width: 0,
      height: 100,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    }));

    secondInput.getBoundingClientRect = vi.fn(() => ({
      top: 0,
      bottom: 72,
      left: 0,
      right: 0,
      width: 0,
      height: 72,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    }));

    const { result, rerender } = renderHook(() =>
      useScrollManager({
        messages: [],
        isLoading: false,
        isStreamingResponse: false,
        latestUserMessageId: undefined,
        isLoadingHistory: false,
        inputValue: '',
        shouldScrollInstantly: false,
        shouldLoadOlderMessages: () => false,
        loadOlderMessages: vi.fn(),
        textareaRef: createRef<HTMLTextAreaElement>(),
        scrollContainerRef: createRef<HTMLDivElement>(),
      })
    );

    act(() => {
      result.current.chatInputContainerRef(firstInput);
    });

    rerender();

    expect(result.current.chatInputOffset).toBe(108);

    act(() => {
      result.current.chatInputContainerRef(secondInput);
    });

    rerender();

    expect(result.current.chatInputOffset).toBe(80);
  });
});
