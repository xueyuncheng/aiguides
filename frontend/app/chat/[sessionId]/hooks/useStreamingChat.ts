'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import type { Session } from '@/app/components/SessionSidebar';
import { getChatPath, resolveSessionId } from '@/app/chat/utils/session';
import type { Message, SelectedImage } from '../types';
import { trimOuterNewlines } from '../utils/messages';
import { consumeAssistantStream, createAssistantErrorMessage } from '../utils/assistantStream';

type AuthenticatedFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

interface UseStreamingChatParams {
  agentId: string;
  sessionId: string;
  setSessionId: React.Dispatch<React.SetStateAction<string>>;
  currentProjectId: number;
  sessions: Session[];
  messages: Message[];
  userId?: string;
  authenticatedFetch: AuthenticatedFetch;
  clearImages: () => void;
  loadSessions: (silent?: boolean) => Promise<Session[] | undefined>;
  setMessages: React.Dispatch<React.SetStateAction<Message[]>>;
  setInputValue: React.Dispatch<React.SetStateAction<string>>;
}

export function useStreamingChat({
  agentId,
  sessionId,
  setSessionId,
  currentProjectId,
  sessions,
  messages,
  userId,
  authenticatedFetch,
  clearImages,
  loadSessions,
  setMessages,
  setInputValue,
}: UseStreamingChatParams) {
  const [isLoading, setIsLoading] = useState(false);
  const abortControllerRef = useRef<AbortController | null>(null);
  const titlePollIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const activeSessionIdRef = useRef(sessionId);

  useEffect(() => {
    activeSessionIdRef.current = sessionId;
  }, [sessionId]);

  const clearTitlePoll = useCallback(() => {
    if (titlePollIntervalRef.current) {
      clearInterval(titlePollIntervalRef.current);
      titlePollIntervalRef.current = null;
    }
  }, []);

  const handleCancelMessage = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    clearTitlePoll();
    setIsLoading(false);
  }, [clearTitlePoll]);

  const sendMessage = useCallback(async (content: string, images: SelectedImage[], targetSessionId: string = sessionId) => {
    if (isLoading) {
      return;
    }

    const resolvedSessionId = resolveSessionId(targetSessionId, activeSessionIdRef.current);
    const isDraftSession = !activeSessionIdRef.current;
    if (resolvedSessionId !== activeSessionIdRef.current || isDraftSession) {
      window.history.pushState(null, '', getChatPath(resolvedSessionId));
      activeSessionIdRef.current = resolvedSessionId;
      setSessionId(resolvedSessionId);
    }

    const trimmedContent = trimOuterNewlines(content);
    const hasImages = images.length > 0;
    const isRetry = !trimmedContent && !hasImages;
    const lastUserMessage = isRetry
      ? [...messages].reverse().find((message) => message.role === 'user')
      : undefined;

    if (isRetry && !lastUserMessage) {
      return;
    }

    if (isRetry) {
      setMessages((prev) => {
        if (prev.length > 0 && prev[prev.length - 1].isError) {
          return prev.slice(0, -1);
        }

        return prev;
      });
    } else {
      const userMessage: Message = {
        id: `msg-${Date.now()}`,
        role: 'user',
        content: trimmedContent,
        images: images.map((image) => image.dataUrl),
        fileNames: images.map((image) => image.name),
        timestamp: new Date(),
      };

      setMessages((prev) => [...prev, userMessage]);
      setInputValue('');
      clearImages();
    }

    setIsLoading(true);
    abortControllerRef.current = new AbortController();

    const isFirstMessage = messages.filter((message) => message.role === 'user').length === 0;
    if (isFirstMessage) {
      clearTitlePoll();

      let pollCount = 0;
      const maxPolls = 30;

      titlePollIntervalRef.current = setInterval(async () => {
        const fetchedSessions = await loadSessions(true);
        const currentSession = fetchedSessions?.find((item) => item.session_id === resolvedSessionId);

        if (currentSession?.title) {
          clearTitlePoll();
          return;
        }

        pollCount += 1;
        if (pollCount >= maxPolls) {
          clearTitlePoll();
        }
      }, 1000);
    }

    try {
      const requestMessage = isRetry ? (lastUserMessage?.content || '') : trimmedContent;
      const imageData = isRetry ? (lastUserMessage?.images || []) : images.map((image) => image.dataUrl);
      const fileNames = isRetry ? (lastUserMessage?.fileNames || []) : images.map((image) => image.name);
      const sessionProjectId = sessions.find((item) => item.session_id === resolvedSessionId)?.project_id ?? currentProjectId;

      const response = await authenticatedFetch(`/api/${agentId}/chats/${resolvedSessionId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          user_id: userId,
          session_id: resolvedSessionId,
          message: requestMessage,
          images: imageData,
          file_names: fileNames,
          project_id: sessionProjectId,
        }),
        signal: abortControllerRef.current.signal,
      });

      if (!response.ok) {
        let errorDetail = `HTTP error! status: ${response.status}`;
        try {
          const errorData = await response.json();
          if (errorData?.error) {
            errorDetail += ` - ${errorData.error}`;
          }
        } catch {
          // Ignore body parse failure and keep HTTP status message.
        }

        throw new Error(errorDetail);
      }

      const reader = response.body?.getReader();

      if (reader) {
        await consumeAssistantStream({
          reader,
          setIsLoading,
          setMessages,
        });
      }
    } catch (error) {
      clearTitlePoll();

      if (error instanceof Error && error.name === 'AbortError') {
        console.log('Request cancelled by user');
      } else {
        console.error('Error sending message:', error);
        setMessages((prev) => [
          ...prev,
          createAssistantErrorMessage(
            '抱歉，发生了错误。请确保后端服务正在运行，并稍后重试。\n\n错误详情：'
              + (error instanceof Error ? error.message : String(error))
          ),
        ]);
      }
    } finally {
      setIsLoading(false);
      abortControllerRef.current = null;
    }
  }, [agentId, authenticatedFetch, clearImages, clearTitlePoll, currentProjectId, isLoading, loadSessions, messages, sessionId, sessions, setInputValue, setMessages, setSessionId, userId]);

  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      clearTitlePoll();
    };
  }, [clearTitlePoll]);

  return {
    isLoading,
    sendMessage,
    handleCancelMessage,
  };
}
