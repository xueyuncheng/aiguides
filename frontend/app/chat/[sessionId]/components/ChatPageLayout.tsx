import { type Ref, type RefObject } from 'react';
import { Menu } from 'lucide-react';
import SessionSidebar from '@/app/components/SessionSidebar';
import { Button } from '@/app/components/ui/button';
import { ChatInput, CreateProjectModal, SelectionAskTooltip } from './index';
import { ChatMessagesPane } from './ChatMessagesPane';
import { ShareModal } from './ShareModal';
import { COMPOSER_MESSAGE_GAP } from '../constants';
import type { AgentInfo, Message, SelectedImage } from '../types';
import type { Session, Project } from '@/app/components/SessionSidebar';

interface ChatPageLayoutProps {
  // Agent / session
  agentInfo: AgentInfo;
  agentId: string;
  sessionId: string;
  sessions: Session[];
  projects: Project[];
  activeProjectId: string;
  currentProjectId: number | null;
  isSessionsLoading: boolean;
  chatUser: { name: string; picture?: string } | null;

  // Messages
  messages: Message[];
  processedMessages: Message[];
  isLoadingHistory: boolean;
  isLoadingOlderMessages: boolean;
  hasMoreMessages: boolean;
  totalMessageCount: number;
  isStreamingResponse: boolean;
  isLoading: boolean;
  latestUserMessageId: string | undefined;

  // Editing
  editingMessageId: string | null;
  editingValue: string;
  isSavingEdit: boolean;
  copiedMessageId: string | null;

  // Scroll refs
  scrollContainerRef: RefObject<HTMLDivElement | null>;
  messagesEndRef: RefObject<HTMLDivElement | null>;
  latestUserMessageRef: RefObject<HTMLDivElement | null>;
  chatInputContainerRef: Ref<HTMLDivElement>;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  chatInputOffset: number;

  // Input
  inputValue: string;
  quotedText: string;
  selectedImages: SelectedImage[];
  imageError: string | null;
  canSend: boolean;

  // UI state
  isMobileSidebarOpen: boolean;
  isShareModalOpen: boolean;
  shareSessionId: string;
  isCreateProjectModalOpen: boolean;
  renamingProjectId: number | null;
  renamingProjectName: string;

  // Handlers – session/sidebar
  onSessionSelect: (sessionId: string) => void;
  onProjectSelect: (projectId: string) => void;
  onNewSession: () => void;
  onDeleteSession: (sessionId: string) => void;
  onShareSession: (sessionId: string) => void;
  onCreateProject: () => void;
  onRenameProject: (projectId: number, projectName: string) => void;
  onDeleteProject: (projectId: number) => void;
  onAssignSessionProject: (sessionId: string, projectId: number) => void;
  onToggleMobileSidebar: () => void;
  onOpenMobileSidebar: () => void;

  // Handlers – messages
  onLoadOlderMessages: () => void;
  onRetry: () => void;
  onEditingValueChange: (value: string) => void;
  onStartEdit: (message: Message) => void;
  onSaveEdit: (message: Message) => void;
  onCancelEdit: () => void;
  onCopyUserMessage: (message: Message) => void;
  onAskAI: (text: string) => void;

  // Handlers – input
  onInputChange: (value: string) => void;
  onKeyDown: (e: React.KeyboardEvent<HTMLTextAreaElement>) => void;
  onPaste: (e: React.ClipboardEvent<HTMLTextAreaElement>) => void;
  onSubmit: (e: React.FormEvent) => void;
  onCancel: () => void;
  onFocus: () => void;
  onRemoveImage: (id: string) => void;
  onImageSelect: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onClearQuote: () => void;

  // Voice input
  isRecording?: boolean;
  isVoiceSupported?: boolean;
  onVoiceToggle?: () => void;
  voiceError?: string | null;

  // Handlers – modals
  onCloseShareModal: () => void;
  onCloseCreateProjectModal: () => void;
  onSubmitCreateProject: (name: string) => Promise<void>;
  onCloseRenameProjectModal: () => void;
  onSubmitRenameProject: (name: string) => Promise<void>;

  // Scroll
  onScroll: () => void;
}

export function ChatPageLayout({
  agentInfo,
  agentId,
  sessionId,
  sessions,
  projects,
  activeProjectId,
  isSessionsLoading,
  chatUser,
  messages,
  processedMessages,
  isLoadingHistory,
  isLoadingOlderMessages,
  hasMoreMessages,
  totalMessageCount,
  isStreamingResponse,
  isLoading,
  latestUserMessageId,
  editingMessageId,
  editingValue,
  isSavingEdit,
  copiedMessageId,
  scrollContainerRef,
  messagesEndRef,
  latestUserMessageRef,
  chatInputContainerRef,
  textareaRef,
  chatInputOffset,
  inputValue,
  quotedText,
  selectedImages,
  imageError,
  canSend,
  isMobileSidebarOpen,
  isShareModalOpen,
  shareSessionId,
  isCreateProjectModalOpen,
  renamingProjectId,
  renamingProjectName,
  onSessionSelect,
  onProjectSelect,
  onNewSession,
  onDeleteSession,
  onShareSession,
  onCreateProject,
  onRenameProject,
  onDeleteProject,
  onAssignSessionProject,
  onToggleMobileSidebar,
  onOpenMobileSidebar,
  onLoadOlderMessages,
  onRetry,
  onEditingValueChange,
  onStartEdit,
  onSaveEdit,
  onCancelEdit,
  onCopyUserMessage,
  onAskAI,
  onInputChange,
  onKeyDown,
  onPaste,
  onSubmit,
  onCancel,
  onFocus,
  onRemoveImage,
  onImageSelect,
  onClearQuote,
  isRecording,
  isVoiceSupported,
  onVoiceToggle,
  voiceError,
  onCloseShareModal,
  onCloseCreateProjectModal,
  onSubmitCreateProject,
  onCloseRenameProjectModal,
  onSubmitRenameProject,
  onScroll,
}: ChatPageLayoutProps) {
  const isEmptyState = messages.length === 0 && !isLoadingHistory;

  const messagesPaneProps = {
    agentInfo,
    user: chatUser,
    messages,
    processedMessages,
    latestUserMessageId,
    latestUserMessageRef,
    messagesEndRef,
    isStreamingResponse,
    isLoadingOlderMessages,
    hasMoreMessages,
    totalMessageCount,
    isLoadingHistory,
    isLoading,
    editingMessageId,
    editingValue,
    isSavingEdit,
    copiedMessageId,
    onLoadOlderMessages,
    onRetry,
    onEditingValueChange,
    onStartEdit,
    onSaveEdit,
    onCancelEdit,
    onCopyUserMessage,
  };

  const chatInputProps = {
    containerRef: chatInputContainerRef,
    ref: textareaRef,
    inputValue,
    onInputChange,
    onKeyDown,
    onPaste,
    onSubmit,
    onCancel,
    onFocus,
    selectedImages,
    onRemoveImage,
    onImageSelect,
    imageError,
    isLoading,
    isLoadingHistory,
    canSend,
    agentName: agentInfo.name,
    quotedText,
    onClearQuote,
    isRecording,
    isVoiceSupported,
    onVoiceToggle,
    voiceError,
  };

  return (
    <div className="flex h-screen bg-background font-sans text-foreground overflow-hidden">
      <SessionSidebar
        sessions={sessions}
        projects={projects}
        activeProjectId={activeProjectId}
        isLoading={isSessionsLoading}
        currentSessionId={sessionId}
        onSessionSelect={onSessionSelect}
        onProjectSelect={onProjectSelect}
        onCreateProject={onCreateProject}
        onRenameProject={onRenameProject}
        onDeleteProject={onDeleteProject}
        onAssignSessionProject={onAssignSessionProject}
        onNewSession={onNewSession}
        onDeleteSession={onDeleteSession}
        onShareSession={onShareSession}
        isMobileOpen={isMobileSidebarOpen}
        onMobileToggle={onToggleMobileSidebar}
      />

      <div className="flex flex-col flex-1 h-full min-w-0 md:pl-[260px] relative transition-all duration-300 overflow-x-hidden">
        <div className="md:hidden fixed top-3 left-3 z-30">
          <Button
            onClick={onOpenMobileSidebar}
            size="icon"
            variant="outline"
            className="h-10 w-10 rounded-full bg-background shadow-lg tap-highlight-transparent min-h-[44px] min-w-[44px]"
            aria-label="打开菜单"
          >
            <Menu className="h-5 w-5" />
          </Button>
        </div>

        {isEmptyState ? (
          <div className="flex-1 flex flex-col items-center justify-center px-3 sm:px-4 md:px-6">
            <div className="w-full max-w-4xl">
              <div className="mx-auto mb-6 max-w-2xl px-3 text-center sm:mb-8">
                <h1 className="text-2xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-100 sm:text-3xl">
                  开始对话
                </h1>
              </div>
              <ChatInput {...chatInputProps} centered />
            </div>
          </div>
        ) : (
          <>
            <div
              ref={scrollContainerRef}
              className="flex-1 overflow-x-hidden overflow-y-auto no-scrollbar mobile-scroll min-w-0"
              onScroll={onScroll}
            >
              <div className="flex flex-col items-center min-w-0 w-full">
                <div
                  className="w-full max-w-5xl px-3 sm:px-4 md:px-6 pt-6 sm:pt-8 md:pt-10 min-w-0 mx-auto"
                  style={{ paddingBottom: `${Math.max(16, chatInputOffset + COMPOSER_MESSAGE_GAP)}px` }}
                >
                  <ChatMessagesPane {...messagesPaneProps} />
                </div>
              </div>
            </div>
            <ChatInput {...chatInputProps} />
          </>
        )}
      </div>

      <SelectionAskTooltip onAskAI={onAskAI} />

      <ShareModal
        isOpen={isShareModalOpen && shareSessionId !== ''}
        onClose={onCloseShareModal}
        sessionId={shareSessionId}
        agentId={agentId}
      />

      <CreateProjectModal
        isOpen={isCreateProjectModalOpen}
        onClose={onCloseCreateProjectModal}
        onSubmit={onSubmitCreateProject}
        title="创建项目"
        description="输入项目名称，用来归类当前会话。"
        submitLabel="创建项目"
        submittingLabel="创建中"
      />

      <CreateProjectModal
        isOpen={renamingProjectId !== null}
        onClose={onCloseRenameProjectModal}
        initialValue={renamingProjectName}
        title="重命名项目"
        description="更新项目名称，相关会话会自动显示新名称。"
        submitLabel="保存"
        submittingLabel="保存中"
        onSubmit={onSubmitRenameProject}
      />
    </div>
  );
}
