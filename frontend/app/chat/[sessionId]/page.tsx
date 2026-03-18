'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Menu } from 'lucide-react';
import SessionSidebar from '@/app/components/SessionSidebar';
import { Button } from '@/app/components/ui/button';
import { useAuth } from '@/app/contexts/AuthContext';
import 'katex/dist/katex.min.css';
import { agentInfoMap, MAX_TEXTAREA_HEIGHT } from './constants';
import { ChatInput, CreateProjectModal, SelectionAskTooltip } from './components';
import { ChatMessagesPane } from './components/ChatMessagesPane';
import { ShareModal } from './components/ShareModal';
import { useFileUpload } from './hooks/useFileUpload';
import { useSessionData } from './hooks/useSessionData';
import { useStreamingChat } from './hooks/useStreamingChat';
import type { Message, SelectedImage } from './types';
import { mergeAssistantMessages, trimOuterNewlines } from './utils/messages';

const createErrorMessage = (content: string): Message => ({
  id: `msg-${Date.now()}-error`,
  role: 'assistant',
  content,
  timestamp: new Date(),
  isError: true,
});

export default function ChatPage() {
  const params = useParams();
  const router = useRouter();
  const { user, loading, authenticatedFetch } = useAuth();
  const urlSessionId = params.sessionId as string;
  const agentId = 'assistant';
  const agentInfo = agentInfoMap[agentId];

  const [inputValue, setInputValue] = useState('');
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false);
  const [copiedMessageId, setCopiedMessageId] = useState<string | null>(null);
  const [editingMessageId, setEditingMessageId] = useState<string | null>(null);
  const [editingValue, setEditingValue] = useState('');
  const [isSavingEdit, setIsSavingEdit] = useState(false);
  const [isShareModalOpen, setIsShareModalOpen] = useState(false);
  const [shareSessionId, setShareSessionId] = useState('');
  const [isCreateProjectModalOpen, setIsCreateProjectModalOpen] = useState(false);
  const [renamingProjectId, setRenamingProjectId] = useState<number | null>(null);
  const [renamingProjectName, setRenamingProjectName] = useState('');
  const [quotedText, setQuotedText] = useState('');

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const latestUserMessageRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const isAtBottomRef = useRef(true);
  const copiedUserMessageTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const {
    selectedImages,
    imageError,
    handleImageSelect,
    handlePaste,
    handleRemoveImage,
    clearImages,
  } = useFileUpload();

  const {
    messages,
    setMessages,
    sessionId,
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
    onSessionChangeStart: () => {
      setEditingMessageId(null);
      setEditingValue('');
    },
  });

  const { isLoading, sendMessage, handleCancelMessage } = useStreamingChat({
    agentId,
    sessionId,
    currentProjectId,
    sessions,
    messages,
    userId: user?.user_id,
    authenticatedFetch,
    clearImages,
    loadSessions,
    setMessages,
    setInputValue,
  });

  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
      return;
    }

    if (!agentInfo) {
      router.push('/');
    }
  }, [agentInfo, loading, router, user]);

  const isStreamingResponse = useMemo(
    () => messages.some((message) => message.isStreaming),
    [messages]
  );

  const latestUserMessageId = useMemo(
    () => [...messages].reverse().find((message) => message.role === 'user')?.id,
    [messages]
  );

  const processedMessages = useMemo(() => mergeAssistantMessages(messages), [messages]);

  const pageTitle = useMemo(() => {
    const currentSession = sessions.find((item) => item.session_id === sessionId);
    return currentSession?.title || currentSession?.first_message || '新对话';
  }, [sessionId, sessions]);

  useEffect(() => {
    document.title = pageTitle;
  }, [pageTitle]);

  useEffect(() => {
    const lastMessage = messages[messages.length - 1];
    if (!lastMessage || isStreamingResponse) {
      return;
    }

    if (lastMessage.role === 'user') {
      isAtBottomRef.current = false;
      latestUserMessageRef.current?.scrollIntoView({
        behavior: shouldScrollInstantly ? 'auto' : 'smooth',
        block: 'start',
      });
      return;
    }

    if (isAtBottomRef.current) {
      messagesEndRef.current?.scrollIntoView({
        behavior: shouldScrollInstantly ? 'auto' : 'smooth',
      });
    }
  }, [isStreamingResponse, latestUserMessageId, messages, shouldScrollInstantly]);

  useEffect(() => {
    if (messages.length === 0 && !isLoadingHistory) {
      textareaRef.current?.focus();
    }
  }, [isLoadingHistory, messages.length]);

  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) {
      return;
    }

    textarea.style.height = 'auto';
    textarea.style.height = `${Math.min(textarea.scrollHeight, MAX_TEXTAREA_HEIGHT)}px`;
  }, [inputValue]);

  useEffect(() => {
    return () => {
      if (copiedUserMessageTimeoutRef.current) {
        clearTimeout(copiedUserMessageTimeoutRef.current);
      }
    };
  }, []);

  const handleScroll = () => {
    const container = scrollContainerRef.current;
    if (!container) {
      return;
    }

    const { scrollTop, scrollHeight, clientHeight } = container;
    isAtBottomRef.current = scrollHeight - scrollTop - clientHeight < 10;

    if (shouldLoadOlderMessages(scrollTop)) {
      loadOlderMessages();
    }
  };

  const handleStartNewSession = () => {
    createNewSession();
    setInputValue('');
    setQuotedText('');
    setTimeout(() => {
      textareaRef.current?.focus();
    }, 0);
  };

  const handleDeleteSession = async (targetSessionId: string) => {
    await deleteSession(targetSessionId);
    if (targetSessionId === sessionId) {
      setInputValue('');
      setQuotedText('');
      setTimeout(() => {
        textareaRef.current?.focus();
      }, 0);
    }
  };

  const handleEditUserMessage = (message: Message) => {
    if (isLoading || isSavingEdit) {
      return;
    }

    if (message.id.startsWith('msg-')) {
      setMessages((prev) => [
        ...prev,
        createErrorMessage('这条消息还没有同步到历史记录。请刷新页面后再编辑。'),
      ]);
      return;
    }

    setEditingMessageId(message.id);
    setEditingValue(message.content || '');
  };

  const handleCancelEditUserMessage = () => {
    if (isSavingEdit) {
      return;
    }

    setEditingMessageId(null);
    setEditingValue('');
  };

  const handleSaveEditedUserMessage = async (message: Message) => {
    if (isLoading || isSavingEdit) {
      return;
    }

    const trimmedEditedText = trimOuterNewlines(editingValue);
    const hasImages = (message.images?.length || 0) > 0;
    if (!trimmedEditedText && !hasImages) {
      setMessages((prev) => [...prev, createErrorMessage('编辑后的消息不能为空。')]);
      return;
    }

    try {
      setIsSavingEdit(true);

      const response = await authenticatedFetch(`/api/${agentId}/sessions/${sessionId}/edit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          user_id: user?.user_id,
          message_id: message.id,
          new_content: trimmedEditedText,
          images: message.images || [],
          file_names: message.fileNames || [],
        }),
      });

      if (!response.ok) {
        let errorDetail = `HTTP error! status: ${response.status}`;
        try {
          const errorData = await response.json();
          if (errorData?.error) {
            errorDetail += ` - ${errorData.error}`;
          }
        } catch {
          // Keep status message when body is not JSON.
        }

        throw new Error(errorDetail);
      }

      const data = await response.json();
      const newSessionId = data?.new_session_id;
      if (!newSessionId) {
        throw new Error('编辑成功但未返回新会话 ID');
      }

      const editedImages: SelectedImage[] = (message.images || []).map((dataUrl, index) => ({
        id: `edited-${Date.now()}-${index}`,
        dataUrl,
        name: message.fileNames?.[index] || `文件 ${index + 1}`,
        isPdf: dataUrl.startsWith('data:application/pdf'),
      }));

      setEditingMessageId(null);
      setEditingValue('');
      await loadSessionHistory(newSessionId, true);
      await sendMessage(trimmedEditedText, editedImages, newSessionId);
      await loadSessions(true);
    } catch (error) {
      console.error('Error editing message:', error);
      setMessages((prev) => [
        ...prev,
        createErrorMessage(
          '编辑消息失败，请稍后重试。\n\n错误详情：' + (error instanceof Error ? error.message : String(error))
        ),
      ]);
    } finally {
      setIsSavingEdit(false);
    }
  };

  const handleCopyUserMessage = async (message: Message) => {
    const content = message.content || '';
    if (!content.trim()) {
      return;
    }

    try {
      await navigator.clipboard.writeText(content);
      setCopiedMessageId(message.id);

      if (copiedUserMessageTimeoutRef.current) {
        clearTimeout(copiedUserMessageTimeoutRef.current);
      }

      copiedUserMessageTimeoutRef.current = setTimeout(() => {
        setCopiedMessageId(null);
        copiedUserMessageTimeoutRef.current = null;
      }, 1500);
    } catch (error) {
      console.error('Failed to copy user message:', error);
    }
  };

  const handleAskAI = (text: string) => {
    setQuotedText(text);
    setTimeout(() => {
      textareaRef.current?.focus();
    }, 0);
  };

  const buildMessageWithQuote = (text: string) => (
    quotedText
      ? `> ${quotedText.split('\n').join('\n> ')}\n\n${text}`.trim()
      : text
  );

  const canSend = inputValue.trim().length > 0 || selectedImages.length > 0 || quotedText.length > 0;

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    const fullMessage = buildMessageWithQuote(inputValue);
    setQuotedText('');
    sendMessage(fullMessage, selectedImages);
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
      if (!canSend) {
        return;
      }

      event.preventDefault();
      const fullMessage = buildMessageWithQuote(inputValue);
      setQuotedText('');
      sendMessage(fullMessage, selectedImages);
    }
  };

  const handleExampleClick = (example: string) => {
    if (isLoading) {
      return;
    }

    sendMessage(example, []);
  };

  if (loading || !agentInfo) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-background font-sans text-foreground">
      <SessionSidebar
        sessions={sessions}
        projects={projects}
        activeProjectId={activeProjectId}
        isLoading={isSessionsLoading}
        currentSessionId={sessionId}
        onSessionSelect={handleSessionSelect}
        onProjectSelect={setActiveProjectId}
        onCreateProject={() => setIsCreateProjectModalOpen(true)}
        onRenameProject={(projectId, projectName) => {
          setRenamingProjectId(projectId);
          setRenamingProjectName(projectName);
        }}
        onDeleteProject={handleDeleteProject}
        onAssignSessionProject={handleAssignSessionProject}
        onNewSession={handleStartNewSession}
        onDeleteSession={handleDeleteSession}
        onShareSession={(targetSessionId) => {
          setShareSessionId(targetSessionId);
          setIsShareModalOpen(true);
        }}
        isMobileOpen={isMobileSidebarOpen}
        onMobileToggle={() => setIsMobileSidebarOpen(!isMobileSidebarOpen)}
      />

      <div className="flex flex-col flex-1 h-full md:pl-[260px] relative transition-all duration-300">
        <div className="md:hidden fixed top-3 left-3 z-30">
          <Button
            onClick={() => setIsMobileSidebarOpen(true)}
            size="icon"
            variant="outline"
            className="h-10 w-10 rounded-full bg-background shadow-lg tap-highlight-transparent min-h-[44px] min-w-[44px]"
            aria-label="打开菜单"
          >
            <Menu className="h-5 w-5" />
          </Button>
        </div>

        <div
          ref={scrollContainerRef}
          className="flex-1 overflow-y-auto no-scrollbar mobile-scroll"
          onScroll={handleScroll}
        >
          <div className="flex flex-col items-center">
            <div className="w-full max-w-5xl px-3 sm:px-4 md:px-6 py-6 sm:py-8 md:py-10">
              <ChatMessagesPane
                agentInfo={agentInfo}
                user={user ? { name: user.name, picture: user.picture } : null}
                messages={messages}
                processedMessages={processedMessages}
                latestUserMessageId={latestUserMessageId}
                latestUserMessageRef={latestUserMessageRef}
                messagesEndRef={messagesEndRef}
                isLoadingOlderMessages={isLoadingOlderMessages}
                hasMoreMessages={hasMoreMessages}
                totalMessageCount={totalMessageCount}
                isLoadingHistory={isLoadingHistory}
                isLoading={isLoading}
                editingMessageId={editingMessageId}
                editingValue={editingValue}
                isSavingEdit={isSavingEdit}
                copiedMessageId={copiedMessageId}
                onLoadOlderMessages={loadOlderMessages}
                onExampleClick={handleExampleClick}
                onRetry={() => sendMessage('', [])}
                onEditingValueChange={setEditingValue}
                onStartEdit={handleEditUserMessage}
                onSaveEdit={handleSaveEditedUserMessage}
                onCancelEdit={handleCancelEditUserMessage}
                onCopyUserMessage={handleCopyUserMessage}
              />
            </div>
          </div>
        </div>

        <ChatInput
          ref={textareaRef}
          inputValue={inputValue}
          onInputChange={setInputValue}
          onKeyDown={handleKeyDown}
          onPaste={handlePaste}
          onSubmit={handleSubmit}
          onCancel={handleCancelMessage}
          onFocus={() => {}}
          selectedImages={selectedImages}
          onRemoveImage={handleRemoveImage}
          onImageSelect={handleImageSelect}
          imageError={imageError}
          isLoading={isLoading}
          isLoadingHistory={isLoadingHistory}
          canSend={canSend}
          agentName={agentInfo.name}
          quotedText={quotedText}
          onClearQuote={() => setQuotedText('')}
        />
      </div>

      <SelectionAskTooltip onAskAI={handleAskAI} />

      <ShareModal
        isOpen={isShareModalOpen && shareSessionId !== ''}
        onClose={() => {
          setIsShareModalOpen(false);
          setShareSessionId('');
        }}
        sessionId={shareSessionId}
        agentId={agentId}
      />

      <CreateProjectModal
        isOpen={isCreateProjectModalOpen}
        onClose={() => setIsCreateProjectModalOpen(false)}
        onSubmit={handleCreateProject}
        title="创建项目"
        description="输入项目名称，用来归类当前会话。"
        submitLabel="创建项目"
        submittingLabel="创建中"
      />

      <CreateProjectModal
        isOpen={renamingProjectId !== null}
        onClose={() => {
          setRenamingProjectId(null);
          setRenamingProjectName('');
        }}
        initialValue={renamingProjectName}
        title="重命名项目"
        description="更新项目名称，相关会话会自动显示新名称。"
        submitLabel="保存"
        submittingLabel="保存中"
        onSubmit={async (projectName) => {
          if (renamingProjectId === null) {
            return;
          }

          await handleRenameProject(renamingProjectId, projectName);
          setRenamingProjectId(null);
          setRenamingProjectName('');
        }}
      />
    </div>
  );
}
