'use client';

import { useCallback, useEffect, useState } from 'react';
import type { Session } from '@/app/components/SessionSidebar';
import { getChatPath } from '@/app/chat/utils/session';

type AuthenticatedFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

interface UseSessionsListParams {
  agentId: string;
  userId?: string;
  sessionId: string;
  activeProjectId: string;
  authenticatedFetch: AuthenticatedFetch;
  setSessionId: React.Dispatch<React.SetStateAction<string>>;
  onSessionReset: () => void;
}

const getProjectIdFromFilter = (projectId: string) => {
  if (projectId === 'all' || projectId === 'none') {
    return 0;
  }

  const parsedProjectId = Number(projectId);
  return Number.isNaN(parsedProjectId) ? 0 : parsedProjectId;
};

export function useSessionsList({
  agentId,
  userId,
  sessionId,
  activeProjectId,
  authenticatedFetch,
  setSessionId,
  onSessionReset,
}: UseSessionsListParams) {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [currentProjectId, setCurrentProjectId] = useState(0);
  const [isSessionsLoading, setIsSessionsLoading] = useState(false);

  const loadSessions = useCallback(async (silent = false) => {
    if (!userId) {
      return undefined;
    }

    try {
      if (!silent) {
        setIsSessionsLoading(true);
      }

      const response = await authenticatedFetch(`/api/${agentId}/sessions?user_id=${userId}`);
      if (!response.ok) {
        return undefined;
      }

      const data = await response.json();
      const sortedSessions = (data || []).sort((a: Session, b: Session) => (
        new Date(b.last_update_time).getTime() - new Date(a.last_update_time).getTime()
      ));

      setSessions(sortedSessions);
      return sortedSessions;
    } catch (error) {
      console.error('Error loading sessions:', error);
      return undefined;
    } finally {
      if (!silent) {
        setIsSessionsLoading(false);
      }
    }
  }, [agentId, authenticatedFetch, userId]);

  const handleNewSession = useCallback(() => {
    window.history.pushState(null, '', getChatPath());
    setSessionId('');
    setCurrentProjectId(getProjectIdFromFilter(activeProjectId));
    onSessionReset();
  }, [activeProjectId, onSessionReset, setSessionId]);

  const handleDeleteSession = useCallback(async (sessionIdToDelete: string) => {
    try {
      const response = await authenticatedFetch(`/api/${agentId}/sessions/${sessionIdToDelete}?user_id=${userId}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        return;
      }

      setSessions((prev) => prev.filter((session) => session.session_id !== sessionIdToDelete));
      if (sessionIdToDelete === sessionId) {
        handleNewSession();
      }
    } catch (error) {
      console.error('Error deleting session:', error);
    }
  }, [agentId, authenticatedFetch, handleNewSession, sessionId, userId]);

  useEffect(() => {
    if (userId) {
      loadSessions();
    }
  }, [loadSessions, userId]);

  useEffect(() => {
    const currentSession = sessions.find((item) => item.session_id === sessionId);
    if (currentSession) {
      setCurrentProjectId(currentSession.project_id ?? 0);
    }
  }, [sessionId, sessions]);

  return {
    sessions,
    setSessions,
    currentProjectId,
    setCurrentProjectId,
    isSessionsLoading,
    loadSessions,
    handleNewSession,
    handleDeleteSession,
  };
}
