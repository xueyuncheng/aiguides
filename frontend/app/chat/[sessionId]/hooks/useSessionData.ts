'use client';

import { useCallback, useState } from 'react';
import type { RefObject } from 'react';
import { useProjectSessions } from './useProjectSessions';
import { useSessionHistory } from './useSessionHistory';

type AuthenticatedFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

interface UseSessionDataParams {
  agentId: string;
  userId?: string;
  urlSessionId: string;
  authenticatedFetch: AuthenticatedFetch;
  clearImages: () => void;
  onSessionChangeStart?: () => void;
  scrollContainerRef: RefObject<HTMLDivElement | null>;
}

export function useSessionData({
  agentId,
  userId,
  urlSessionId,
  authenticatedFetch,
  clearImages,
  onSessionChangeStart,
  scrollContainerRef,
}: UseSessionDataParams) {
  const [sessionId, setSessionId] = useState(urlSessionId || '');
  const historyState = useSessionHistory({
    agentId,
    userId,
    sessionId,
    urlSessionId,
    authenticatedFetch,
    clearImages,
    onSessionChangeStart,
    scrollContainerRef,
    setSessionId,
  });

  const projectState = useProjectSessions({
    agentId,
    userId,
    sessionId,
    messagesCount: historyState.messages.length,
    authenticatedFetch,
    setSessionId,
    onSessionReset: historyState.resetSessionView,
  });

  const handleSessionSelect = useCallback(async (newSessionId: string) => {
    if (newSessionId === sessionId) {
      return;
    }

    const selectedSession = projectState.sessions.find((item) => item.session_id === newSessionId);
    projectState.setCurrentProjectId(selectedSession?.project_id ?? 0);
    await historyState.loadSessionHistory(newSessionId, true);
  }, [historyState, projectState, sessionId]);

  return {
    messages: historyState.messages,
    setMessages: historyState.setMessages,
    sessionId,
    sessions: projectState.sessions,
    projects: projectState.projects,
    activeProjectId: projectState.activeProjectId,
    setActiveProjectId: projectState.setActiveProjectId,
    currentProjectId: projectState.currentProjectId,
    isLoadingHistory: historyState.isLoadingHistory,
    isLoadingOlderMessages: historyState.isLoadingOlderMessages,
    hasMoreMessages: historyState.hasMoreMessages,
    totalMessageCount: historyState.totalMessageCount,
    isSessionsLoading: projectState.isSessionsLoading,
    shouldScrollInstantly: historyState.shouldScrollInstantly,
    loadSessions: projectState.loadSessions,
    loadProjects: projectState.loadProjects,
    loadSessionHistory: historyState.loadSessionHistory,
    loadOlderMessages: historyState.loadOlderMessages,
    handleSessionSelect,
    handleNewSession: projectState.handleNewSession,
    handleDeleteSession: projectState.handleDeleteSession,
    handleCreateProject: projectState.handleCreateProject,
    handleDeleteProject: projectState.handleDeleteProject,
    handleRenameProject: projectState.handleRenameProject,
    handleAssignSessionProject: projectState.handleAssignSessionProject,
    shouldLoadOlderMessages: historyState.shouldLoadOlderMessages,
  };
}
