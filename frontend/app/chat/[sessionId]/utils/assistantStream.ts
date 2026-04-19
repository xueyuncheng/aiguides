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
  let currentAuthor = '';
  let assistantContent = '';
  let assistantThought = '';
  let assistantImages: string[] = [];
  let assistantVideos: string[] = [];

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
          assistantImages = [...assistantImages, ...data.images.filter((item): item is string => typeof item === 'string')];

          setMessages((prev) => {
            const nextMessages = [...prev];
            const lastIndex = nextMessages.length - 1;

            if (lastIndex >= 0 && nextMessages[lastIndex].role === 'assistant') {
              nextMessages[lastIndex] = {
                ...nextMessages[lastIndex],
                images: assistantImages,
                isStreaming: true,
              };
            }

            return nextMessages;
          });
        }

        if (Array.isArray(data.videos)) {
          assistantVideos = [...assistantVideos, ...data.videos.filter((item): item is string => typeof item === 'string')];

          setMessages((prev) => {
            const nextMessages = [...prev];
            const lastIndex = nextMessages.length - 1;

            if (lastIndex >= 0 && nextMessages[lastIndex].role === 'assistant') {
              nextMessages[lastIndex] = {
                ...nextMessages[lastIndex],
                videos: assistantVideos,
                isStreaming: true,
              };
            }

            return nextMessages;
          });
        }

        if (typeof data.content !== 'string' || data.content.length === 0) {
          continue;
        }

        const isThought = Boolean(data.is_thought);
        const isCompleteDuplicate = !isThought && data.content === assistantContent;
        if (isCompleteDuplicate) {
          continue;
        }

        if (typeof data.author === 'string' && data.author !== currentAuthor) {
          currentAuthor = data.author;
          assistantContent = isThought ? '' : data.content;
          assistantThought = isThought ? data.content : '';
          assistantImages = [];
          assistantVideos = [];

          setMessages((prev) => [
            ...prev,
            {
              id: `msg-${Date.now()}-${currentAuthor}`,
              role: 'assistant',
              content: assistantContent,
              thought: assistantThought,
              timestamp: new Date(),
              author: currentAuthor,
              isStreaming: true,
              images: [],
              videos: [],
            },
          ]);
          continue;
        }

        if (isThought) {
          assistantThought += data.content;
        } else {
          assistantContent += data.content;
        }

        setMessages((prev) => {
          const nextMessages = [...prev];
          const lastIndex = nextMessages.length - 1;

          if (lastIndex >= 0 && nextMessages[lastIndex].role === 'assistant') {
            nextMessages[lastIndex] = {
              ...nextMessages[lastIndex],
              content: assistantContent,
              thought: assistantThought,
              images: assistantImages,
              videos: assistantVideos,
              isStreaming: true,
            };
          }

          return nextMessages;
        });
      } catch (error) {
        console.warn('JSON parse error:', error);
      }
    }
  }

  setMessages((prev) => prev.map((message) => ({ ...message, isStreaming: false })));
}
