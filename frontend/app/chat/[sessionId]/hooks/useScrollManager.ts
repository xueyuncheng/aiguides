import { useCallback, useEffect, useRef, useState } from 'react';
import { MAX_TEXTAREA_HEIGHT } from '../constants';
import type { Message } from '../types';

const DEFAULT_COMPOSER_OFFSET = 160;

interface UseScrollManagerParams {
  messages: Message[];
  isStreamingResponse: boolean;
  latestUserMessageId: string | undefined;
  isLoadingHistory: boolean;
  inputValue: string;
  shouldScrollInstantly: boolean;
  shouldLoadOlderMessages: (scrollTop: number) => boolean;
  loadOlderMessages: () => void;
  textareaRef: React.RefObject<HTMLTextAreaElement | null>;
  scrollContainerRef: React.RefObject<HTMLDivElement | null>;
}

interface UseScrollManagerResult {
  messagesEndRef: React.RefObject<HTMLDivElement | null>;
  latestUserMessageRef: React.RefObject<HTMLDivElement | null>;
  chatInputContainerRef: React.RefObject<HTMLDivElement | null>;
  chatInputOffset: number;
  isAtBottomRef: React.MutableRefObject<boolean>;
  handleScroll: () => void;
}

export function useScrollManager({
  messages,
  isStreamingResponse,
  latestUserMessageId,
  isLoadingHistory,
  inputValue,
  shouldScrollInstantly,
  shouldLoadOlderMessages,
  loadOlderMessages,
  textareaRef,
  scrollContainerRef,
}: UseScrollManagerParams): UseScrollManagerResult {
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const latestUserMessageRef = useRef<HTMLDivElement>(null);
  const chatInputContainerRef = useRef<HTMLDivElement>(null);
  const isAtBottomRef = useRef(true);
  const [chatInputOffset, setChatInputOffset] = useState(DEFAULT_COMPOSER_OFFSET);

  const scrollElementAboveComposer = useCallback(
    (element: HTMLDivElement, behavior: ScrollBehavior) => {
      const container = scrollContainerRef.current;
      if (!container) {
        element.scrollIntoView({ behavior, block: 'end' });
        return;
      }

      const elementRect = element.getBoundingClientRect();
      const containerRect = container.getBoundingClientRect();
      const offsetWithinContainer = elementRect.bottom - containerRect.top;
      const visibleHeight = container.clientHeight - chatInputOffset;
      const targetScrollTop = container.scrollTop + offsetWithinContainer - visibleHeight;

      container.scrollTo({
        top: Math.max(0, targetScrollTop),
        behavior,
      });
    },
    [chatInputOffset, scrollContainerRef]
  );

  // Auto-scroll when messages change
  useEffect(() => {
    const lastMessage = messages[messages.length - 1];
    if (!lastMessage) return;

    const latestUserMessageElement = latestUserMessageRef.current;
    if (lastMessage.role === 'user') {
      isAtBottomRef.current = true;
      if (latestUserMessageElement) {
        scrollElementAboveComposer(
          latestUserMessageElement,
          shouldScrollInstantly ? 'auto' : 'smooth'
        );
      }
      return;
    }

    if (lastMessage.isStreaming) {
      // Stop auto-scrolling if the user has manually scrolled up.
      // isAtBottomRef is kept up-to-date by handleScroll (scroll event listener).
      if (!isAtBottomRef.current) return;

      if (messagesEndRef.current) {
        scrollElementAboveComposer(messagesEndRef.current, 'auto');
      }
      return;
    }

    if (isAtBottomRef.current) {
      if (messagesEndRef.current) {
        scrollElementAboveComposer(
          messagesEndRef.current,
          shouldScrollInstantly ? 'auto' : 'smooth'
        );
      }
    }
  }, [chatInputOffset, isStreamingResponse, latestUserMessageId, messages, scrollContainerRef, scrollElementAboveComposer, shouldScrollInstantly]);

  // Focus textarea when chat is empty
  useEffect(() => {
    if (messages.length === 0 && !isLoadingHistory) {
      textareaRef.current?.focus();
    }
  }, [isLoadingHistory, messages.length, textareaRef]);

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;
    textarea.style.height = 'auto';
    textarea.style.height = `${Math.min(textarea.scrollHeight, MAX_TEXTAREA_HEIGHT)}px`;
  }, [inputValue, textareaRef]);

  // Track chat input height via ResizeObserver
  useEffect(() => {
    const chatInputElement = chatInputContainerRef.current;
    if (!chatInputElement) return;

    const updateChatInputOffset = () => {
      const nextOffset = Math.ceil(chatInputElement.getBoundingClientRect().height) + 8;
      setChatInputOffset((prev) => (prev === nextOffset ? prev : nextOffset));
    };

    updateChatInputOffset();

    if (typeof ResizeObserver === 'undefined') {
      window.addEventListener('resize', updateChatInputOffset);
      return () => window.removeEventListener('resize', updateChatInputOffset);
    }

    const resizeObserver = new ResizeObserver(updateChatInputOffset);
    resizeObserver.observe(chatInputElement);
    window.addEventListener('resize', updateChatInputOffset);

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener('resize', updateChatInputOffset);
    };
  }, []);

  const handleScroll = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const { scrollTop, scrollHeight, clientHeight } = container;
    const bottomThreshold = Math.max(10, chatInputOffset + 10);
    isAtBottomRef.current = scrollHeight - scrollTop - clientHeight < bottomThreshold;

    if (shouldLoadOlderMessages(scrollTop)) {
      loadOlderMessages();
    }
  }, [chatInputOffset, loadOlderMessages, scrollContainerRef, shouldLoadOlderMessages]);

  return {
    messagesEndRef,
    latestUserMessageRef,
    chatInputContainerRef,
    chatInputOffset,
    isAtBottomRef,
    handleScroll,
  };
}
