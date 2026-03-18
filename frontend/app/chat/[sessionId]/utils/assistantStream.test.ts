import { describe, expect, it, vi } from 'vitest';
import { consumeAssistantStream } from './assistantStream';
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
      'data: {"tool_name":"web_search","tool_label":"Search","tool_args":{"q":"test"},"author":"assistant"}\n',
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
          toolName: 'web_search',
          label: 'Search',
          args: { q: 'test' },
        },
      ],
    });
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
