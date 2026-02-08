'use client';

import { useState, useEffect, useRef, useMemo } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/app/contexts/AuthContext';
import 'katex/dist/katex.min.css';
import SessionSidebar, { Session } from '@/app/components/SessionSidebar';
import { Button } from '@/app/components/ui/button';
import { Check, Copy, Menu, Pencil, X } from 'lucide-react';
import { cn } from '@/app/lib/utils';

// 导入类型和常量
import type { Message, SelectedImage } from './types';
import { agentInfoMap, MESSAGES_PER_PAGE, LOAD_MORE_THRESHOLD, MIN_SCROLL_THRESHOLD, MIN_SCROLL_DISTANCE, SCROLL_RESET_DELAY } from './constants';

// 导入组件
import { AIAvatar, UserAvatar, ChatSkeleton, AIMessageContent, UserMessage, ChatInput } from './components';

// 导入 hooks
import { useFileUpload } from './hooks/useFileUpload';

export default function ChatPage() {
  const params = useParams();
  const router = useRouter();
  const { user, loading, authenticatedFetch } = useAuth();
  const urlSessionId = params.sessionId as string;
  const agentId = 'assistant';
  const agentInfo = agentInfoMap[agentId];

  // 状态管理
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string>(urlSessionId || '');
  const [isLoadingHistory, setIsLoadingHistory] = useState(false);
  const [isLoadingOlderMessages, setIsLoadingOlderMessages] = useState(false);
  const [hasMoreMessages, setHasMoreMessages] = useState(false);
  const [totalMessageCount, setTotalMessageCount] = useState(0);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [isSessionsLoading, setIsSessionsLoading] = useState(false);
  const [shouldScrollInstantly, setShouldScrollInstantly] = useState(false);
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false);
  const [isInputVisible, setIsInputVisible] = useState(true);
  const [copiedUserMessageId, setCopiedUserMessageId] = useState<string | null>(null);
  const [editingMessageId, setEditingMessageId] = useState<string | null>(null);
  const [editingValue, setEditingValue] = useState('');
  const [isSavingEdit, setIsSavingEdit] = useState(false);

  // 使用文件上传 hook
  const {
    selectedImages,
    imageError,
    handleImageSelect,
    handlePaste,
    handleRemoveImage,
    clearImages,
  } = useFileUpload();

  // Refs
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const titlePollIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const scrollResetTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previousScrollHeightRef = useRef<number>(0);
  const isAtBottomRef = useRef(true);
  const lastScrollTopRef = useRef(0);
  const scrollDirectionTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const copiedUserMessageTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // 会话管理
  const loadSessions = async (silent = false) => {
    if (!user?.user_id) return;

    try {
      if (!silent) setIsSessionsLoading(true);
      const response = await authenticatedFetch(`/api/${agentId}/sessions?user_id=${user.user_id}`);
      if (response.ok) {
        const data = await response.json();
        const sortedSessions = (data || []).sort((a: Session, b: Session) =>
          new Date(b.last_update_time).getTime() - new Date(a.last_update_time).getTime()
        );
        setSessions(sortedSessions);
        return sortedSessions;
      }
    } catch (error) {
      console.error('Error loading sessions:', error);
    } finally {
      if (!silent) setIsSessionsLoading(false);
    }
  };

  useEffect(() => {
    if (user?.user_id) {
      loadSessions();
    }
  }, [user?.user_id]);

  const loadSessionHistory = async (targetSessionId: string, updateUrl: boolean = true) => {
    if (updateUrl && targetSessionId !== sessionId) {
      window.history.pushState(null, '', `/chat/${targetSessionId}`);
      setSessionId(targetSessionId);
    }

    setMessages([]);
    setHasMoreMessages(false);
    setTotalMessageCount(0);
    setIsLoadingHistory(true);
    setShouldScrollInstantly(true);
    setIsInputVisible(true);
    clearImages();
    setEditingMessageId(null);
    setEditingValue('');

    if (scrollResetTimeoutRef.current) {
      clearTimeout(scrollResetTimeoutRef.current);
    }

    try {
      const response = await authenticatedFetch(`/api/${agentId}/sessions/${targetSessionId}/history?user_id=${user?.user_id}&limit=${MESSAGES_PER_PAGE}&offset=0`);
      if (response.ok) {
        const data = await response.json();
        const historyMessages = data.messages.map((msg: any) => ({
          id: msg.id,
          role: msg.role,
          content: msg.content,
          thought: msg.thought,
          timestamp: new Date(msg.timestamp),
          images: msg.images || [],
          fileNames: msg.file_names || [],
        }));
        setMessages(historyMessages);
        setHasMoreMessages(data.has_more || false);
        setTotalMessageCount(data.total || 0);
      }
    } catch (error) {
      console.error('Error loading history:', error);
    } finally {
      setIsLoadingHistory(false);
      scrollResetTimeoutRef.current = setTimeout(() => {
        setShouldScrollInstantly(false);
        scrollResetTimeoutRef.current = null;
      }, SCROLL_RESET_DELAY);
    }
  };

  const handleSessionSelect = async (newSessionId: string) => {
    if (newSessionId === sessionId) return;
    await loadSessionHistory(newSessionId, true);
  };

  const loadOlderMessages = async () => {
    if (isLoadingOlderMessages || !hasMoreMessages || !sessionId) return;

    setIsLoadingOlderMessages(true);

    const container = scrollContainerRef.current;
    if (container) {
      previousScrollHeightRef.current = container.scrollHeight;
    }

    try {
      const currentOffset = messages.length;
      const response = await authenticatedFetch(`/api/${agentId}/sessions/${sessionId}/history?user_id=${user?.user_id}&limit=${MESSAGES_PER_PAGE}&offset=${currentOffset}`);
      if (response.ok) {
        const data = await response.json();
        const olderMessages = data.messages.map((msg: any) => ({
          id: msg.id,
          role: msg.role,
          content: msg.content,
          thought: msg.thought,
          timestamp: new Date(msg.timestamp),
          images: msg.images || [],
          fileNames: msg.file_names || [],
        }));

        setMessages(prev => [...olderMessages, ...prev]);
        setHasMoreMessages(data.has_more || false);

        setTimeout(() => {
          const container = scrollContainerRef.current;
          if (container && previousScrollHeightRef.current) {
            const newScrollHeight = container.scrollHeight;
            const scrollDiff = newScrollHeight - previousScrollHeightRef.current;
            container.scrollTop = scrollDiff;
          }
        }, 0);
      }
    } catch (error) {
      console.error('Error loading older messages:', error);
    } finally {
      setIsLoadingOlderMessages(false);
    }
  };

  const handleNewSession = async () => {
    const newSessionId = `session-${Date.now()}-${Math.random().toString(36).substring(7)}`;
    window.history.pushState(null, '', `/chat/${newSessionId}`);
    setSessionId(newSessionId);
    setMessages([]);
    setHasMoreMessages(false);
    setTotalMessageCount(0);
    setIsInputVisible(true);
    setInputValue('');
    clearImages();
    setTimeout(() => {
      textareaRef.current?.focus();
    }, 0);
  };

  const handleDeleteSession = async (sessionIdToDelete: string) => {
    try {
      const response = await authenticatedFetch(`/api/${agentId}/sessions/${sessionIdToDelete}?user_id=${user?.user_id}`, {
        method: 'DELETE',
      });

      if (response.ok) {
        setSessions(prev => prev.filter(s => s.session_id !== sessionIdToDelete));
        if (sessionIdToDelete === sessionId) {
          handleNewSession();
        }
      }
    } catch (error) {
      console.error('Error deleting session:', error);
    }
  };

  // 认证和路由处理
  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
      return;
    }

    if (!agentInfo) {
      router.push('/');
      return;
    }
  }, [agentInfo, router, user, loading]);

  // 加载会话历史
  useEffect(() => {
    if (!user?.user_id || !urlSessionId) return;

    if (messages.length === 0 && !isLoadingHistory) {
      loadSessionHistory(urlSessionId, false);
    }
  }, [urlSessionId, user?.user_id]);

  // 滚动处理
  const handleScroll = () => {
    if (scrollContainerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
      const atBottom = scrollHeight - scrollTop - clientHeight < 10;
      isAtBottomRef.current = atBottom;

      if (scrollTop < LOAD_MORE_THRESHOLD && hasMoreMessages && !isLoadingOlderMessages && !isLoadingHistory) {
        loadOlderMessages();
      }

      const scrollDelta = scrollTop - lastScrollTopRef.current;

      if (atBottom) {
        setIsInputVisible(true);
        lastScrollTopRef.current = scrollTop;
        return;
      }

      if (Math.abs(scrollDelta) > MIN_SCROLL_THRESHOLD && scrollTop > MIN_SCROLL_DISTANCE) {
        if (scrollDirectionTimeoutRef.current) {
          clearTimeout(scrollDirectionTimeoutRef.current);
        }

        if (scrollDelta < 0) {
          setIsInputVisible(false);
        } else if (scrollDelta > 0) {
          scrollDirectionTimeoutRef.current = setTimeout(() => {
            setIsInputVisible(true);
          }, 100);
        }
      }

      lastScrollTopRef.current = scrollTop;
    }
  };

  // 滚动到底部
  useEffect(() => {
    const isNewUserMessage = messages.length > 0 && messages[messages.length - 1].role === 'user';

    if (isNewUserMessage || isAtBottomRef.current) {
      messagesEndRef.current?.scrollIntoView({
        behavior: shouldScrollInstantly ? 'auto' : 'smooth'
      });
    }

    if (isNewUserMessage) {
      setIsInputVisible(true);
    }
  }, [messages, shouldScrollInstantly]);

  // 空会话时显示输入框
  useEffect(() => {
    if (messages.length === 0 && !isLoadingHistory) {
      if (!isInputVisible) {
        setIsInputVisible(true);
      }
      textareaRef.current?.focus();
    }
  }, [messages.length, isLoadingHistory, isInputVisible]);

  // 清理超时
  useEffect(() => {
    return () => {
      if (scrollResetTimeoutRef.current) {
        clearTimeout(scrollResetTimeoutRef.current);
      }
      if (scrollDirectionTimeoutRef.current) {
        clearTimeout(scrollDirectionTimeoutRef.current);
      }
      if (copiedUserMessageTimeoutRef.current) {
        clearTimeout(copiedUserMessageTimeoutRef.current);
      }
    };
  }, []);

  // 自动调整 textarea 高度
  useEffect(() => {
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = 'auto';
      textarea.style.height = `${Math.min(textarea.scrollHeight, 160)}px`;
    }
  }, [inputValue]);

  // 合并连续的 assistant 消息
  const processedMessages = useMemo(() => {
    if (messages.length === 0) return [];

    const result: Message[] = [];
    messages.forEach((msg) => {
      const last = result[result.length - 1];
      if (
        last &&
        last.role === 'assistant' &&
        msg.role === 'assistant' &&
        !last.isError &&
        !msg.isError
      ) {
        const merged = { ...last };
        merged.content = (merged.content || '') + (msg.content || '');
        if (msg.thought) {
          merged.thought = (merged.thought || '') + (merged.thought ? '\n\n' : '') + msg.thought;
        }
        if (msg.images && msg.images.length > 0) {
          merged.images = [...(merged.images || []), ...(msg.images || [])];
        }
        merged.isStreaming = last.isStreaming || msg.isStreaming;

        result[result.length - 1] = merged;
      } else {
        result.push(msg);
      }
    });
    return result;
  }, [messages]);

  const handleCancelMessage = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    if (titlePollIntervalRef.current) {
      clearInterval(titlePollIntervalRef.current);
      titlePollIntervalRef.current = null;
    }

    setIsLoading(false);
  };

  const handleEditUserMessage = (message: Message) => {
    if (isLoading || isSavingEdit) return;

    if (message.id.startsWith('msg-')) {
      const errorMessage: Message = {
        id: `msg-${Date.now()}-error`,
        role: 'assistant',
        content: '这条消息还没有同步到历史记录。请刷新页面后再编辑。',
        timestamp: new Date(),
        isError: true,
      };
      setMessages((prev) => [...prev, errorMessage]);
      return;
    }

    setEditingMessageId(message.id);
    setEditingValue(message.content || '');
  };

  const handleCancelEditUserMessage = () => {
    if (isSavingEdit) return;
    setEditingMessageId(null);
    setEditingValue('');
  };

  const handleSaveEditedUserMessage = async (message: Message) => {
    if (isLoading || isSavingEdit) return;

    const trimmedEditedText = editingValue.replace(/^[\n\r]+|[\n\r]+$/g, '');
    const hasImages = (message.images?.length || 0) > 0;
    if (!trimmedEditedText && !hasImages) {
      const errorMessage: Message = {
        id: `msg-${Date.now()}-error`,
        role: 'assistant',
        content: '编辑后的消息不能为空。',
        timestamp: new Date(),
        isError: true,
      };
      setMessages((prev) => [...prev, errorMessage]);
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
      const errorMessage: Message = {
        id: `msg-${Date.now()}-error`,
        role: 'assistant',
        content: '编辑消息失败，请稍后重试。\n\n错误详情：' + (error instanceof Error ? error.message : String(error)),
        timestamp: new Date(),
        isError: true,
      };
      setMessages((prev) => [...prev, errorMessage]);
    } finally {
      setIsSavingEdit(false);
    }
  };

  const handleCopyUserMessage = async (message: Message) => {
    const content = message.content || '';
    if (!content.trim()) return;

    try {
      await navigator.clipboard.writeText(content);
      setCopiedUserMessageId(message.id);
      if (copiedUserMessageTimeoutRef.current) {
        clearTimeout(copiedUserMessageTimeoutRef.current);
      }
      copiedUserMessageTimeoutRef.current = setTimeout(() => {
        setCopiedUserMessageId(null);
        copiedUserMessageTimeoutRef.current = null;
      }, 1500);
    } catch (error) {
      console.error('Failed to copy user message:', error);
    }
  };

  // 发送消息函数（SSE 流式处理）
  const sendMessage = async (content: string, images: SelectedImage[], targetSessionId: string = sessionId) => {
    if (isLoading) return;

    // Only trim leading and trailing newlines, preserving internal line breaks
    const trimmedContent = content.replace(/^[\n\r]+|[\n\r]+$/g, '');
    const hasImages = images.length > 0;
    const isRetry = !trimmedContent && !hasImages;
    const lastUserMessage = isRetry
      ? [...messages].reverse().find((msg) => msg.role === 'user')
      : undefined;
    if (isRetry && !lastUserMessage) return;

    if (isRetry) {
      setMessages((prev) => {
        if (prev.length > 0 && prev[prev.length - 1].isError) {
          return prev.slice(0, -1);
        }
        return prev;
      });
    } else {
      const imageData = images.map((image) => image.dataUrl);
      const fileNames = images.map((image) => image.name);
      const userMessage: Message = {
        id: `msg-${Date.now()}`,
        role: 'user',
        content: trimmedContent,
        images: imageData,
        fileNames: fileNames,
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, userMessage]);
      setInputValue('');
      clearImages();
    }

    setIsLoading(true);
    abortControllerRef.current = new AbortController();

    const isFirstMessage = messages.filter(m => m.role === 'user').length === 0;
    if (isFirstMessage) {
      if (titlePollIntervalRef.current) {
        clearInterval(titlePollIntervalRef.current);
      }

      let pollCount = 0;
      const maxPolls = 30;
      titlePollIntervalRef.current = setInterval(async () => {
        const fetchedSessions = await loadSessions(true);
        const currentSession = fetchedSessions?.find((s: Session) => s.session_id === targetSessionId);

        if (currentSession?.title) {
          if (titlePollIntervalRef.current) {
            clearInterval(titlePollIntervalRef.current);
            titlePollIntervalRef.current = null;
          }
          return;
        }

        pollCount++;
        if (pollCount >= maxPolls) {
          if (titlePollIntervalRef.current) {
            clearInterval(titlePollIntervalRef.current);
            titlePollIntervalRef.current = null;
          }
        }
      }, 1000);
    }

    try {
      const requestMessage = isRetry ? (lastUserMessage?.content || '') : trimmedContent;
      const imageData = isRetry ? (lastUserMessage?.images || []) : images.map((image) => image.dataUrl);
      const fileNames = isRetry ? (lastUserMessage?.fileNames || []) : images.map((image) => image.name);

      const response = await authenticatedFetch(`/api/${agentId}/chats/${targetSessionId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          user_id: user?.user_id,
          session_id: targetSessionId,
          message: requestMessage,
          images: imageData,
          file_names: fileNames,
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
      const decoder = new TextDecoder();
      let currentAuthor = '';
      let assistantContent = '';
      let assistantThought = '';
      let assistantImages: string[] = [];

      if (reader) {
        let buffer = '';
        let currentEventType = 'data';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          const chunk = decoder.decode(value, { stream: true });
          buffer += chunk;
          const lines = buffer.split('\n');
          buffer = lines.pop() || '';

          for (const line of lines) {
            const trimmedLine = line.trim();

            if (trimmedLine.startsWith('event:')) {
              currentEventType = trimmedLine.substring(6).trim();
              continue;
            }

            if (trimmedLine.startsWith('data:')) {
              try {
                const jsonStr = trimmedLine.substring(5).trim();
                if (!jsonStr) continue;

                const data = JSON.parse(jsonStr);

                if (currentEventType === 'heartbeat') {
                  currentEventType = 'data';
                  continue;
                }

                if (currentEventType === 'error') {
                  const errorMessage: Message = {
                    id: `msg-${Date.now()}-error`,
                    role: 'assistant',
                    content: `❌ **错误**: ${data.error || '发生未知错误，请稍后重试。'}`,
                    timestamp: new Date(),
                    isError: true,
                  };
                  setMessages((prev) => [...prev, errorMessage]);
                  setIsLoading(false);
                  currentEventType = 'data';
                  continue;
                }

                if (currentEventType === 'stop') {
                  currentEventType = 'data';
                  continue;
                }

                if (data.images && Array.isArray(data.images)) {
                  assistantImages = [...assistantImages, ...data.images];

                  setMessages((prev) => {
                    const newMessages = [...prev];
                    const lastIndex = newMessages.length - 1;
                    if (lastIndex >= 0 && newMessages[lastIndex].role === 'assistant') {
                      newMessages[lastIndex] = {
                        ...newMessages[lastIndex],
                        images: assistantImages,
                        isStreaming: true,
                      };
                    }
                    return newMessages;
                  });
                }

                if (data.content) {
                  const isCompleteDuplicate = !data.is_thought && data.content === assistantContent;

                  if (!isCompleteDuplicate) {
                    if (data.author && data.author !== currentAuthor) {
                      currentAuthor = data.author;
                      assistantContent = data.is_thought ? '' : data.content;
                      assistantThought = data.is_thought ? data.content : '';
                      assistantImages = [];

                      const newMessage: Message = {
                        id: `msg-${Date.now()}-${currentAuthor}`,
                        role: 'assistant',
                        content: assistantContent,
                        thought: assistantThought,
                        timestamp: new Date(),
                        author: currentAuthor,
                        isStreaming: true,
                        images: [],
                      };
                      setMessages((prev) => [...prev, newMessage]);
                    } else {
                      if (data.is_thought) {
                        assistantThought += data.content;
                      } else {
                        assistantContent += data.content;
                      }

                      setMessages((prev) => {
                        const newMessages = [...prev];
                        const lastIndex = newMessages.length - 1;
                        if (lastIndex >= 0 && newMessages[lastIndex].role === 'assistant') {
                          newMessages[lastIndex] = {
                            ...newMessages[lastIndex],
                            content: assistantContent,
                            thought: assistantThought,
                            images: assistantImages,
                            isStreaming: true,
                          };
                        }
                        return newMessages;
                      });
                    }
                  }
                }
              } catch (e) {
                console.warn('JSON parse error:', e);
              }
            }
          }
        }

        setMessages((prev) => prev.map(msg => ({ ...msg, isStreaming: false })));
      }
    } catch (error) {
      if (titlePollIntervalRef.current) {
        clearInterval(titlePollIntervalRef.current);
        titlePollIntervalRef.current = null;
      }

      if (error instanceof Error && error.name === 'AbortError') {
        console.log('Request cancelled by user');
      } else {
        console.error('Error sending message:', error);
        const errorMessage: Message = {
          id: `msg-${Date.now()}-error`,
          role: 'assistant',
          content: '抱歉，发生了错误。请确保后端服务正在运行，并稍后重试。\n\n错误详情：' + (error instanceof Error ? error.message : String(error)),
          timestamp: new Date(),
          isError: true,
        };
        setMessages((prev) => [...prev, errorMessage]);
      }
    } finally {
      setIsLoading(false);
      abortControllerRef.current = null;
    }
  };

  const canSend = inputValue.trim().length > 0 || selectedImages.length > 0;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    sendMessage(inputValue, selectedImages);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
      if (!canSend) return;
      e.preventDefault();
      sendMessage(inputValue, selectedImages);
    }
  };

  const handleExampleClick = (example: string) => {
    if (isLoading) return;
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
        isLoading={isSessionsLoading}
        currentSessionId={sessionId}
        onSessionSelect={handleSessionSelect}
        onNewSession={handleNewSession}
        onDeleteSession={handleDeleteSession}
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
            <div className="w-full max-w-5xl px-3 sm:px-4 md:px-6 py-6 sm:py-8 md:py-10 space-y-6 sm:space-y-8">
              {isLoadingOlderMessages && (
                <div className="flex justify-center py-4">
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
                    <span>加载更早的消息...</span>
                  </div>
                </div>
              )}

              {hasMoreMessages && !isLoadingOlderMessages && messages.length > 0 && (
                <div className="flex justify-center py-2">
                  <button
                    onClick={loadOlderMessages}
                    className="text-sm text-muted-foreground hover:text-foreground transition-colors underline"
                  >
                    还有 {totalMessageCount - messages.length} 条更早的消息，向上滚动或点击加载
                  </button>
                </div>
              )}

              {isLoadingHistory ? (
                <div className="flex justify-center w-full">
                  <ChatSkeleton />
                </div>
              ) : messages.length === 0 ? (
                <div className="text-center py-12 sm:py-16 md:py-20 animate-fade-in px-4">
                  <div className="flex justify-center mb-4 sm:mb-6">
                    <div className="p-3 sm:p-4 bg-secondary rounded-xl sm:rounded-2xl">
                      <span className="text-3xl sm:text-4xl">{agentInfo.icon}</span>
                    </div>
                  </div>
                  <h2 className="text-2xl font-semibold mb-8 tracking-tight">
                    {agentInfo.name} 能够为您做什么？
                  </h2>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 max-w-2xl mx-auto">
                    {agentInfo.examples.map((example, index) => (
                      <button
                        key={index}
                        onClick={() => handleExampleClick(example)}
                        className="p-3 sm:p-4 text-left border border-zinc-200 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700 rounded-lg sm:rounded-xl bg-zinc-50/50 dark:bg-zinc-900/30 hover:bg-white dark:hover:bg-zinc-900 active:bg-zinc-100 dark:active:bg-zinc-800 shadow-sm hover:shadow-md transition-all duration-200 text-xs sm:text-sm text-balance tap-highlight-transparent min-h-[44px]"
                      >
                        {example}
                      </button>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="space-y-6 sm:space-y-8 animate-fade-in">
                  {processedMessages.map((message) => (
                    <div
                      key={message.id}
                      className={cn(
                        "flex w-full group/message",
                        message.role === 'user' ? "justify-end" : "justify-start"
                      )}
                    >
                      <div className={cn(
                        "flex gap-2 sm:gap-3 md:gap-4 max-w-[95%] sm:max-w-[90%] md:max-w-[85%]",
                        message.role === 'user' ? "flex-row-reverse" : "flex-row"
                      )}>
                        {message.role === 'assistant' ? (
                          <AIAvatar icon={agentInfo.icon} />
                        ) : (
                          <UserAvatar user={user} />
                        )}

                        {message.role === 'assistant' ? (
                          <div className="relative text-sm w-full leading-relaxed pt-1 flex-1">
                            <AIMessageContent
                              content={message.content}
                              thought={message.thought}
                              isStreaming={message.isStreaming}
                              images={message.images}
                              isError={message.isError}
                              onRetry={() => sendMessage("", [])}
                            />
                          </div>
                        ) : (
                          <div className="relative flex flex-col items-end">
                            <div className="relative text-sm leading-relaxed bg-zinc-100 dark:bg-zinc-800 px-4 py-2.5 rounded-2xl rounded-tr-sm max-w-fit min-w-[180px]">
                              {editingMessageId === message.id ? (
                                <div className="space-y-2">
                                  {(message.images?.length || 0) > 0 && (
                                    <div className="text-xs text-muted-foreground">附件保持不变</div>
                                  )}
                                  <textarea
                                    value={editingValue}
                                    onChange={(e) => setEditingValue(e.target.value)}
                                    autoFocus
                                    rows={Math.max(4, Math.min(10, (message.content?.match(/\n/g)?.length || 0) + 2))}
                                    className="w-[min(520px,82vw)] min-h-[120px] resize-y rounded-md bg-transparent px-3 py-2 text-sm leading-relaxed text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-500 dark:placeholder:text-zinc-400 outline-none focus:ring-2 focus:ring-primary/30"
                                    placeholder="编辑消息内容..."
                                    disabled={isSavingEdit}
                                  />
                                </div>
                              ) : (
                                <UserMessage
                                  content={message.content}
                                  images={message.images}
                                  fileNames={message.fileNames}
                                />
                              )}
                            </div>

                            <div className={cn(
                              "mt-1 flex justify-end gap-1 transition-opacity duration-200",
                              editingMessageId === message.id
                                ? "opacity-100"
                                : "opacity-0 group-hover/message:opacity-100 group-focus-within/message:opacity-100"
                            )}>
                              {editingMessageId === message.id ? (
                                <>
                                  <Button
                                    size="sm"
                                    variant="ghost"
                                    onClick={() => handleSaveEditedUserMessage(message)}
                                    className="h-6 w-6 p-0 text-muted-foreground hover:text-foreground"
                                    title="保存编辑"
                                    aria-label="保存编辑"
                                    disabled={isSavingEdit}
                                  >
                                    <Check className="h-3.5 w-3.5" />
                                  </Button>
                                  <Button
                                    size="sm"
                                    variant="ghost"
                                    onClick={handleCancelEditUserMessage}
                                    className="h-6 w-6 p-0 text-muted-foreground hover:text-foreground"
                                    title="取消编辑"
                                    aria-label="取消编辑"
                                    disabled={isSavingEdit}
                                  >
                                    <X className="h-3.5 w-3.5" />
                                  </Button>
                                </>
                              ) : (
                                <>
                                  <Button
                                    size="sm"
                                    variant="ghost"
                                    onClick={() => handleEditUserMessage(message)}
                                    className="h-6 w-6 p-0 text-muted-foreground hover:text-foreground"
                                    title="编辑并创建新版本"
                                    aria-label="编辑并创建新版本"
                                  >
                                    <Pencil className="h-3.5 w-3.5" />
                                  </Button>
                                  <Button
                                    size="sm"
                                    variant="ghost"
                                    onClick={() => handleCopyUserMessage(message)}
                                    className="h-6 w-6 p-0 text-muted-foreground hover:text-foreground"
                                    title="复制文本"
                                    aria-label="复制文本"
                                    disabled={!message.content?.trim()}
                                  >
                                    {copiedUserMessageId === message.id ? (
                                      <Check className="h-3.5 w-3.5 text-green-600" />
                                    ) : (
                                      <Copy className="h-3.5 w-3.5" />
                                    )}
                                  </Button>
                                </>
                              )}
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                  {isLoading && (messages.length === 0 || messages[messages.length - 1].role !== 'assistant') && (
                    <div className="flex w-full justify-start">
                      <div className="flex gap-4 max-w-[85%]">
                        <AIAvatar icon={agentInfo.icon} />
                        <div className="pt-2">
                          <div className="flex items-center gap-2 text-sm text-muted-foreground animate-pulse">
                            <div className="flex space-x-1">
                              <div className="w-1.5 h-1.5 bg-muted-foreground/40 rounded-full animate-bounce [animation-delay:-0.3s]"></div>
                              <div className="w-1.5 h-1.5 bg-muted-foreground/40 rounded-full animate-bounce [animation-delay:-0.15s]"></div>
                              <div className="w-1.5 h-1.5 bg-muted-foreground/40 rounded-full animate-bounce"></div>
                            </div>
                            <span>AI 正在思考...</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  )}
                  <div ref={messagesEndRef} className="h-24" />
                </div>
              )}
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
          onFocus={() => setIsInputVisible(true)}
          selectedImages={selectedImages}
          onRemoveImage={handleRemoveImage}
          onImageSelect={handleImageSelect}
          imageError={imageError}
          isLoading={isLoading}
          isLoadingHistory={isLoadingHistory}
          canSend={canSend}
          agentName={agentInfo.name}
          isVisible={isInputVisible}
        />
      </div>
    </div>
  );
}
