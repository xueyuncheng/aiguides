import { useCallback, useState } from 'react';

interface UseUIStateResult {
  isMobileSidebarOpen: boolean;
  isShareModalOpen: boolean;
  shareSessionId: string;
  isCreateProjectModalOpen: boolean;
  renamingProjectId: number | null;
  renamingProjectName: string;
  handleOpenCreateProjectModal: () => void;
  handleStartRenameProject: (projectId: number, projectName: string) => void;
  handleOpenShareModal: (targetSessionId: string) => void;
  handleCloseShareModal: () => void;
  handleCloseCreateProjectModal: () => void;
  handleCloseRenameProjectModal: () => void;
  handleToggleMobileSidebar: () => void;
  handleOpenMobileSidebar: () => void;
}

export function useUIState(): UseUIStateResult {
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false);
  const [isShareModalOpen, setIsShareModalOpen] = useState(false);
  const [shareSessionId, setShareSessionId] = useState('');
  const [isCreateProjectModalOpen, setIsCreateProjectModalOpen] = useState(false);
  const [renamingProjectId, setRenamingProjectId] = useState<number | null>(null);
  const [renamingProjectName, setRenamingProjectName] = useState('');

  const handleOpenCreateProjectModal = useCallback(() => {
    setIsCreateProjectModalOpen(true);
  }, []);

  const handleCloseCreateProjectModal = useCallback(() => {
    setIsCreateProjectModalOpen(false);
  }, []);

  const handleStartRenameProject = useCallback((projectId: number, projectName: string) => {
    setRenamingProjectId(projectId);
    setRenamingProjectName(projectName);
  }, []);

  const handleCloseRenameProjectModal = useCallback(() => {
    setRenamingProjectId(null);
    setRenamingProjectName('');
  }, []);

  const handleOpenShareModal = useCallback((targetSessionId: string) => {
    setShareSessionId(targetSessionId);
    setIsShareModalOpen(true);
  }, []);

  const handleCloseShareModal = useCallback(() => {
    setIsShareModalOpen(false);
    setShareSessionId('');
  }, []);

  const handleToggleMobileSidebar = useCallback(() => {
    setIsMobileSidebarOpen((prev) => !prev);
  }, []);

  const handleOpenMobileSidebar = useCallback(() => {
    setIsMobileSidebarOpen(true);
  }, []);

  return {
    isMobileSidebarOpen,
    isShareModalOpen,
    shareSessionId,
    isCreateProjectModalOpen,
    renamingProjectId,
    renamingProjectName,
    handleOpenCreateProjectModal,
    handleStartRenameProject,
    handleOpenShareModal,
    handleCloseShareModal,
    handleCloseCreateProjectModal,
    handleCloseRenameProjectModal,
    handleToggleMobileSidebar,
    handleOpenMobileSidebar,
  };
}
