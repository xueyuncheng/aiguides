'use client';

import { useState, useEffect } from 'react';
import { Button } from './ui/button';
import { Separator } from './ui/separator';
import { Plus, ChevronLeft, ChevronRight, Trash2 } from 'lucide-react';

export interface Session {
  session_id: string;
  app_name: string;
  user_id: string;
  last_update_time: string;
  message_count: number;
  first_message?: string;
  title?: string;
}

interface SessionSidebarProps {
  sessions: Session[];
  currentSessionId: string;
  onSessionSelect: (sessionId: string) => void;
  onNewSession: () => void;
  onDeleteSession: (sessionId: string) => void;
  isLoading: boolean;
}

export default function SessionSidebar({
  sessions,
  currentSessionId,
  onSessionSelect,
  onNewSession,
  onDeleteSession,
  isLoading,
}: SessionSidebarProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);

  const handleDelete = (sessionId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (confirm('确定要删除这个会话吗？')) {
      onDeleteSession(sessionId);
    }
  };

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return '刚刚';
    if (diffMins < 60) return `${diffMins}分钟前`;
    if (diffHours < 24) return `${diffHours}小时前`;
    if (diffDays < 7) return `${diffDays}天前`;

    return date.toLocaleDateString('zh-CN', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  if (isCollapsed) {
    return (
      <div className="fixed left-0 top-0 h-full bg-card border-r shadow-lg z-50">
        <Button
          onClick={() => setIsCollapsed(false)}
          variant="ghost"
          size="icon"
          className="m-4"
          aria-label="展开侧边栏"
        >
          <ChevronRight className="h-6 w-6" />
        </Button>
      </div>
    );
  }

  return (
    <div className="fixed left-0 top-0 h-full w-80 bg-card border-r shadow-lg flex flex-col z-50">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <h2 className="text-lg font-semibold">会话历史</h2>
        <Button
          onClick={() => setIsCollapsed(true)}
          variant="ghost"
          size="icon"
          aria-label="收起侧边栏"
        >
          <ChevronLeft className="h-5 w-5" />
        </Button>
      </div>

      {/* New Session Button */}
      <div className="p-4">
        <Button
          onClick={onNewSession}
          className="w-full gap-2"
          size="lg"
        >
          <Plus className="h-5 w-5" />
          新建对话
        </Button>
      </div>

      {/* Sessions List */}
      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <div className="p-4 text-center text-muted-foreground">
            加载中...
          </div>
        ) : sessions.length === 0 ? (
          <div className="p-4 text-center text-muted-foreground">
            暂无会话历史
          </div>
        ) : (
          <div className="space-y-1 p-2">
            {sessions.map((session) => (
              <div
                key={session.session_id}
                onClick={() => onSessionSelect(session.session_id)}
                className={`group relative p-3 rounded-lg cursor-pointer transition-colors ${
                  session.session_id === currentSessionId
                    ? 'bg-accent border border-border'
                    : 'hover:bg-accent/50'
                }`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium truncate">
                      {session.title || session.first_message || '新对话'}
                    </div>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs text-muted-foreground">
                        {formatTime(session.last_update_time)}
                      </span>
                      {session.message_count > 0 && (
                        <span className="text-xs text-muted-foreground">
                          · {session.message_count} 条消息
                        </span>
                      )}
                    </div>
                  </div>
                  <Button
                    onClick={(e) => handleDelete(session.session_id, e)}
                    variant="ghost"
                    size="icon"
                    className="opacity-0 group-hover:opacity-100 h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
                    aria-label="删除会话"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
