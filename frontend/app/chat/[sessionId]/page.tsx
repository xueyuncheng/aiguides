'use client';

import { useCallback, useEffect, useMemo, useRef } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import 'katex/dist/katex.min.css';
import { agentInfoMap } from './constants';
import { ChatPageLayout } from './components/ChatPageLayout';
import { useFileUpload } from './hooks/useFileUpload';
import { useMessageActions } from './hooks/useMessageActions';
import { useScrollManager } from './hooks/useScrollManager';
import { useSessionData } from './hooks/useSessionData';
import { useStreamingChat } from './hooks/useStreamingChat';
import { useUIState } from './hooks/useUIState';
import { mergeAssistantMessages } from './utils/messages';

export default function ChatPage() {
  const params = useParams();
  const router = useRouter();
  const { user, loading, authenticatedFetch } = useAuth();
  const urlSessionId = params.sessionId as string | undefined;
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

  // Wire the session-change callback now that actions is available
  onSessionChangeStartRef.current = () => {
    actions.setEditingMessageId(null);
    actions.setEditingValue('');
  };
  setInputValueRef.current = actions.setInputValue;

  const scroll = useScrollManager({
    messages,
    isStreamingResponse,
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
      // Agent / session
      agentInfo={agentInfo}
      agentId={agentId}
      sessionId={sessionId}
      sessions={sessions}
      projects={projects}
      activeProjectId={activeProjectId}
      currentProjectId={currentProjectId}
      isSessionsLoading={isSessionsLoading}
      chatUser={chatUser}
      // Messages
      messages={messages}
      processedMessages={processedMessages}
      isLoadingHistory={isLoadingHistory}
      isLoadingOlderMessages={isLoadingOlderMessages}
      hasMoreMessages={hasMoreMessages}
      totalMessageCount={totalMessageCount}
      isStreamingResponse={isStreamingResponse}
      isLoading={isLoading}
      latestUserMessageId={latestUserMessageId}
      // Editing
      editingMessageId={actions.editingMessageId}
      editingValue={actions.editingValue}
      isSavingEdit={actions.isSavingEdit}
      copiedMessageId={actions.copiedMessageId}
      // Scroll refs
      scrollContainerRef={scrollContainerRef}
      messagesEndRef={scroll.messagesEndRef}
      latestUserMessageRef={scroll.latestUserMessageRef}
      chatInputContainerRef={scroll.chatInputContainerRef}
      textareaRef={textareaRef}
      chatInputOffset={scroll.chatInputOffset}
      // Input
      inputValue={actions.inputValue}
      quotedText={actions.quotedText}
      selectedImages={selectedImages}
      imageError={imageError}
      canSend={canSend}
      // UI state
      isMobileSidebarOpen={ui.isMobileSidebarOpen}
      isShareModalOpen={ui.isShareModalOpen}
      shareSessionId={ui.shareSessionId}
      isCreateProjectModalOpen={ui.isCreateProjectModalOpen}
      renamingProjectId={ui.renamingProjectId}
      renamingProjectName={ui.renamingProjectName}
      // Session/sidebar handlers
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
      // Message handlers
      onLoadOlderMessages={loadOlderMessages}
      onRetry={actions.handleRetry}
      onEditingValueChange={actions.setEditingValue}
      onStartEdit={actions.handleEditUserMessage}
      onSaveEdit={actions.handleSaveEditedUserMessage}
      onCancelEdit={actions.handleCancelEditUserMessage}
      onCopyUserMessage={actions.handleCopyUserMessage}
      onAskAI={actions.handleAskAI}
      // Input handlers
      onInputChange={actions.setInputValue}
      onKeyDown={handleKeyDown}
      onPaste={handlePaste}
      onSubmit={handleSubmit}
      onCancel={handleCancelMessage}
      onFocus={actions.handleInputFocus}
      onRemoveImage={handleRemoveImage}
      onImageSelect={handleImageSelect}
      onClearQuote={actions.handleClearQuote}
      // Modal handlers
      onCloseShareModal={ui.handleCloseShareModal}
      onCloseCreateProjectModal={ui.handleCloseCreateProjectModal}
      onSubmitCreateProject={handleCreateProject}
      onCloseRenameProjectModal={ui.handleCloseRenameProjectModal}
      onSubmitRenameProject={handleSubmitRenameProject}
      // Scroll
      onScroll={scroll.handleScroll}
    />
  );
}
