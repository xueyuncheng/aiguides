'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import type { RefObject } from 'react';
import { LOAD_MORE_THRESHOLD, MESSAGES_PER_PAGE, SCROLL_RESET_DELAY } from '../constants';
import type { Message, SessionHistoryResponse } from '../types';
import { mapHistoryMessage } from '../utils/messages';

type AuthenticatedFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

interface UseSessionHistoryParams {
  agentId: string;
  userId?: string;
  sessionId: string;
  urlSessionId: string;
  authenticatedFetch: AuthenticatedFetch;
  clearImages: () => void;
  onSessionChangeStart?: () => void;
  scrollContainerRef: RefObject<HTMLDivElement | null>;
  setSessionId: React.Dispatch<React.SetStateAction<string>>;
}

export function useSessionHistory({
  agentId,
  userId,
  sessionId,
  urlSessionId,
  authenticatedFetch,
  clearImages,
  onSessionChangeStart,
  scrollContainerRef,
  setSessionId,
}: UseSessionHistoryParams) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoadingHistory, setIsLoadingHistory] = useState(false);
  const [isLoadingOlderMessages, setIsLoadingOlderMessages] = useState(false);
  const [hasMoreMessages, setHasMoreMessages] = useState(false);
  const [totalMessageCount, setTotalMessageCount] = useState(0);
  const [shouldScrollInstantly, setShouldScrollInstantly] = useState(false);
  const scrollResetTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previousScrollHeightRef = useRef(0);

  const resetSessionView = useCallback(() => {
    setMessages([]);
    setHasMoreMessages(false);
    setTotalMessageCount(0);
    setShouldScrollInstantly(true);
    clearImages();
    onSessionChangeStart?.();

    if (scrollResetTimeoutRef.current) {
      clearTimeout(scrollResetTimeoutRef.current);
    }
  }, [clearImages, onSessionChangeStart]);

  const loadSessionHistory = useCallback(async (targetSessionId: string, updateUrl = true) => {
    if (!userId) {
      return;
    }

    if (updateUrl && targetSessionId !== sessionId) {
      window.history.pushState(null, '', `/chat/${targetSessionId}`);
      setSessionId(targetSessionId);
    }

    resetSessionView();
    setIsLoadingHistory(true);

    try {
      const response = await authenticatedFetch(
        `/api/${agentId}/sessions/${targetSessionId}/history?user_id=${userId}&limit=${MESSAGES_PER_PAGE}&offset=0`
      );

      if (!response.ok) {
        return;
      }

      const data: SessionHistoryResponse = await response.json();
      setMessages(data.messages.map(mapHistoryMessage));
      setHasMoreMessages(data.has_more || false);
      setTotalMessageCount(data.total || 0);
    } catch (error) {
      console.error('Error loading history:', error);
    } finally {
      setIsLoadingHistory(false);
      scrollResetTimeoutRef.current = setTimeout(() => {
        setShouldScrollInstantly(false);
        scrollResetTimeoutRef.current = null;
      }, SCROLL_RESET_DELAY);
    }
  }, [agentId, authenticatedFetch, resetSessionView, sessionId, setSessionId, userId]);

  const loadOlderMessages = useCallback(async () => {
    if (isLoadingOlderMessages || !hasMoreMessages || !sessionId || !userId) {
      return;
    }

    setIsLoadingOlderMessages(true);

    const container = scrollContainerRef.current;
    if (container) {
      previousScrollHeightRef.current = container.scrollHeight;
    }

    try {
      const currentOffset = messages.length;
      const response = await authenticatedFetch(
        `/api/${agentId}/sessions/${sessionId}/history?user_id=${userId}&limit=${MESSAGES_PER_PAGE}&offset=${currentOffset}`
      );

      if (!response.ok) {
        return;
      }

      const data: SessionHistoryResponse = await response.json();
      const olderMessages = data.messages.map(mapHistoryMessage);

      setMessages((prev) => [...olderMessages, ...prev]);
      setHasMoreMessages(data.has_more || false);

      setTimeout(() => {
        const nextContainer = scrollContainerRef.current;
        if (nextContainer && previousScrollHeightRef.current) {
          const scrollDiff = nextContainer.scrollHeight - previousScrollHeightRef.current;
          nextContainer.scrollTop = scrollDiff;
        }
      }, 0);
    } catch (error) {
      console.error('Error loading older messages:', error);
    } finally {
      setIsLoadingOlderMessages(false);
    }
  }, [agentId, authenticatedFetch, hasMoreMessages, isLoadingOlderMessages, messages.length, scrollContainerRef, sessionId, userId]);

  useEffect(() => {
    if (!userId || !urlSessionId) {
      return;
    }

    if (messages.length === 0 && !isLoadingHistory) {
      loadSessionHistory(urlSessionId, false);
    }
  }, [isLoadingHistory, loadSessionHistory, messages.length, urlSessionId, userId]);

  useEffect(() => {
    return () => {
      if (scrollResetTimeoutRef.current) {
        clearTimeout(scrollResetTimeoutRef.current);
      }
    };
  }, []);

  const shouldLoadOlderMessages = useCallback((scrollTop: number) => {
    return scrollTop < LOAD_MORE_THRESHOLD && hasMoreMessages && !isLoadingOlderMessages && !isLoadingHistory;
  }, [hasMoreMessages, isLoadingHistory, isLoadingOlderMessages]);

  return {
    messages,
    setMessages,
    isLoadingHistory,
    isLoadingOlderMessages,
    hasMoreMessages,
    totalMessageCount,
    shouldScrollInstantly,
    resetSessionView,
    loadSessionHistory,
    loadOlderMessages,
    shouldLoadOlderMessages,
  };
}
