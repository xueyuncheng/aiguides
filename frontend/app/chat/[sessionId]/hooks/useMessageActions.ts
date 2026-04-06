import { useCallback, useRef, useState } from 'react';
import type { Message, SelectedImage } from '../types';
import { trimOuterNewlines } from '../utils/messages';

const createErrorMessage = (content: string): Message => ({
  id: `msg-${Date.now()}-error`,
  role: 'assistant',
  content,
  timestamp: new Date(),
  isError: true,
});

interface UseMessageActionsParams {
  agentId: string;
  sessionId: string;
  userId: string | undefined;
  isLoading: boolean;
  authenticatedFetch: (url: string, options?: RequestInit) => Promise<Response>;
  setMessages: React.Dispatch<React.SetStateAction<Message[]>>;
  loadSessionHistory: (sessionId: string, instant?: boolean) => Promise<void>;
  loadSessions: (force?: boolean) => Promise<void>;
  sendMessage: (text: string, images: SelectedImage[], overrideSessionId?: string) => Promise<void>;
  textareaRef: React.RefObject<HTMLTextAreaElement | null>;
}

interface UseMessageActionsResult {
  inputValue: string;
  setInputValue: React.Dispatch<React.SetStateAction<string>>;
  quotedText: string;
  setQuotedText: React.Dispatch<React.SetStateAction<string>>;
  editingMessageId: string | null;
  setEditingMessageId: React.Dispatch<React.SetStateAction<string | null>>;
  editingValue: string;
  setEditingValue: React.Dispatch<React.SetStateAction<string>>;
  isSavingEdit: boolean;
  copiedMessageId: string | null;
  canSend: (selectedImages: SelectedImage[]) => boolean;
  handleSubmit: (event: React.FormEvent, selectedImages: SelectedImage[]) => void;
  handleKeyDown: (event: React.KeyboardEvent<HTMLTextAreaElement>, selectedImages: SelectedImage[]) => void;
  handleEditUserMessage: (message: Message) => void;
  handleCancelEditUserMessage: () => void;
  handleSaveEditedUserMessage: (message: Message) => Promise<void>;
  handleCopyUserMessage: (message: Message) => Promise<void>;
  handleRetry: () => void;
  handleAskAI: (text: string) => void;
  handleClearQuote: () => void;
  handleInputFocus: () => void;
}

export function useMessageActions({
  agentId,
  sessionId,
  userId,
  isLoading,
  authenticatedFetch,
  setMessages,
  loadSessionHistory,
  loadSessions,
  sendMessage,
  textareaRef,
}: UseMessageActionsParams): UseMessageActionsResult {
  const [inputValue, setInputValue] = useState('');
  const [quotedText, setQuotedText] = useState('');
  const [editingMessageId, setEditingMessageId] = useState<string | null>(null);
  const [editingValue, setEditingValue] = useState('');
  const [isSavingEdit, setIsSavingEdit] = useState(false);
  const [copiedMessageId, setCopiedMessageId] = useState<string | null>(null);
  const copiedTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const buildMessageWithQuote = useCallback(
    (text: string) =>
      quotedText
        ? `> ${quotedText.split('\n').join('\n> ')}\n\n${text}`.trim()
        : text,
    [quotedText]
  );

  const canSend = useCallback(
    (selectedImages: SelectedImage[]) =>
      inputValue.trim().length > 0 || selectedImages.length > 0 || quotedText.length > 0,
    [inputValue, quotedText]
  );

  const handleSubmit = useCallback(
    (event: React.FormEvent, selectedImages: SelectedImage[]) => {
      event.preventDefault();
      const fullMessage = buildMessageWithQuote(inputValue);
      setQuotedText('');
      sendMessage(fullMessage, selectedImages);
    },
    [buildMessageWithQuote, inputValue, sendMessage]
  );

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>, selectedImages: SelectedImage[]) => {
      const isComposing = event.nativeEvent.isComposing || event.nativeEvent.keyCode === 229;

      if (
        event.key === 'Enter' &&
        !event.shiftKey &&
        !isComposing &&
        canSend(selectedImages)
      ) {
        event.preventDefault();
        const fullMessage = buildMessageWithQuote(inputValue);
        setQuotedText('');
        sendMessage(fullMessage, selectedImages);
      }
    },
    [buildMessageWithQuote, canSend, inputValue, sendMessage]
  );

  const handleEditUserMessage = useCallback(
    (message: Message) => {
      if (isLoading || isSavingEdit) return;

      if (message.id.startsWith('msg-')) {
        setMessages((prev) => [
          ...prev,
          createErrorMessage('这条消息还没有同步到历史记录。请刷新页面后再编辑。'),
        ]);
        return;
      }

      setEditingMessageId(message.id);
      setEditingValue(message.content || '');
    },
    [isLoading, isSavingEdit, setMessages]
  );

  const handleCancelEditUserMessage = useCallback(() => {
    if (isSavingEdit) return;
    setEditingMessageId(null);
    setEditingValue('');
  }, [isSavingEdit]);

  const handleSaveEditedUserMessage = useCallback(
    async (message: Message) => {
      if (isLoading || isSavingEdit) return;

      const trimmedEditedText = trimOuterNewlines(editingValue);
      const hasImages = (message.images?.length || 0) > 0;
      if (!trimmedEditedText && !hasImages) {
        setMessages((prev) => [...prev, createErrorMessage('编辑后的消息不能为空。')]);
        return;
      }

      try {
        setIsSavingEdit(true);

        const response = await authenticatedFetch(
          `/api/${agentId}/sessions/${sessionId}/edit`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify({
              user_id: userId,
              message_id: message.id,
              new_content: trimmedEditedText,
              images: message.images || [],
              file_names: message.fileNames || [],
            }),
          }
        );

        if (!response.ok) {
          let errorDetail = `HTTP error! status: ${response.status}`;
          try {
            const errorData = await response.json();
            if (errorData?.error) errorDetail += ` - ${errorData.error}`;
          } catch {
            // Keep status message when body is not JSON.
          }
          throw new Error(errorDetail);
        }

        const data = await response.json();
        const newSessionId = data?.new_session_id;
        if (!newSessionId) throw new Error('编辑成功但未返回新会话 ID');

        const editedImages: SelectedImage[] = (message.images || []).map((dataUrl, index) => ({
          id: `edited-${Date.now()}-${index}`,
          dataUrl,
          name: message.fileNames?.[index] || `文件 ${index + 1}`,
          mimeType: dataUrl.slice(5, dataUrl.indexOf(';')),
          isPdf: dataUrl.startsWith('data:application/pdf'),
          isAudio: dataUrl.startsWith('data:audio/'),
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
            '编辑消息失败，请稍后重试。\n\n错误详情：' +
              (error instanceof Error ? error.message : String(error))
          ),
        ]);
      } finally {
        setIsSavingEdit(false);
      }
    },
    [agentId, authenticatedFetch, editingValue, isLoading, isSavingEdit, loadSessionHistory, loadSessions, sendMessage, sessionId, setMessages, userId]
  );

  const handleCopyUserMessage = useCallback(async (message: Message) => {
    const content = message.content || '';
    if (!content.trim()) return;

    try {
      await navigator.clipboard.writeText(content);
      setCopiedMessageId(message.id);

      if (copiedTimeoutRef.current) clearTimeout(copiedTimeoutRef.current);
      copiedTimeoutRef.current = setTimeout(() => {
        setCopiedMessageId(null);
        copiedTimeoutRef.current = null;
      }, 1500);
    } catch (error) {
      console.error('Failed to copy user message:', error);
    }
  }, []);

  const handleRetry = useCallback(() => {
    sendMessage('', []);
  }, [sendMessage]);

  const handleAskAI = useCallback(
    (text: string) => {
      setQuotedText(text);
      setTimeout(() => textareaRef.current?.focus(), 0);
    },
    [textareaRef]
  );

  const handleClearQuote = useCallback(() => setQuotedText(''), []);

  const handleInputFocus = useCallback(() => {
    // Intentionally empty; keeps ChatInput props stable.
  }, []);

  return {
    inputValue,
    setInputValue,
    quotedText,
    setQuotedText,
    editingMessageId,
    setEditingMessageId,
    editingValue,
    setEditingValue,
    isSavingEdit,
    copiedMessageId,
    canSend,
    handleSubmit,
    handleKeyDown,
    handleEditUserMessage,
    handleCancelEditUserMessage,
    handleSaveEditedUserMessage,
    handleCopyUserMessage,
    handleRetry,
    handleAskAI,
    handleClearQuote,
    handleInputFocus,
  };
}
