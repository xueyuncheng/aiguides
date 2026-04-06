'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import type { RefObject } from 'react';
import { getChatPath } from '@/app/chat/utils/session';
import { LOAD_MORE_THRESHOLD, MESSAGES_PER_PAGE, SCROLL_RESET_DELAY } from '../constants';
import type { Message, SessionHistoryResponse } from '../types';
import { mapHistoryMessage } from '../utils/messages';

type AuthenticatedFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

interface UseSessionHistoryParams {
  agentId: string;
  userId?: string;
  sessionId: string;
  urlSessionId?: string;
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
  const autoLoadedSessionIdRef = useRef<string | null>(null);
  const pendingSessionIdRef = useRef<string | null>(null);
  const historyRequestIdRef = useRef(0);
  const scrollResetTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previousScrollHeightRef = useRef(0);

  const resetSessionView = useCallback((preserveRequestId = false) => {
    if (!preserveRequestId) {
      historyRequestIdRef.current += 1;
      autoLoadedSessionIdRef.current = null;
    }

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
    if (!userId || !targetSessionId) {
      return;
    }

    autoLoadedSessionIdRef.current = targetSessionId;
    const requestId = ++historyRequestIdRef.current;

    if (updateUrl && targetSessionId !== sessionId) {
      pendingSessionIdRef.current = targetSessionId;
      window.history.pushState(null, '', getChatPath(targetSessionId));
      setSessionId(targetSessionId);
    } else {
      pendingSessionIdRef.current = null;
    }

    resetSessionView(true);
    setIsLoadingHistory(true);

    try {
      const response = await authenticatedFetch(
        `/api/${agentId}/sessions/${targetSessionId}/history?user_id=${userId}&limit=${MESSAGES_PER_PAGE}&offset=0`
      );

      if (!response.ok) {
        return;
      }

      const data: SessionHistoryResponse = await response.json();
      if (historyRequestIdRef.current !== requestId) {
        return;
      }

      setMessages(data.messages.map(mapHistoryMessage));
      setHasMoreMessages(data.has_more || false);
      setTotalMessageCount(data.total || 0);
    } catch (error) {
      console.error('Error loading history:', error);
    } finally {
      if (historyRequestIdRef.current !== requestId) {
        return;
      }

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
    if (!userId || !sessionId || autoLoadedSessionIdRef.current === sessionId) {
      return;
    }

    if (pendingSessionIdRef.current && pendingSessionIdRef.current !== sessionId) {
      return;
    }

    pendingSessionIdRef.current = null;

    if (!urlSessionId && autoLoadedSessionIdRef.current === null) {
      return;
    }

    void loadSessionHistory(sessionId, false);
  }, [loadSessionHistory, sessionId, urlSessionId, userId]);

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
