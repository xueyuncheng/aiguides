'use client';

import { useState, useMemo, memo, useEffect } from 'react';
import { Button } from '@/app/components/ui/button';
import { Plus, ChevronLeft, ChevronRight, Trash2, LogOut } from 'lucide-react';
import { cn } from '@/app/lib/utils';
import { useAuth } from '@/app/contexts/AuthContext';
import { Avatar, AvatarFallback, AvatarImage } from '@/app/components/ui/avatar';
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
  const { user, logout } = useAuth();
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

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl + K: New Session
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        handleNewSessionClick();
      }
      // Cmd/Ctrl + B: Toggle Sidebar
      if ((e.metaKey || e.ctrlKey) && e.key === 'b') {
        e.preventDefault();
        setIsCollapsed(prev => !prev);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onNewSession, isMobileOpen, onMobileToggle]);

  // Desktop collapsed state
  if (isCollapsed) {
    return (
      <div className="hidden md:flex fixed left-0 top-0 h-full bg-[#171717] border-r border-[#2c2c2c] z-50 flex-col items-center py-4">
        <div className="mt-auto flex flex-col items-center gap-4 pb-4">
          <Button
            onClick={() => setIsCollapsed(false)}
            variant="ghost"
            size="icon"
            className="text-[#8e8e8e] hover:text-[#ececec] hover:bg-[#2c2c2c] h-8 w-8"
            aria-label="展开侧边栏"
          >
            <ChevronRight className="h-5 w-5" />
          </Button>
          <Avatar className="h-8 w-8">
            <AvatarImage src={user?.picture} alt={user?.name} />
            <AvatarFallback className="bg-blue-500 text-white text-xs">
              {user?.name?.charAt(0).toUpperCase()}
            </AvatarFallback>
          </Avatar>
        </div>
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
      <div className={cn(
        "fixed left-0 top-0 h-full w-[260px] bg-[#171717] flex flex-col z-50 transition-transform duration-300",
        isMobileOpen ? "translate-x-0" : "-translate-x-full md:translate-x-0"
      )}>
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
          </div>
          <div className="space-y-2">

            <Button
              onClick={handleNewSessionClick}
              className="w-full gap-2 justify-between border border-[#424242] bg-transparent text-[#ececec] hover:bg-[#2c2c2c] transition-colors h-10 px-3 rounded-lg group/btn"
            >
              <div className="flex items-center gap-2">
                <Plus className="h-4 w-4" />
                <span className="text-sm">新建对话</span>
              </div>
              <kbd className="hidden md:inline-flex h-5 select-none items-center gap-1 rounded border border-[#424242] bg-[#212121] px-1.5 font-mono text-[10px] font-medium text-[#8e8e8e] opacity-0 group-hover/btn:opacity-100 transition-opacity duration-200">
                <span className="text-xs">⌘</span>K
              </kbd>
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
                  className={cn(
                    "group relative p-2.5 rounded-lg cursor-pointer transition-colors text-sm",
                    session.session_id === currentSessionId
                      ? "bg-[#2c2c2c] text-[#ececec]"
                      : "text-[#ececec] hover:bg-[#212121]"
                  )}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate flex-1">
                      {session.title || session.first_message || '新对话'}
                    </span>

                    {/* Delete button only visible on hover */}
                    <div className={cn(
                      "flex items-center opacity-0 group-hover:opacity-100",
                      session.session_id === currentSessionId && "opacity-100"
                    )}>
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

        {/* User Profile */}
        <div className="p-3 border-t border-[#2c2c2c] flex items-center gap-2">
          <div className="flex-1">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  className="flex items-center justify-center h-10 w-10 p-0 hover:bg-[#2c2c2c] text-[#ececec] rounded-full focus-visible:ring-0 focus-visible:ring-offset-0"
                >
                  <Avatar className="h-8 w-8">
                    <AvatarImage src={user?.picture} alt={user?.name} />
                    <AvatarFallback className="bg-blue-500 text-white text-sm">
                      {user?.name?.charAt(0).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" side="top" className="w-56 bg-[#2c2c2c] border-[#424242] text-[#ececec]">
                <DropdownMenuItem
                  className="text-red-400 focus:text-red-400 focus:bg-red-900/20 cursor-pointer"
                  onClick={logout}
                >
                  <LogOut className="mr-2 h-4 w-4" />
                  <span>退出登录</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <Button
            onClick={() => setIsCollapsed(true)}
            variant="ghost"
            size="icon"
            className="hidden md:flex text-[#8e8e8e] hover:text-[#ececec] hover:bg-[#2c2c2c] h-8 w-8 flex-shrink-0"
            aria-label="收起侧边栏"
          >
            <ChevronLeft className="h-5 w-5" />
          </Button>
        </div>
      </div>
    </>
  );
});

SessionSidebar.displayName = 'SessionSidebar';

export default SessionSidebar;
