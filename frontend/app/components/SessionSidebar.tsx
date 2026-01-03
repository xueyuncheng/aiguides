'use client';

import { useState, useEffect, useMemo, memo } from 'react';
import Link from 'next/link';
import { Button } from '@/app/components/ui/button';
import { Plus, ChevronLeft, ChevronRight, Trash2, Home, Menu } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/app/components/ui/dropdown-menu';

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
  isMobileOpen?: boolean;
  onMobileToggle?: () => void;
}

const SessionSidebar = memo(({
  sessions,
  currentSessionId,
  onSessionSelect,
  onNewSession,
  onDeleteSession,
  isLoading,
  isMobileOpen = false,
  onMobileToggle,
}: SessionSidebarProps) => {
  const [isCollapsed, setIsCollapsed] = useState(false);

  // Memoize filtered sessions to avoid redundant filtering
  const filteredSessions = useMemo(
    () => sessions.filter(s => s.title || s.first_message),
    [sessions]
  );

  const handleSessionClick = (sessionId: string) => {
    onSessionSelect(sessionId);
    // Close mobile menu after selection
    if (onMobileToggle && isMobileOpen) {
      onMobileToggle();
    }
  };

  const handleNewSessionClick = () => {
    onNewSession();
    // Close mobile menu after action
    if (onMobileToggle && isMobileOpen) {
      onMobileToggle();
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
      <div className="hidden md:flex fixed left-0 top-0 h-full bg-[#171717] border-r border-[#2c2c2c] z-50 flex-col items-center py-4">
        <Button
          onClick={() => setIsCollapsed(false)}
          variant="ghost"
          size="icon"
          className="text-[#ececec] hover:bg-[#2c2c2c]"
          aria-label="展开侧边栏"
        >
          <ChevronRight className="h-5 w-5" />
        </Button>
      </div>
    );
  }

  return (
    <>
      {/* Mobile overlay backdrop */}
      {isMobileOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-40 md:hidden"
          onClick={onMobileToggle}
        />
      )}
      
      {/* Sidebar */}
      <div className={`fixed left-0 top-0 h-full w-[260px] bg-[#171717] flex flex-col z-50 transition-transform duration-300 ${isMobileOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}`}>
        {/* Header & New Chat */}
        <div className="p-3">
          <div className="flex justify-between items-center mb-2">
            {/* Mobile close button */}
            <Button
              onClick={onMobileToggle}
              variant="ghost"
              size="icon"
              className="md:hidden text-[#ececec] hover:bg-[#2c2c2c] h-8 w-8"
              aria-label="关闭侧边栏"
            >
              <ChevronLeft className="h-5 w-5" />
            </Button>
            {/* Desktop collapse button */}
            <Button
              onClick={() => setIsCollapsed(true)}
              variant="ghost"
              size="icon"
              className="hidden md:block text-[#ececec] hover:bg-[#2c2c2c] ml-auto h-8 w-8"
              aria-label="收起侧边栏"
            >
              <ChevronLeft className="h-5 w-5" />
            </Button>
          </div>
          <div className="space-y-2">
            <Link href="/" className="block">
              <Button
                className="w-full gap-2 justify-start border border-[#424242] bg-transparent text-[#ececec] hover:bg-[#2c2c2c] transition-colors h-10 px-3 rounded-lg"
              >
                <Home className="h-4 w-4" />
                <span className="text-sm">返回首页</span>
              </Button>
            </Link>
            <Button
              onClick={handleNewSessionClick}
              className="w-full gap-2 justify-start border border-[#424242] bg-transparent text-[#ececec] hover:bg-[#2c2c2c] transition-colors h-10 px-3 rounded-lg"
            >
              <Plus className="h-4 w-4" />
              <span className="text-sm">新建对话</span>
            </Button>
          </div>
        </div>

      {/* Sessions List */}
      <div className="flex-1 overflow-y-auto px-3">
        {isLoading ? (
          <div className="p-4 text-center text-[#8e8e8e] text-sm">
            加载中...
          </div>
        ) : filteredSessions.length === 0 ? (
          <div className="p-4 text-center text-[#8e8e8e] text-sm">
            暂无会话历史
          </div>
        ) : (
          <div className="space-y-1">
            <h3 className="px-3 py-2 text-xs font-semibold text-[#8e8e8e]">最近</h3>
            {filteredSessions.map((session) => (
              <div
                key={session.session_id}
                onClick={() => handleSessionClick(session.session_id)}
                className={`group relative p-2.5 rounded-lg cursor-pointer transition-colors text-sm ${session.session_id === currentSessionId
                  ? 'bg-[#2c2c2c] text-[#ececec]'
                  : 'text-[#ececec] hover:bg-[#212121]'
                  }`}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="truncate flex-1">
                    {session.title || session.first_message || '新对话'}
                  </span>

                  {/* Delete button only visible on hover */}
                  <div className={`flex items-center opacity-0 group-hover:opacity-100 ${session.session_id === currentSessionId ? 'opacity-100' : ''}`}>
                    {/* Shadow gradient to cover text */}
                    <div className="absolute right-0 top-0 bottom-0 w-12 bg-gradient-to-l from-[#171717] to-transparent pointer-events-none group-hover:from-[#212121] group-hover:via-[#212121]"></div>
                    {session.session_id === currentSessionId && <div className="absolute right-0 top-0 bottom-0 w-12 bg-gradient-to-l from-[#2c2c2c] to-transparent pointer-events-none"></div>}

                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="relative z-10 h-6 w-6 text-[#8e8e8e] hover:text-white"
                          aria-label="删除选项"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-48 bg-[#2c2c2c] border-[#424242] text-[#ececec]">
                        <DropdownMenuItem
                          className="text-red-400 focus:text-red-400 focus:bg-red-900/20 cursor-pointer"
                          onClick={(e) => {
                            e.stopPropagation();
                            onDeleteSession(session.session_id);
                          }}
                        >
                          <Trash2 className="mr-2 h-4 w-4" />
                          <span>确认删除</span>
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
      </div>
    </>
  );
});

SessionSidebar.displayName = 'SessionSidebar';

export default SessionSidebar;
