import { describe, expect, it, vi } from 'vitest';
import { consumeAssistantStream } from './assistantStream';
import { mergeAssistantMessages } from './messages';
import type { Message } from '../types';

const createReader = (chunks: string[]) => {
  const encoder = new TextEncoder();
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      chunks.forEach((chunk) => controller.enqueue(encoder.encode(chunk)));
      controller.close();
    },
  });

  return stream.getReader();
};

const createMessageState = () => {
  let messages: Message[] = [];
  const setMessages = vi.fn((updater: React.SetStateAction<Message[]>) => {
    messages = typeof updater === 'function' ? updater(messages) : updater;
  });

  return {
    getMessages: () => messages,
    setMessages,
  };
};

describe('consumeAssistantStream', () => {
  it('parses content and tool calls then marks messages as complete', async () => {
    const reader = createReader([
      'data: {"author":"assistant","content":"Hello "}\n',
      'event: tool_call\n',
      'data: {"tool_call_id":"call-1","tool_name":"web_search","tool_label":"Search","tool_args":{"q":"test"},"author":"assistant"}\n',
      'event: tool_result\n',
      'data: {"tool_call_id":"call-1","tool_name":"web_search","tool_result":{"items":1},"author":"assistant"}\n',
      'data: {"author":"assistant","content":"world"}\n',
    ]);
    const { getMessages, setMessages } = createMessageState();
    const setIsLoading = vi.fn();

    await consumeAssistantStream({
      reader,
      setIsLoading,
      setMessages,
    });

    expect(setIsLoading).not.toHaveBeenCalled();
    expect(getMessages()).toHaveLength(1);
    expect(getMessages()[0]).toMatchObject({
      role: 'assistant',
      content: 'Hello world',
      author: 'assistant',
      isStreaming: false,
      toolCalls: [
        {
          toolCallId: 'call-1',
          toolName: 'web_search',
          label: 'Search',
          args: { q: 'test' },
          result: { items: 1 },
          status: 'completed',
        },
      ],
    });
  });

  it('deduplicates repeated tool call events for the same assistant message', async () => {
    const reader = createReader([
      'event: tool_call\n',
      'data: {"tool_call_id":"call-2","tool_name":"web_fetch","tool_label":"Fetch page","tool_args":{"url":"https://example.com"},"author":"assistant"}\n',
      'event: tool_call\n',
      'data: {"tool_call_id":"call-2","tool_name":"web_fetch","tool_label":"Fetch page","tool_args":{"url":"https://example.com"},"author":"assistant"}\n',
      'event: tool_result\n',
      'data: {"tool_call_id":"call-2","tool_name":"web_fetch","tool_result":{"ok":true},"author":"assistant"}\n',
      'data: {"author":"assistant","content":"done"}\n',
    ]);
    const { getMessages, setMessages } = createMessageState();
    const setIsLoading = vi.fn();

    await consumeAssistantStream({
      reader,
      setIsLoading,
      setMessages,
    });

    const mergedMessages = mergeAssistantMessages(getMessages());

    expect(setIsLoading).not.toHaveBeenCalled();
    expect(mergedMessages).toHaveLength(1);
    expect(mergedMessages[0]).toMatchObject({
      role: 'assistant',
      content: 'done',
      toolCalls: [
        {
          toolCallId: 'call-2',
          toolName: 'web_fetch',
          label: 'Fetch page',
          args: { url: 'https://example.com' },
          result: { ok: true },
          status: 'completed',
        },
      ],
    });
  });

  it('marks a running tool call as completed when tool_result arrives', async () => {
    const reader = createReader([
      'event: tool_call\n',
      'data: {"tool_call_id":"call-3","tool_name":"web_fetch","tool_label":"Fetch page","tool_args":{"url":"https://example.com"},"author":"assistant"}\n',
      'event: tool_result\n',
      'data: {"tool_call_id":"call-3","tool_name":"web_fetch","tool_result":{"title":"Example"},"author":"assistant"}\n',
    ]);
    const { getMessages, setMessages } = createMessageState();
    const setIsLoading = vi.fn();

    await consumeAssistantStream({
      reader,
      setIsLoading,
      setMessages,
    });

    expect(getMessages()).toHaveLength(1);
    expect(getMessages()[0]).toMatchObject({
      toolCalls: [
        {
          toolCallId: 'call-3',
          toolName: 'web_fetch',
          status: 'completed',
          result: { title: 'Example' },
        },
      ],
      isStreaming: false,
    });
  });

  it('updates a running audio transcription tool call from tool_progress events', async () => {
    const reader = createReader([
      'event: tool_call\n',
      'data: {"tool_call_id":"call-audio","tool_name":"audio_transcribe","tool_label":"Transcribe audio","tool_args":{"file_id":42},"author":"assistant"}\n',
      'event: tool_progress\n',
      'data: {"tool_name":"audio_transcribe","tool_result":{"job_id":9,"chunk_count":3,"completed_chunks":1,"transcript":"Hello"},"author":"assistant"}\n',
      'event: tool_progress\n',
      'data: {"tool_name":"audio_transcribe","tool_result":{"job_id":9,"chunk_count":3,"completed_chunks":2,"transcript":"Hello world"},"author":"assistant"}\n',
    ]);
    const { getMessages, setMessages } = createMessageState();
    const setIsLoading = vi.fn();

    await consumeAssistantStream({
      reader,
      setIsLoading,
      setMessages,
    });

    expect(setIsLoading).not.toHaveBeenCalled();
    expect(getMessages()).toHaveLength(1);
    expect(getMessages()[0]).toMatchObject({
      toolCalls: [
        {
          toolCallId: 'call-audio',
          toolName: 'audio_transcribe',
          status: 'running',
          result: {
            job_id: 9,
            chunk_count: 3,
            completed_chunks: 2,
            transcript: 'Hello world',
          },
        },
      ],
      isStreaming: false,
    });
  });

  it('deduplicates repeated batches of tool calls within the same assistant message', async () => {
    const reader = createReader([
      'event: tool_call\n',
      'data: {"tool_call_id":"call-a","tool_name":"web_search","tool_label":"Search A","tool_args":{"q":"a"},"author":"assistant"}\n',
      'event: tool_call\n',
      'data: {"tool_call_id":"call-b","tool_name":"web_search","tool_label":"Search B","tool_args":{"q":"b"},"author":"assistant"}\n',
      'event: tool_call\n',
      'data: {"tool_call_id":"call-c","tool_name":"web_search","tool_label":"Search C","tool_args":{"q":"c"},"author":"assistant"}\n',
      'event: tool_call\n',
      'data: {"tool_call_id":"call-a","tool_name":"web_search","tool_label":"Search A","tool_args":{"q":"a"},"author":"assistant"}\n',
      'event: tool_call\n',
      'data: {"tool_call_id":"call-b","tool_name":"web_search","tool_label":"Search B","tool_args":{"q":"b"},"author":"assistant"}\n',
      'event: tool_call\n',
      'data: {"tool_call_id":"call-c","tool_name":"web_search","tool_label":"Search C","tool_args":{"q":"c"},"author":"assistant"}\n',
    ]);
    const { getMessages, setMessages } = createMessageState();
    const setIsLoading = vi.fn();

    await consumeAssistantStream({
      reader,
      setIsLoading,
      setMessages,
    });

    expect(getMessages()).toHaveLength(1);
    expect(getMessages()[0].toolCalls).toHaveLength(3);
    expect(getMessages()[0].toolCalls?.map((toolCall) => toolCall.toolCallId)).toEqual(['call-a', 'call-b', 'call-c']);
  });

  it('appends an error message for error events', async () => {
    const reader = createReader([
      'event: error\n',
      'data: {"error":"boom"}\n',
    ]);
    const { getMessages, setMessages } = createMessageState();
    const setIsLoading = vi.fn();

    await consumeAssistantStream({
      reader,
      setIsLoading,
      setMessages,
    });

    expect(setIsLoading).toHaveBeenCalledWith(false);
    expect(getMessages()).toHaveLength(1);
    expect(getMessages()[0]).toMatchObject({
      role: 'assistant',
      isError: true,
      content: '❌ **错误**: boom',
    });
  });
});
