import type { HistoryMessageResponse, Message, ToolCallItem, ToolCallResponse } from '../types';
import { DEEP_RESEARCH_AGENTS } from '../constants';

export const trimOuterNewlines = (value: string) => value.replace(/^[\n\r]+|[\n\r]+$/g, '');

export const mapToolCall = (toolCall: ToolCallResponse) => ({
  toolCallId: toolCall.tool_call_id || undefined,
  toolName: toolCall.tool_name,
  label: toolCall.label,
  args: toolCall.args || undefined,
  result: toolCall.result || undefined,
  status: 'completed' as const,
});

const getToolCallKey = (toolCall: ToolCallItem) => JSON.stringify({
  toolCallId: toolCall.toolCallId || null,
  toolName: toolCall.toolName,
  label: toolCall.label,
  args: toolCall.args || null,
});

export const mergeToolCalls = (toolCalls: ToolCallItem[]) => {
  const merged = new Map<string, ToolCallItem>();

  toolCalls.forEach((toolCall) => {
    const key = getToolCallKey(toolCall);
    const existing = merged.get(key);
    if (!existing) {
      merged.set(key, toolCall);
      return;
    }

    merged.set(key, {
      ...existing,
      ...toolCall,
      result: toolCall.result || existing.result,
      status: toolCall.status === 'completed' || existing.status === 'completed' ? 'completed' : 'running',
    });
  });

  return [...merged.values()];
};

export const mapHistoryMessage = (message: HistoryMessageResponse): Message => ({
  id: message.id,
  role: message.role,
  author: message.author || undefined,
  content: message.content,
  thought: message.thought,
  timestamp: new Date(message.timestamp),
  images: message.images || [],
  videos: message.videos || [],
  fileNames: message.file_names || [],
  files: message.files || [],
  toolCalls: (message.tool_calls || []).map(mapToolCall),
  voiceAudioFileId: message.voice_audio_file_id || undefined,
});

const isDeepResearchAgent = (author?: string) =>
  author ? DEEP_RESEARCH_AGENTS.has(author) : false;

const hasOnlyThought = (msg: Message) =>
  !msg.content && !!msg.thought && (!msg.toolCalls || msg.toolCalls.length === 0);

const canMergeAssistantMessages = (a: Message, b: Message) => {
  if (a.role !== 'assistant' || b.role !== 'assistant') return false;
  if (a.isError || b.isError) return false;
  if ((a.author || '') === (b.author || '')) return true;
  if (isDeepResearchAgent(a.author) && isDeepResearchAgent(b.author)) return true;
  // Merge a thought-only delegation message into the following research block.
  if (hasOnlyThought(a) && isDeepResearchAgent(b.author)) return true;
  return false;
};

export const mergeAssistantMessages = (messages: Message[]) => {
  if (messages.length === 0) {
    return [];
  }

  const mergedMessages: Message[] = [];

  messages.forEach((message) => {
    const lastMessage = mergedMessages[mergedMessages.length - 1];

    if (lastMessage && canMergeAssistantMessages(lastMessage, message)) {
      mergedMessages[mergedMessages.length - 1] = {
        ...lastMessage,
        content: (lastMessage.content || '') + (message.content || ''),
        thought: message.thought
          ? (lastMessage.thought || '') + ((lastMessage.thought || '') ? '\n\n' : '') + message.thought
          : lastMessage.thought,
        images: [...(lastMessage.images || []), ...(message.images || [])],
        videos: [...(lastMessage.videos || []), ...(message.videos || [])],
        fileNames: [...(lastMessage.fileNames || []), ...(message.fileNames || [])],
        toolCalls: mergeToolCalls([...(lastMessage.toolCalls || []), ...(message.toolCalls || [])]),
        author: message.author || lastMessage.author,
        isStreaming: lastMessage.isStreaming || message.isStreaming,
      };
      return;
    }

    mergedMessages.push(message);
  });

  return mergedMessages;
};
