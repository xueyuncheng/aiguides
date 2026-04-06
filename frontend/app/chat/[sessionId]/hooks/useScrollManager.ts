import { useCallback, useEffect, useRef, useState } from 'react';
import { COMPOSER_MESSAGE_GAP, MAX_TEXTAREA_HEIGHT } from '../constants';
import type { Message } from '../types';

const DEFAULT_COMPOSER_OFFSET = 160;

interface UseScrollManagerParams {
  messages: Message[];
  isStreamingResponse: boolean;
  isLoading: boolean;
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
  chatInputContainerRef: React.RefCallback<HTMLDivElement>;
  chatInputOffset: number;
  isAtBottomRef: React.MutableRefObject<boolean>;
  handleScroll: () => void;
}

export function useScrollManager({
  messages,
  isStreamingResponse,
  isLoading,
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
  const chatInputContainerElementRef = useRef<HTMLDivElement | null>(null);
  const [chatInputElement, setChatInputElement] = useState<HTMLDivElement | null>(null);
  const isAtBottomRef = useRef(true);
  const autoScrollTargetRef = useRef<number | null>(null);
  const lastObservedScrollTopRef = useRef(0);
  const [chatInputOffset, setChatInputOffset] = useState(DEFAULT_COMPOSER_OFFSET);

  const chatInputContainerRef = useCallback((node: HTMLDivElement | null) => {
    chatInputContainerElementRef.current = node;
    setChatInputElement((prev) => {
      if (prev === node) {
        return prev;
      }

      return node;
    });
  }, []);

  useEffect(() => {
    if (!chatInputElement && chatInputContainerElementRef.current) {
      setChatInputElement(chatInputContainerElementRef.current);
    }
  }, [chatInputElement]);

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
      const visibleHeight = container.clientHeight - chatInputOffset - COMPOSER_MESSAGE_GAP;
      const targetScrollTop = Math.max(0, container.scrollTop + offsetWithinContainer - visibleHeight);

      autoScrollTargetRef.current = targetScrollTop;
      lastObservedScrollTopRef.current = container.scrollTop;

      container.scrollTo({
        top: targetScrollTop,
        behavior,
      });
    },
    [chatInputOffset, scrollContainerRef]
  );

  // Auto-scroll when messages change
  useEffect(() => {
    if (isLoadingHistory) return;

    const lastMessage = messages[messages.length - 1];
    if (!lastMessage) return;

    const latestUserMessageElement = latestUserMessageRef.current;
    if (lastMessage.role === 'user') {
      isAtBottomRef.current = true;
      if (isLoading && messagesEndRef.current) {
        scrollElementAboveComposer(
          messagesEndRef.current,
          shouldScrollInstantly ? 'auto' : 'smooth'
        );
      } else if (latestUserMessageElement) {
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
  }, [chatInputOffset, isLoading, isLoadingHistory, isStreamingResponse, latestUserMessageId, messages, scrollContainerRef, scrollElementAboveComposer, shouldScrollInstantly]);

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
  }, [chatInputElement]);

  const handleScroll = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const { scrollTop, scrollHeight, clientHeight } = container;
    const autoScrollTarget = autoScrollTargetRef.current;

    if (autoScrollTarget !== null) {
      const previousScrollTop = lastObservedScrollTopRef.current;
      const hasReachedTarget = Math.abs(scrollTop - autoScrollTarget) <= 2;
      const isMovingTowardTarget = autoScrollTarget >= previousScrollTop
        ? scrollTop >= previousScrollTop && scrollTop <= autoScrollTarget + 2
        : scrollTop <= previousScrollTop && scrollTop >= autoScrollTarget - 2;

      lastObservedScrollTopRef.current = scrollTop;

      if (hasReachedTarget) {
        autoScrollTargetRef.current = null;
        return;
      } else if (isMovingTowardTarget) {
        return;
      } else {
        autoScrollTargetRef.current = null;
      }
    } else {
      lastObservedScrollTopRef.current = scrollTop;
    }

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
