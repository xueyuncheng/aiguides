'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from './contexts/AuthContext';
import { Button } from './components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from './components/ui/avatar';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from './components/ui/dropdown-menu';
import { ChevronDown } from 'lucide-react';

interface Agent {
  id: string;
  name: string;
  description: string;
  icon: string;
  color: string;
}

const agents: Agent[] = [
  {
    id: 'assistant',
    name: 'AI Assistant',
    description: '信息检索和事实核查助手，帮您快速获取准确、全面的信息',
    icon: '🔍',
    color: 'bg-blue-500',
  },
  {
    id: 'web_summary',
    name: 'WebSummary Agent',
    description: '专业的网页内容分析助手，快速提取和总结网页关键信息',
    icon: '🌐',
    color: 'bg-green-500',
  },
  {
    id: 'email_summary',
    name: 'EmailSummary Agent',
    description: '智能邮件总结助手，自动分析和归类重要邮件（仅限 macOS）',
    icon: '📧',
    color: 'bg-purple-500',
  },
  {
    id: 'travel',
    name: 'Travel Agent',
    description: '旅游规划助手，为您定制详细的旅行计划和地图路线',
    icon: '✈️',
    color: 'bg-orange-500',
  },
];

export default function Home() {
  const router = useRouter();
  const { user, loading, logout } = useAuth();
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);

  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
    }
  }, [user, loading, router]);

  const handleLogout = async () => {
    await logout();
    router.push('/login');
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
        <div className="text-center">
          <div className="w-16 h-16 border-4 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-muted-foreground">加载中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
      {/* Header */}
      <header className="border-b bg-white/80 backdrop-blur-sm dark:bg-gray-800/80">
        <div className="container mx-auto px-4 py-6 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-foreground">
              🤖 AIGuide - AI 助手平台
            </h1>
            <p className="mt-2 text-muted-foreground">
              基于 Google ADK 构建的智能助手服务
            </p>
          </div>

          {/* User Menu */}
          {user && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" className="gap-3">
                  <Avatar className="w-8 h-8">
                    <AvatarImage src={user.picture} alt={user.name} />
                    <AvatarFallback className="bg-blue-500 text-white">
                      {user.name.charAt(0).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                  <div className="text-left hidden sm:block">
                    <p className="text-sm font-medium">
                      {user.name}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {user.email}
                    </p>
                  </div>
                  <ChevronDown className="h-4 w-4 text-muted-foreground" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuItem onClick={handleLogout}>
                  退出登录
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-12">
        <div className="mb-8">
          <h2 className="text-2xl font-semibold mb-4">
            选择您的 AI 助手
          </h2>
          <p className="text-muted-foreground">
            点击下方卡片与不同的 AI 助手交互
          </p>
        </div>

        {/* Agent Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-2 gap-6 mb-12">
          {agents.map((agent) => (
            <Link key={agent.id} href={`/chat/${agent.id}`}>
              <Card
                className={`
                  transition-all duration-300 cursor-pointer hover:shadow-xl hover:scale-105
                  ${selectedAgent === agent.id ? 'shadow-lg border-2' : ''}
                `}
                onMouseEnter={() => setSelectedAgent(agent.id)}
                onMouseLeave={() => setSelectedAgent(null)}
              >
                <CardHeader>
                  <div className="flex items-start gap-4">
                    <div className={`text-4xl p-3 rounded-lg ${agent.color} bg-opacity-10`}>
                      {agent.icon}
                    </div>
                    <div className="flex-1">
                      <CardTitle className="text-xl mb-2">
                        {agent.name}
                      </CardTitle>
                      <CardDescription className="text-sm leading-relaxed">
                        {agent.description}
                      </CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center justify-end text-sm font-medium text-muted-foreground">
                    开始对话 →
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>

        {/* Quick Start Guide */}
        <Card className="mb-8">
          <CardHeader>
            <CardTitle className="text-xl">
              🚀 快速开始
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-muted-foreground">
            <p>
              <strong>1.</strong> 选择上方的 AI 助手卡片进入对话界面
            </p>
            <p>
              <strong>2.</strong> 在对话框中输入您的问题或需求
            </p>
            <p>
              <strong>3.</strong> AI 助手将实时为您提供专业的回答和建议
            </p>
          </CardContent>
        </Card>

        {/* Features */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card className="bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800">
            <CardHeader>
              <div className="text-2xl mb-2">⚡</div>
              <CardTitle className="text-base">实时响应</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                基于 Google Gemini 的强大模型，快速准确的回复
              </p>
            </CardContent>
          </Card>
          <Card className="bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800">
            <CardHeader>
              <div className="text-2xl mb-2">🎯</div>
              <CardTitle className="text-base">专业分工</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                每个助手都针对特定场景优化，提供专业服务
              </p>
            </CardContent>
          </Card>
          <Card className="bg-purple-50 dark:bg-purple-900/20 border-purple-200 dark:border-purple-800">
            <CardHeader>
              <div className="text-2xl mb-2">🔧</div>
              <CardTitle className="text-base">工具集成</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                集成 Google Search、网页抓取、地图等实用工具
              </p>
            </CardContent>
          </Card>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t bg-white/80 backdrop-blur-sm dark:bg-gray-800/80 mt-12">
        <div className="container mx-auto px-4 py-6 text-center text-muted-foreground">
          <p>基于 Google ADK (Agent Development Kit) 构建 | Powered by Google Gemini</p>
        </div>
      </footer>
    </div>
  );
}
