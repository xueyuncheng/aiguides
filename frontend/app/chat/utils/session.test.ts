import { describe, expect, it, vi } from 'vitest';
import { createSessionId, getChatPath, resolveSessionId } from './session';

describe('chat session utils', () => {
  it('builds chat paths for draft and persisted sessions', () => {
    expect(getChatPath()).toBe('/chat');
    expect(getChatPath('session-123')).toBe('/chat/session-123');
  });

  it('reuses provided session ids before generating a new one', () => {
    expect(resolveSessionId('session-target', 'session-current')).toBe('session-target');
    expect(resolveSessionId('', 'session-current')).toBe('session-current');
  });

  it('generates a session id when none exists', () => {
    vi.spyOn(Date, 'now').mockReturnValue(1234567890);
    vi.spyOn(Math, 'random').mockReturnValue(0.123456789);

    expect(createSessionId()).toBe('session-1234567890-xjylrx');
    expect(resolveSessionId()).toBe('session-1234567890-xjylrx');
  });
});
