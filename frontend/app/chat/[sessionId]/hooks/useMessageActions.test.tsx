import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useMessageActions } from './useMessageActions';

describe('useMessageActions', () => {
  const createHook = () => {
    const sendMessage = vi.fn().mockResolvedValue(undefined);

    const hook = renderHook(() => useMessageActions({
      agentId: 'assistant',
      sessionId: 'session-1',
      userId: 'user-1',
      isLoading: false,
      authenticatedFetch: vi.fn(),
      setMessages: vi.fn(),
      loadSessionHistory: vi.fn().mockResolvedValue(undefined),
      loadSessions: vi.fn().mockResolvedValue(undefined),
      sendMessage,
      textareaRef: { current: null },
    }));

    return { ...hook, sendMessage };
  };

  it('does not send on Enter while IME confirmation is in progress', async () => {
    const { result, sendMessage } = createHook();
    const preventDefault = vi.fn();

    await act(async () => {
      result.current.setInputValue('nihao');
    });

    await act(async () => {
      result.current.handleKeyDown({
        key: 'Enter',
        shiftKey: false,
        preventDefault,
        nativeEvent: {
          isComposing: false,
          keyCode: 229,
        },
      } as React.KeyboardEvent<HTMLTextAreaElement>, []);
    });

    expect(preventDefault).not.toHaveBeenCalled();
    expect(sendMessage).not.toHaveBeenCalled();
  });

  it('sends on plain Enter when not composing', async () => {
    const { result, sendMessage } = createHook();
    const preventDefault = vi.fn();

    await act(async () => {
      result.current.setInputValue('hello');
    });

    await act(async () => {
      result.current.handleKeyDown({
        key: 'Enter',
        shiftKey: false,
        preventDefault,
        nativeEvent: {
          isComposing: false,
          keyCode: 13,
        },
      } as React.KeyboardEvent<HTMLTextAreaElement>, []);
    });

    expect(preventDefault).toHaveBeenCalledTimes(1);
    expect(sendMessage).toHaveBeenCalledWith('hello', []);
  });
});
