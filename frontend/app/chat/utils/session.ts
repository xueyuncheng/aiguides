export const createSessionId = () => `session-${Date.now()}-${Math.random().toString(36).substring(7)}`;

export const getChatPath = (sessionId?: string) => (sessionId ? `/chat/${sessionId}` : '/chat');

export const resolveSessionId = (targetSessionId?: string, currentSessionId?: string) => (
  targetSessionId || currentSessionId || createSessionId()
);
