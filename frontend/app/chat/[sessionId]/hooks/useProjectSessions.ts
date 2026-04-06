'use client';

import { useEffect, useState } from 'react';
import { useSessionsList } from './useSessionsList';
import { useProjects } from './useProjects';

type AuthenticatedFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

interface UseProjectSessionsParams {
  agentId: string;
  userId?: string;
  sessionId: string;
  messagesCount: number;
  authenticatedFetch: AuthenticatedFetch;
  setSessionId: React.Dispatch<React.SetStateAction<string>>;
  onSessionReset: () => void;
}

export function useProjectSessions({
  agentId,
  userId,
  sessionId,
  messagesCount,
  authenticatedFetch,
  setSessionId,
  onSessionReset,
}: UseProjectSessionsParams) {
  const [activeProjectId, setActiveProjectId] = useState(() => {
    if (typeof window === 'undefined') {
      return 'all';
    }

    return localStorage.getItem('activeProjectId') || 'all';
  });

  const sessionsState = useSessionsList({
    agentId,
    userId,
    sessionId,
    activeProjectId,
    authenticatedFetch,
    setSessionId,
    onSessionReset,
  });

  const projectsState = useProjects({
    agentId,
    userId,
    sessionId,
    messagesCount,
    activeProjectId,
    currentProjectId: sessionsState.currentProjectId,
    authenticatedFetch,
    loadSessions: sessionsState.loadSessions,
    setSessions: sessionsState.setSessions,
    setCurrentProjectId: sessionsState.setCurrentProjectId,
    setActiveProjectId,
  });

  useEffect(() => {
    localStorage.setItem('activeProjectId', activeProjectId);
  }, [activeProjectId]);

  return {
    sessions: sessionsState.sessions,
    projects: projectsState.projects,
    activeProjectId,
    setActiveProjectId,
    currentProjectId: sessionsState.currentProjectId,
    setCurrentProjectId: sessionsState.setCurrentProjectId,
    isSessionsLoading: sessionsState.isSessionsLoading,
    loadSessions: sessionsState.loadSessions,
    loadProjects: projectsState.loadProjects,
    handleNewSession: sessionsState.handleNewSession,
    handleDeleteSession: sessionsState.handleDeleteSession,
    handleCreateProject: projectsState.handleCreateProject,
    handleDeleteProject: projectsState.handleDeleteProject,
    handleRenameProject: projectsState.handleRenameProject,
    handleAssignSessionProject: projectsState.handleAssignSessionProject,
  };
}
