import { describe, expect, it } from 'vitest';
import type { Message } from '../types';
import { mergeAssistantMessages } from './messages';

const createMessage = (overrides: Partial<Message>): Message => ({
  id: 'message',
  role: 'assistant',
  content: '',
  timestamp: new Date('2026-01-01T00:00:00Z'),
  ...overrides,
});

describe('mergeAssistantMessages', () => {
  it('merges consecutive assistant messages from different agents into one response', () => {
    const messages = [
      createMessage({ id: 'root', author: 'assistant', thought: 'Root thought' }),
      createMessage({ id: 'web', author: 'web_agent', thought: 'Web thought' }),
      createMessage({ id: 'final', author: 'assistant', content: 'Final answer' }),
    ];

    expect(mergeAssistantMessages(messages)).toEqual([
      expect.objectContaining({
        id: 'root',
        content: 'Final answer',
        thought: 'Root thought\n\nWeb thought',
      }),
    ]);
  });

  it('starts a new response after a user message', () => {
    const messages = [
      createMessage({ id: 'first', thought: 'First thought' }),
      createMessage({ id: 'user', role: 'user', content: 'Next question' }),
      createMessage({ id: 'second', thought: 'Second thought' }),
    ];

    expect(mergeAssistantMessages(messages).map((message) => message.id)).toEqual([
      'first',
      'user',
      'second',
    ]);
  });
});
