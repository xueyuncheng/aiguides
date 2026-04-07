'use client';

import { useCallback, useEffect, useMemo, useRef } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import 'katex/dist/katex.min.css';
import { agentInfoMap } from './[sessionId]/constants';
import { ChatPageLayout } from './[sessionId]/components/ChatPageLayout';
import { useFileUpload } from './[sessionId]/hooks/useFileUpload';
import { useMessageActions } from './[sessionId]/hooks/useMessageActions';
import { useScrollManager } from './[sessionId]/hooks/useScrollManager';
import { useSessionData } from './[sessionId]/hooks/useSessionData';
import { useStreamingChat } from './[sessionId]/hooks/useStreamingChat';
import { useUIState } from './[sessionId]/hooks/useUIState';
import { mergeAssistantMessages } from './[sessionId]/utils/messages';

export default function ChatPageClient() {
  const params = useParams();
  const router = useRouter();
  const { user, loading, authenticatedFetch } = useAuth();
  const routeSessionId = params.sessionId;
  const urlSessionId = Array.isArray(routeSessionId) ? routeSessionId[0] : routeSessionId as string | undefined;
  const agentId = 'assistant';
  const agentInfo = agentInfoMap[agentId];

  // Shared refs — created here so both useSessionData and useScrollManager can use them
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  // Stable ref — filled after actions is initialized
  const onSessionChangeStartRef = useRef<() => void>(() => {});
  const setInputValueRef = useRef<React.Dispatch<React.SetStateAction<string>>>(() => {});

  const { selectedImages, imageError, handleImageSelect, handlePaste, handleRemoveImage, clearImages } =
    useFileUpload();

  const {
    messages,
    setMessages,
    sessionId,
    setSessionId,
    sessions,
    projects,
    activeProjectId,
    setActiveProjectId,
    currentProjectId,
    isLoadingHistory,
    isLoadingOlderMessages,
    hasMoreMessages,
    totalMessageCount,
    isSessionsLoading,
    shouldScrollInstantly,
    loadSessions,
    loadSessionHistory,
    loadOlderMessages,
    handleSessionSelect,
    handleNewSession: createNewSession,
    handleDeleteSession: deleteSession,
    handleCreateProject,
    handleDeleteProject,
    handleRenameProject,
    handleAssignSessionProject,
    shouldLoadOlderMessages,
    markSessionLoaded,
  } = useSessionData({
    agentId,
    userId: user?.user_id,
    urlSessionId,
    authenticatedFetch,
    clearImages,
    scrollContainerRef,
    onSessionChangeStart: () => onSessionChangeStartRef.current(),
  });

  const { isLoading, sendMessage, handleCancelMessage } = useStreamingChat({
    agentId,
    sessionId,
    setSessionId,
    currentProjectId,
    sessions,
    messages,
    userId: user?.user_id,
    authenticatedFetch,
    clearImages,
    loadSessions,
    setMessages,
    setInputValue: (v) => setInputValueRef.current(v),
    markSessionLoaded,
  });

  const isStreamingResponse = useMemo(
    () => messages.some((m) => m.isStreaming),
    [messages]
  );

  const latestUserMessageId = useMemo(
    () => [...messages].reverse().find((m) => m.role === 'user')?.id,
    [messages]
  );

  const actions = useMessageActions({
    agentId,
    sessionId,
    userId: user?.user_id,
    isLoading,
    authenticatedFetch,
    setMessages,
    loadSessionHistory,
    loadSessions,
    sendMessage,
    textareaRef,
  });

  const { setEditingMessageId, setEditingValue, setInputValue } = actions;

  useEffect(() => {
    onSessionChangeStartRef.current = () => {
      setEditingMessageId(null);
      setEditingValue('');
    };
    setInputValueRef.current = setInputValue;
  }, [setEditingMessageId, setEditingValue, setInputValue]);

  const scroll = useScrollManager({
    messages,
    isStreamingResponse,
    isLoading,
    latestUserMessageId,
    isLoadingHistory,
    inputValue: actions.inputValue,
    shouldScrollInstantly,
    shouldLoadOlderMessages,
    loadOlderMessages,
    textareaRef,
    scrollContainerRef,
  });

  const ui = useUIState();
  const wasLoadingRef = useRef(false);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (ui.isShareModalOpen || ui.isCreateProjectModalOpen || ui.renamingProjectId !== null) {
        return;
      }

      if (event.ctrlKey && event.key.toLowerCase() === 'o') {
        event.preventDefault();
        textareaRef.current?.focus();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [ui.isCreateProjectModalOpen, ui.isShareModalOpen, ui.renamingProjectId]);

  useEffect(() => {
    const wasLoading = wasLoadingRef.current;
    wasLoadingRef.current = isLoading;

    if (!wasLoading || isLoading || isLoadingHistory) {
      return;
    }

    if (ui.isShareModalOpen || ui.isCreateProjectModalOpen || ui.renamingProjectId !== null) {
      return;
    }

    if (actions.editingMessageId !== null) {
      return;
    }

    setTimeout(() => textareaRef.current?.focus(), 0);
  }, [
    actions.editingMessageId,
    isLoading,
    isLoadingHistory,
    ui.isCreateProjectModalOpen,
    ui.isShareModalOpen,
    ui.renamingProjectId,
  ]);

  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
      return;
    }
    if (!agentInfo) router.push('/');
  }, [agentInfo, loading, router, user]);

  const processedMessages = useMemo(() => mergeAssistantMessages(messages), [messages]);
  const chatUser = useMemo(
    () => (user ? { name: user.name, picture: user.picture } : null),
    [user]
  );

  const pageTitle = useMemo(() => {
    const current = sessions.find((s) => s.session_id === sessionId);
    return current?.title || current?.first_message || '新对话';
  }, [sessionId, sessions]);

  useEffect(() => {
    document.title = pageTitle;
  }, [pageTitle]);

  const handleStartNewSession = useCallback(() => {
    createNewSession();
    actions.setInputValue('');
    actions.setQuotedText('');
    setTimeout(() => textareaRef.current?.focus(), 0);
  }, [actions, createNewSession]);

  const handleDeleteSession = useCallback(
    async (targetSessionId: string) => {
      await deleteSession(targetSessionId);
      if (targetSessionId === sessionId) {
        actions.setInputValue('');
        actions.setQuotedText('');
        setTimeout(() => textareaRef.current?.focus(), 0);
      }
    },
    [actions, deleteSession, sessionId]
  );

  const canSend = actions.canSend(selectedImages);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => actions.handleSubmit(e, selectedImages),
    [actions, selectedImages]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => actions.handleKeyDown(e, selectedImages),
    [actions, selectedImages]
  );

  const handleSubmitRenameProject = useCallback(
    async (projectName: string) => {
      if (ui.renamingProjectId === null) return;
      await handleRenameProject(ui.renamingProjectId, projectName);
      ui.handleCloseRenameProjectModal();
    },
    [handleRenameProject, ui]
  );

  if (loading || !agentInfo) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <ChatPageLayout
      agentInfo={agentInfo}
      agentId={agentId}
      sessionId={sessionId}
      sessions={sessions}
      projects={projects}
      activeProjectId={activeProjectId}
      currentProjectId={currentProjectId}
      isSessionsLoading={isSessionsLoading}
      chatUser={chatUser}
      messages={messages}
      processedMessages={processedMessages}
      isLoadingHistory={isLoadingHistory}
      isLoadingOlderMessages={isLoadingOlderMessages}
      hasMoreMessages={hasMoreMessages}
      totalMessageCount={totalMessageCount}
      isStreamingResponse={isStreamingResponse}
      isLoading={isLoading}
      latestUserMessageId={latestUserMessageId}
      editingMessageId={actions.editingMessageId}
      editingValue={actions.editingValue}
      isSavingEdit={actions.isSavingEdit}
      copiedMessageId={actions.copiedMessageId}
      scrollContainerRef={scrollContainerRef}
      messagesEndRef={scroll.messagesEndRef}
      latestUserMessageRef={scroll.latestUserMessageRef}
      chatInputContainerRef={scroll.chatInputContainerRef}
      textareaRef={textareaRef}
      chatInputOffset={scroll.chatInputOffset}
      inputValue={actions.inputValue}
      quotedText={actions.quotedText}
      selectedImages={selectedImages}
      imageError={imageError}
      canSend={canSend}
      isMobileSidebarOpen={ui.isMobileSidebarOpen}
      isShareModalOpen={ui.isShareModalOpen}
      shareSessionId={ui.shareSessionId}
      isCreateProjectModalOpen={ui.isCreateProjectModalOpen}
      renamingProjectId={ui.renamingProjectId}
      renamingProjectName={ui.renamingProjectName}
      onSessionSelect={handleSessionSelect}
      onProjectSelect={setActiveProjectId}
      onNewSession={handleStartNewSession}
      onDeleteSession={handleDeleteSession}
      onShareSession={ui.handleOpenShareModal}
      onCreateProject={ui.handleOpenCreateProjectModal}
      onRenameProject={ui.handleStartRenameProject}
      onDeleteProject={handleDeleteProject}
      onAssignSessionProject={handleAssignSessionProject}
      onToggleMobileSidebar={ui.handleToggleMobileSidebar}
      onOpenMobileSidebar={ui.handleOpenMobileSidebar}
      onLoadOlderMessages={loadOlderMessages}
      onRetry={actions.handleRetry}
      onEditingValueChange={actions.setEditingValue}
      onStartEdit={actions.handleEditUserMessage}
      onSaveEdit={actions.handleSaveEditedUserMessage}
      onCancelEdit={actions.handleCancelEditUserMessage}
      onCopyUserMessage={actions.handleCopyUserMessage}
      onAskAI={actions.handleAskAI}
      onInputChange={actions.setInputValue}
      onKeyDown={handleKeyDown}
      onPaste={handlePaste}
      onSubmit={handleSubmit}
      onCancel={handleCancelMessage}
      onFocus={actions.handleInputFocus}
      onRemoveImage={handleRemoveImage}
      onImageSelect={handleImageSelect}
      onClearQuote={actions.handleClearQuote}
      onCloseShareModal={ui.handleCloseShareModal}
      onCloseCreateProjectModal={ui.handleCloseCreateProjectModal}
      onSubmitCreateProject={handleCreateProject}
      onCloseRenameProjectModal={ui.handleCloseRenameProjectModal}
      onSubmitRenameProject={handleSubmitRenameProject}
      onScroll={scroll.handleScroll}
    />
  );
}
