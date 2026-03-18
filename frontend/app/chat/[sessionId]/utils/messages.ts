import type { HistoryMessageResponse, Message, ToolCallResponse } from '../types';

export const trimOuterNewlines = (value: string) => value.replace(/^[\n\r]+|[\n\r]+$/g, '');

export const mapToolCall = (toolCall: ToolCallResponse) => ({
  toolName: toolCall.tool_name,
  label: toolCall.label,
  args: toolCall.args || undefined,
  result: toolCall.result || undefined,
});

export const mapHistoryMessage = (message: HistoryMessageResponse): Message => ({
  id: message.id,
  role: message.role,
  content: message.content,
  thought: message.thought,
  timestamp: new Date(message.timestamp),
  images: message.images || [],
  fileNames: message.file_names || [],
  files: message.files || [],
  toolCalls: (message.tool_calls || []).map(mapToolCall),
});

export const mergeAssistantMessages = (messages: Message[]) => {
  if (messages.length === 0) {
    return [];
  }

  const mergedMessages: Message[] = [];

  messages.forEach((message) => {
    const lastMessage = mergedMessages[mergedMessages.length - 1];

    if (
      lastMessage &&
      lastMessage.role === 'assistant' &&
      message.role === 'assistant' &&
      !lastMessage.isError &&
      !message.isError
    ) {
      mergedMessages[mergedMessages.length - 1] = {
        ...lastMessage,
        content: (lastMessage.content || '') + (message.content || ''),
        thought: message.thought
          ? (lastMessage.thought || '') + ((lastMessage.thought || '') ? '\n\n' : '') + message.thought
          : lastMessage.thought,
        images: [...(lastMessage.images || []), ...(message.images || [])],
        fileNames: [...(lastMessage.fileNames || []), ...(message.fileNames || [])],
        toolCalls: [...(lastMessage.toolCalls || []), ...(message.toolCalls || [])],
        author: message.author || lastMessage.author,
        isStreaming: lastMessage.isStreaming || message.isStreaming,
      };
      return;
    }

    mergedMessages.push(message);
  });

  return mergedMessages;
};
