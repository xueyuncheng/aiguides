import type { Dispatch, SetStateAction } from 'react';
import type { Message, ToolCallItem } from '../types';
import { mergeToolCalls } from './messages';

const serializeToolCall = (toolCall: { toolCallId?: string; toolName: string; label: string; args?: Record<string, unknown> }) => (
  JSON.stringify({
    toolCallId: toolCall.toolCallId || null,
    toolName: toolCall.toolName,
    label: toolCall.label,
    args: toolCall.args || null,
  })
);

const updateMatchingToolCall = (
  messages: Message[],
  matcher: (toolCall: ToolCallItem) => boolean,
  updater: (toolCall: ToolCallItem) => ToolCallItem
) => {
  for (let i = messages.length - 1; i >= 0; i -= 1) {
    const message = messages[i];
    if (!message || message.role !== 'assistant' || !message.toolCalls || message.toolCalls.length === 0) {
      continue;
    }

    const toolCallIndex = message.toolCalls.findLastIndex(matcher);
    if (toolCallIndex === -1) {
      continue;
    }

    const nextToolCalls = [...message.toolCalls];
    nextToolCalls[toolCallIndex] = updater(nextToolCalls[toolCallIndex]);
    messages[i] = {
      ...message,
      toolCalls: nextToolCalls,
      isStreaming: true,
    };
    return true;
  }

  return false;
};

interface ConsumeAssistantStreamParams {
  reader: ReadableStreamDefaultReader<Uint8Array>;
  setIsLoading: Dispatch<SetStateAction<boolean>>;
  setMessages: Dispatch<SetStateAction<Message[]>>;
}

export const createAssistantErrorMessage = (content: string): Message => ({
  id: `msg-${Date.now()}-error`,
  role: 'assistant',
  content,
  timestamp: new Date(),
  isError: true,
});

export async function consumeAssistantStream({
  reader,
  setIsLoading,
  setMessages,
}: ConsumeAssistantStreamParams) {
  const decoder = new TextDecoder();
  let buffer = '';
  let currentEventType = 'data';

  // Per-author state tracking to support parallel agents without interleaving.
  interface AuthorState {
    content: string;
    thought: string;
    images: string[];
    videos: string[];
    messageId: string;
  }
  const authorStates = new Map<string, AuthorState>();

  const getOrCreateAuthor = (author: string): AuthorState => {
    let state = authorStates.get(author);
    if (state) return state;

    const messageId = `msg-${Date.now()}-${author || 'assistant'}`;
    state = { content: '', thought: '', images: [], videos: [], messageId };
    authorStates.set(author, state);
    setMessages((prev) => [
      ...prev,
      {
        id: messageId,
        role: 'assistant',
        content: '',
        thought: '',
        timestamp: new Date(),
        author: author || undefined,
        isStreaming: true,
        images: [],
        videos: [],
      },
    ]);
    return state;
  };

  const updateAuthorMessage = (state: AuthorState) => {
    setMessages((prev) => {
      const nextMessages = [...prev];
      const idx = nextMessages.findLastIndex((m) => m.id === state.messageId);
      if (idx >= 0) {
        nextMessages[idx] = {
          ...nextMessages[idx],
          content: state.content,
          thought: state.thought,
          images: state.images,
          videos: state.videos,
          isStreaming: true,
        };
      }
      return nextMessages;
    });
  };

  const resetEventType = () => {
    currentEventType = 'data';
  };

  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';

    for (const line of lines) {
      const trimmedLine = line.trim();

      if (trimmedLine.startsWith('event:')) {
        currentEventType = trimmedLine.substring(6).trim();
        continue;
      }

      if (!trimmedLine.startsWith('data:')) {
        continue;
      }

      try {
        const jsonStr = trimmedLine.substring(5).trim();
        if (!jsonStr) {
          continue;
        }

        const data = JSON.parse(jsonStr) as Record<string, unknown>;

        if (currentEventType === 'heartbeat') {
          resetEventType();
          continue;
        }

        if (currentEventType === 'error') {
          const errorText = typeof data.error === 'string' ? data.error : '发生未知错误，请稍后重试。';
          setMessages((prev) => [...prev, createAssistantErrorMessage(`❌ **错误**: ${errorText}`)]);
          setIsLoading(false);
          resetEventType();
          continue;
        }

        if (currentEventType === 'stop') {
          resetEventType();
          continue;
        }

        if (currentEventType === 'tool_call') {
          const toolCall = {
            toolCallId: typeof data.tool_call_id === 'string' ? data.tool_call_id : undefined,
            toolName: typeof data.tool_name === 'string' ? data.tool_name : '',
            label: typeof data.tool_label === 'string' ? data.tool_label : '',
            args: (data.tool_args as Record<string, unknown> | undefined) || undefined,
            status: 'running' as const,
          };
          const toolCallKey = serializeToolCall(toolCall);

          setMessages((prev) => {
            const nextMessages = [...prev];
            const lastIndex = nextMessages.length - 1;

            if (lastIndex >= 0 && nextMessages[lastIndex].role === 'assistant') {
              const existingToolCalls = nextMessages[lastIndex].toolCalls || [];
              const hasDuplicateToolCall = existingToolCalls.some((existingToolCall) => (
                (toolCall.toolCallId && existingToolCall.toolCallId === toolCall.toolCallId) ||
                serializeToolCall(existingToolCall) === toolCallKey
              ));
              if (hasDuplicateToolCall) {
                nextMessages[lastIndex] = {
                  ...nextMessages[lastIndex],
                  isStreaming: true,
                };
                return nextMessages;
              }

              nextMessages[lastIndex] = {
                ...nextMessages[lastIndex],
                toolCalls: mergeToolCalls([...existingToolCalls, toolCall]),
                isStreaming: true,
              };
            } else {
              nextMessages.push({
                id: `msg-${Date.now()}-${typeof data.author === 'string' ? data.author : 'assistant'}`,
                role: 'assistant',
                content: '',
                timestamp: new Date(),
                author: typeof data.author === 'string' ? data.author : undefined,
                isStreaming: true,
                toolCalls: [toolCall],
              });
            }

            return nextMessages;
          });

          resetEventType();
          continue;
        }

        if (currentEventType === 'tool_result') {
          const toolCallId = typeof data.tool_call_id === 'string' ? data.tool_call_id : undefined;
          const toolName = typeof data.tool_name === 'string' ? data.tool_name : undefined;
          const toolResult = (data.tool_result as Record<string, unknown> | undefined) || undefined;

          setMessages((prev) => {
            const nextMessages = [...prev];
            const matched = updateMatchingToolCall(
              nextMessages,
              (toolCall) => (
                (toolCallId && toolCall.toolCallId === toolCallId) ||
                (!toolCallId && toolName ? toolCall.toolName === toolName && toolCall.status !== 'completed' : false)
              ),
              (toolCall) => ({
                ...toolCall,
                result: toolResult || toolCall.result,
                status: 'completed',
              })
            );

            return matched ? nextMessages : prev;
          });

          resetEventType();
          continue;
        }

        if (currentEventType === 'tool_progress') {
          const toolCallId = typeof data.tool_call_id === 'string' ? data.tool_call_id : undefined;
          const toolName = typeof data.tool_name === 'string' ? data.tool_name : undefined;
          const toolResult = (data.tool_result as Record<string, unknown> | undefined) || undefined;

          setMessages((prev) => {
            const nextMessages = [...prev];
            const matched = updateMatchingToolCall(
              nextMessages,
              (toolCall) => (
                (toolCallId && toolCall.toolCallId === toolCallId)
                || (!toolCallId && toolName ? toolCall.toolName === toolName && toolCall.status !== 'completed' : false)
              ),
              (toolCall) => ({
                ...toolCall,
                result: {
                  ...(toolCall.result || {}),
                  ...(toolResult || {}),
                },
                status: 'running',
              })
            );

            return matched ? nextMessages : prev;
          });

          resetEventType();
          continue;
        }

        if (Array.isArray(data.images)) {
          const eventAuthor = typeof data.author === 'string' ? data.author : '';
          const state = getOrCreateAuthor(eventAuthor);
          state.images = [...state.images, ...data.images.filter((item): item is string => typeof item === 'string')];
          updateAuthorMessage(state);
        }

        if (Array.isArray(data.videos)) {
          const eventAuthor = typeof data.author === 'string' ? data.author : '';
          const state = getOrCreateAuthor(eventAuthor);
          state.videos = [...state.videos, ...data.videos.filter((item): item is string => typeof item === 'string')];
          updateAuthorMessage(state);
        }

        if (typeof data.content !== 'string' || data.content.length === 0) {
          continue;
        }

        const isThought = Boolean(data.is_thought);
        const eventAuthor = typeof data.author === 'string' ? data.author : '';
        const state = getOrCreateAuthor(eventAuthor);

        const isCompleteDuplicate = !isThought && data.content === state.content;
        if (isCompleteDuplicate) {
          continue;
        }

        if (isThought) {
          state.thought += data.content;
        } else {
          state.content += data.content;
        }

        updateAuthorMessage(state);
      } catch (error) {
        console.warn('JSON parse error:', error);
      }
    }
  }

  setMessages((prev) => prev.map((message) => ({ ...message, isStreaming: false })));
}
