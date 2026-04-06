'use client';

import { useState, useMemo, memo, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/app/components/ui/button';
import { Plus, ChevronLeft, ChevronRight, Trash2, LogOut, FolderOpen, MoreHorizontal, Pencil, Brain, Share2, Terminal } from 'lucide-react';
import { cn } from '@/app/lib/utils';
import { useAuth } from '@/app/contexts/AuthContext';
import { Avatar, AvatarFallback, AvatarImage } from '@/app/components/ui/avatar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/app/components/ui/dropdown-menu';

export interface Session {
  session_id: string;
  app_name: string;
  user_id: string;
  thread_id?: string;
  version?: number;
  last_update_time: string;
  message_count: number;
  first_message?: string;
  title?: string;
  project_id?: number;
  project_name?: string;
}

export interface Project {
  id: number;
  name: string;
}

interface SessionSidebarProps {
  sessions: Session[];
  projects: Project[];
  activeProjectId: string;
  currentSessionId: string;
  onSessionSelect: (sessionId: string) => void;
  onProjectSelect: (projectId: string) => void;
  onCreateProject: () => void;
  onRenameProject: (projectId: number, projectName: string) => void;
  onDeleteProject: (projectId: number) => void;
  onAssignSessionProject: (sessionId: string, projectId: number) => void;
  onNewSession: () => void;
  onDeleteSession: (sessionId: string) => void;
  onShareSession: (sessionId: string) => void;
  isLoading: boolean;
  isMobileOpen?: boolean;
  onMobileToggle?: () => void;
}

const SessionSidebar = memo(({
  sessions,
  projects,
  activeProjectId,
  currentSessionId,
  onSessionSelect,
  onProjectSelect,
  onCreateProject,
  onRenameProject,
  onDeleteProject,
  onAssignSessionProject,
  onNewSession,
  onDeleteSession,
  onShareSession,
  isLoading,
  isMobileOpen = false,
  onMobileToggle,
}: SessionSidebarProps) => {
  const { user, logout } = useAuth();
  const router = useRouter();
  const [isCollapsed, setIsCollapsed] = useState(false);

  const filteredSessions = useMemo(() => {
    const sessionWithContent = sessions.filter((session) => {
      if (!session.title && !session.first_message) {
        return false;
      }
      if (activeProjectId === 'all') {
        return true;
      }
      if (activeProjectId === 'none') {
        return (session.project_id ?? 0) === 0;
      }

      return String(session.project_id ?? 0) === activeProjectId;
    });
    const byThread = new Map<string, Session>();

    sessionWithContent.forEach((session) => {
      const threadId = session.thread_id || session.session_id;
      const existing = byThread.get(threadId);
      if (!existing) {
        byThread.set(threadId, session);
        return;
      }

      const existingTime = new Date(existing.last_update_time).getTime();
      const currentTime = new Date(session.last_update_time).getTime();
      if (currentTime > existingTime) {
        byThread.set(threadId, session);
        return;
      }

      if (currentTime === existingTime && (session.version || 1) > (existing.version || 1)) {
        byThread.set(threadId, session);
      }
    });

    return Array.from(byThread.values()).sort((a, b) =>
      new Date(b.last_update_time).getTime() - new Date(a.last_update_time).getTime()
    );
  }, [activeProjectId, sessions]);

  const projectOptions = useMemo(() => ([
    { id: 'all', name: '全部会话' },
    { id: 'none', name: '未归档' },
    ...projects.map((project) => ({
      id: String(project.id),
      name: project.name,
    })),
  ]), [projects]);

  const currentThreadId = useMemo(() => {
    const current = sessions.find((s) => s.session_id === currentSessionId);
    if (!current) return currentSessionId;
    return current.thread_id || current.session_id;
  }, [sessions, currentSessionId]);

  const handleSessionClick = (sessionId: string) => {
    onSessionSelect(sessionId);
    // Close mobile menu after selection
    if (onMobileToggle && isMobileOpen) {
      onMobileToggle();
    }
  };

  const handleNewSessionClick = useCallback(() => {
    onNewSession();
    // Close mobile menu after action
    if (onMobileToggle && isMobileOpen) {
      onMobileToggle();
    }
  }, [isMobileOpen, onMobileToggle, onNewSession]);

  const handleMemoryCenterClick = useCallback(() => {
    router.push('/memory');
    if (onMobileToggle && isMobileOpen) {
      onMobileToggle();
    }
  }, [isMobileOpen, onMobileToggle, router]);

  const handleSSHServersClick = useCallback(() => {
    router.push('/settings/ssh-servers');
    if (onMobileToggle && isMobileOpen) {
      onMobileToggle();
    }
  }, [isMobileOpen, onMobileToggle, router]);

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
  }, [handleNewSessionClick]);

  // Desktop collapsed state
  if (isCollapsed) {
    return (
      <div className="hidden md:flex fixed left-0 top-0 h-full bg-zinc-950 border-r border-zinc-800 z-50 flex-col items-center py-4">
        <div className="mt-auto flex flex-col items-center gap-4 pb-4">
          <Button
            onClick={() => setIsCollapsed(false)}
            variant="ghost"
            size="icon"
            className="text-muted-foreground hover:text-foreground hover:bg-zinc-900 h-8 w-8"
            aria-label="展开侧边栏"
          >
            <ChevronRight className="h-5 w-5" />
          </Button>
          <Avatar className="h-8 w-8">
            <AvatarImage src={user?.picture} alt={user?.name} />
            <AvatarFallback className="bg-primary text-primary-foreground text-xs">
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
        "fixed left-0 top-0 h-full w-[260px] bg-zinc-950 flex flex-col z-50 transition-transform duration-300 border-r border-zinc-800",
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
              className="md:hidden text-foreground hover:bg-zinc-900 h-8 w-8"
              aria-label="关闭侧边栏"
            >
              <ChevronLeft className="h-5 w-5" />
            </Button>
          </div>
          <div className="space-y-2">
            <Button
              onClick={handleNewSessionClick}
              variant="outline"
              className="w-full gap-2 justify-between border-zinc-800 bg-transparent text-zinc-100 hover:bg-zinc-900 transition-colors h-10 px-3 rounded-lg group/btn shadow-sm"
            >
              <div className="flex items-center gap-2">
                <Plus className="h-4 w-4" />
                <span className="text-sm font-medium">新建对话</span>
              </div>
              <kbd className="hidden md:inline-flex h-5 select-none items-center gap-1 rounded border border-zinc-700 bg-zinc-800 px-1.5 font-mono text-[10px] font-medium text-zinc-400 opacity-0 group-hover/btn:opacity-100 transition-opacity duration-200">
                <span className="text-xs">⌘</span>K
              </kbd>
            </Button>

            <div className="rounded-lg border border-zinc-800 bg-zinc-900/40 p-2">
              <div className="mb-2 flex items-center justify-between px-1">
                <div className="flex items-center gap-2 text-xs font-semibold text-zinc-400">
                  <FolderOpen className="h-3.5 w-3.5" />
                  <span>项目</span>
                </div>
                <Button
                  onClick={onCreateProject}
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
                  aria-label="新建项目"
                >
                  <Plus className="h-3.5 w-3.5" />
                </Button>
              </div>
              <div className="space-y-1">
                {projectOptions.map((project) => {
                  const isBuiltIn = project.id === 'all' || project.id === 'none';
                  const numericProjectId = Number(project.id);

                  return (
                    <div
                      key={project.id}
                      className={cn(
                        'group/project flex items-center gap-1 rounded-md transition-colors',
                        activeProjectId === project.id
                          ? 'bg-zinc-800'
                          : 'hover:bg-zinc-800/70'
                      )}
                    >
                      <button
                        type="button"
                        onClick={() => onProjectSelect(project.id)}
                        className={cn(
                          'flex min-w-0 flex-1 items-center rounded-md px-2 py-1.5 text-left text-sm transition-colors',
                          activeProjectId === project.id
                            ? 'text-zinc-100'
                            : 'text-zinc-400 group-hover/project:text-zinc-100'
                        )}
                      >
                        <span className="truncate">{project.name}</span>
                      </button>

                      {!isBuiltIn && !Number.isNaN(numericProjectId) && (
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              onClick={(event) => event.stopPropagation()}
                              className="mr-1 h-7 w-7 shrink-0 text-zinc-500 opacity-0 group-hover/project:opacity-100 hover:bg-transparent hover:text-zinc-100 focus-visible:bg-zinc-700 data-[state=open]:bg-zinc-700 data-[state=open]:opacity-100 data-[state=open]:text-zinc-100"
                              aria-label={`项目 ${project.name} 的更多操作`}
                            >
                              <MoreHorizontal className="h-3.5 w-3.5" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end" className="w-40 bg-zinc-900 border-zinc-800 text-zinc-100 shadow-xl">
                            <DropdownMenuItem
                              className="cursor-pointer"
                              onClick={(event) => {
                                event.stopPropagation();
                                onRenameProject(numericProjectId, project.name);
                              }}
                            >
                              <Pencil className="mr-2 h-4 w-4" />
                              <span>重命名</span>
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              className="cursor-pointer text-red-400 focus:bg-red-900/20 focus:text-red-400"
                              onClick={(event) => {
                                event.stopPropagation();
                                onDeleteProject(numericProjectId);
                              }}
                            >
                              <Trash2 className="mr-2 h-4 w-4" />
                              <span>删除</span>
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
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
                    "group relative p-2 rounded-md cursor-pointer transition-colors text-sm",
                    (session.thread_id || session.session_id) === currentThreadId
                      ? "bg-zinc-900 text-zinc-100"
                      : "text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900"
                  )}
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="min-w-0 flex-1">
                      <div className="truncate">
                        {session.title || session.first_message || '新对话'}
                      </div>
                      {session.project_name && (
                        <div className="truncate text-[11px] text-zinc-500">
                          {session.project_name}
                        </div>
                      )}
                    </div>
                    {(session.version || 1) > 1 && (
                      <span className="text-[10px] text-zinc-500">v{session.version}</span>
                    )}

                    {/* Delete button only visible on hover */}
                    <div className={cn(
                      "flex items-center opacity-0 group-hover:opacity-100",
                      (session.thread_id || session.session_id) === currentThreadId && "opacity-100"
                    )}>
                      {/* Shadow gradient to cover text */}
                      <div className="absolute right-0 top-0 bottom-0 w-12 bg-gradient-to-l from-zinc-950 to-transparent pointer-events-none group-hover:from-zinc-900 group-hover:via-zinc-900"></div>
                      {(session.thread_id || session.session_id) === currentThreadId && <div className="absolute right-0 top-0 bottom-0 w-12 bg-gradient-to-l from-zinc-900 to-transparent pointer-events-none"></div>}

                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="relative z-10 h-6 w-6 text-zinc-500 hover:text-zinc-200"
                            aria-label="会话选项"
                            onClick={(e) => e.stopPropagation()}
                          >
                            <MoreHorizontal className="h-3.5 w-3.5" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-48 bg-zinc-900 border-zinc-800 text-zinc-100 shadow-xl">
                          <DropdownMenuSub>
                            <DropdownMenuSubTrigger>
                              移动到项目
                            </DropdownMenuSubTrigger>
                            <DropdownMenuSubContent className="w-48 bg-zinc-900 border-zinc-800 text-zinc-100 shadow-xl">
                              <DropdownMenuRadioGroup value={session.project_id === 0 ? 'none' : String(session.project_id ?? 0)}>
                                <DropdownMenuRadioItem
                                  value="none"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    onAssignSessionProject(session.session_id, 0);
                                  }}
                                >
                                  未归档
                                </DropdownMenuRadioItem>
                                {projects.map((project) => (
                                  <DropdownMenuRadioItem
                                    key={project.id}
                                    value={String(project.id)}
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      onAssignSessionProject(session.session_id, project.id);
                                    }}
                                  >
                                    {project.name}
                                  </DropdownMenuRadioItem>
                                ))}
                              </DropdownMenuRadioGroup>
                            </DropdownMenuSubContent>
                          </DropdownMenuSub>
                          <DropdownMenuItem
                            className="cursor-pointer"
                            onClick={(e) => {
                              e.stopPropagation();
                              onCreateProject();
                            }}
                          >
                            <Plus className="mr-2 h-4 w-4" />
                            <span>新建项目</span>
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            className="cursor-pointer"
                            onClick={(e) => {
                              e.stopPropagation();
                              onShareSession(session.session_id);
                            }}
                          >
                            <Share2 className="mr-2 h-4 w-4" />
                            <span>分享会话</span>
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
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
        <div className="p-3 border-t border-zinc-800 flex items-center gap-2">
          <div className="flex-1">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  className="flex items-center justify-center h-10 w-10 p-0 hover:bg-zinc-900 text-zinc-100 rounded-full focus-visible:ring-0 transition-colors"
                >
                  <Avatar className="h-8 w-8">
                    <AvatarImage src={user?.picture} alt={user?.name} />
                    <AvatarFallback className="bg-primary text-primary-foreground text-xs font-semibold">
                      {user?.name?.charAt(0).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" side="top" className="w-56 bg-zinc-900 border-zinc-800 text-zinc-100">
                <DropdownMenuItem
                  className="cursor-pointer"
                  onClick={handleMemoryCenterClick}
                >
                  <Brain className="mr-2 h-4 w-4" />
                  <span>记忆中心</span>
                </DropdownMenuItem>
                <DropdownMenuItem
                  className="cursor-pointer"
                  onClick={handleSSHServersClick}
                >
                  <Terminal className="mr-2 h-4 w-4" />
                  <span>SSH Servers</span>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
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
            className="hidden md:flex text-zinc-500 hover:text-zinc-200 hover:bg-zinc-900 h-8 w-8 flex-shrink-0"
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
