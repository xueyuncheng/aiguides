'use client';

import { useCallback, useEffect, useState } from 'react';
import type { Project, Session } from '@/app/components/SessionSidebar';

type AuthenticatedFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

interface UseProjectsParams {
  agentId: string;
  userId?: string;
  sessionId: string;
  messagesCount: number;
  activeProjectId: string;
  currentProjectId: number;
  authenticatedFetch: AuthenticatedFetch;
  loadSessions: (silent?: boolean) => Promise<Session[] | undefined>;
  setSessions: React.Dispatch<React.SetStateAction<Session[]>>;
  setCurrentProjectId: React.Dispatch<React.SetStateAction<number>>;
  setActiveProjectId: React.Dispatch<React.SetStateAction<string>>;
}

export function useProjects({
  agentId,
  userId,
  sessionId,
  messagesCount,
  activeProjectId,
  currentProjectId,
  authenticatedFetch,
  loadSessions,
  setSessions,
  setCurrentProjectId,
  setActiveProjectId,
}: UseProjectsParams) {
  const [projects, setProjects] = useState<Project[]>([]);

  const loadProjects = useCallback(async () => {
    try {
      const response = await authenticatedFetch('/api/assistant/projects');
      if (!response.ok) {
        return;
      }

      const data = await response.json();
      setProjects(data || []);
    } catch (error) {
      console.error('Error loading projects:', error);
    }
  }, [authenticatedFetch]);

  const handleCreateProject = useCallback(async (projectName: string) => {
    try {
      const response = await authenticatedFetch('/api/assistant/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: projectName }),
      });

      if (!response.ok) {
        let errorMessage = '创建项目失败，请稍后重试';

        try {
          const errorData = await response.json();
          errorMessage = errorData.error || errorData.message || errorMessage;
        } catch {
          errorMessage = `创建项目失败 (${response.status})`;
        }

        throw new Error(errorMessage);
      }

      const project = await response.json();
      setProjects((prev) => {
        const nextProjects = prev.filter((item) => item.id !== project.id);
        return [project, ...nextProjects];
      });
      setActiveProjectId(String(project.id));

      if (messagesCount === 0) {
        setCurrentProjectId(project.id);
      }
    } catch (error) {
      console.error('Error creating project:', error);
      throw error;
    }
  }, [authenticatedFetch, messagesCount, setActiveProjectId, setCurrentProjectId]);

  const handleDeleteProject = useCallback(async (projectId: number) => {
    try {
      const response = await authenticatedFetch(`/api/assistant/projects/${projectId}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        let errorMessage = '删除项目失败，请稍后重试';

        try {
          const errorData = await response.json();
          errorMessage = errorData.error || errorData.message || errorMessage;
        } catch {
          errorMessage = `删除项目失败 (${response.status})`;
        }

        throw new Error(errorMessage);
      }

      setProjects((prev) => prev.filter((project) => project.id !== projectId));
      setSessions((prev) => prev.map((session) => (
        session.project_id === projectId
          ? { ...session, project_id: 0, project_name: '' }
          : session
      )));

      if (activeProjectId === String(projectId)) {
        setActiveProjectId('all');
      }

      if (currentProjectId === projectId) {
        setCurrentProjectId(0);
      }
    } catch (error) {
      console.error('Error deleting project:', error);
    }
  }, [activeProjectId, authenticatedFetch, currentProjectId, setActiveProjectId, setCurrentProjectId, setSessions]);

  const handleRenameProject = useCallback(async (projectId: number, projectName: string) => {
    const trimmedProjectName = projectName.trim();
    if (!trimmedProjectName) {
      return;
    }

    try {
      const response = await authenticatedFetch(`/api/assistant/projects/${projectId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: trimmedProjectName }),
      });

      if (!response.ok) {
        let errorMessage = '重命名项目失败，请稍后重试';

        try {
          const errorData = await response.json();
          errorMessage = errorData.error || errorData.message || errorMessage;
        } catch {
          errorMessage = `重命名项目失败 (${response.status})`;
        }

        throw new Error(errorMessage);
      }

      const updatedProject = await response.json();
      setProjects((prev) => prev.map((project) => (
        project.id === projectId
          ? { ...project, name: updatedProject.name }
          : project
      )));
      setSessions((prev) => prev.map((session) => (
        session.project_id === projectId
          ? { ...session, project_name: updatedProject.name }
          : session
      )));
    } catch (error) {
      console.error('Error renaming project:', error);
      throw error;
    }
  }, [authenticatedFetch, setSessions]);

  const handleAssignSessionProject = useCallback(async (targetSessionId: string, projectId: number) => {
    try {
      const response = await authenticatedFetch(`/api/${agentId}/sessions/${targetSessionId}/project`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ project_id: projectId }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      await loadSessions(true);
      await loadProjects();

      if (targetSessionId === sessionId) {
        setCurrentProjectId(projectId);
      }
    } catch (error) {
      console.error('Error updating session project:', error);
    }
  }, [agentId, authenticatedFetch, loadProjects, loadSessions, sessionId, setCurrentProjectId]);

  useEffect(() => {
    if (userId) {
      loadProjects();
    }
  }, [loadProjects, userId]);

  return {
    projects,
    activeProjectId,
    setActiveProjectId,
    loadProjects,
    handleCreateProject,
    handleDeleteProject,
    handleRenameProject,
    handleAssignSessionProject,
  };
}
