'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from '@/app/contexts/AuthContext';
import { Button } from '@/app/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/app/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/app/components/ui/avatar';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/app/components/ui/dropdown-menu';
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
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b bg-background">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-4 sm:py-6 flex items-center justify-between">
          <div className="flex items-center gap-3 sm:gap-4">
            <div className="text-3xl sm:text-4xl">🤖</div>
            <div>
              <h1 className="text-2xl sm:text-3xl font-bold text-foreground">
                AIGuide
              </h1>
              <p className="text-sm sm:text-base text-muted-foreground mt-0.5">
                AI 助手平台
              </p>
            </div>
          </div>

          {/* User Menu */}
          {user && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" className="gap-2 sm:gap-3">
                  <Avatar className="w-7 h-7 sm:w-8 sm:h-8">
                    <AvatarImage src={user.picture} alt={user.name} />
                    <AvatarFallback className="bg-blue-500 text-white text-sm">
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
      <main className="container mx-auto px-4 sm:px-6 lg:px-8 py-8 sm:py-12 lg:py-16">
        {/* Hero Section */}
        <div className="text-center mb-12 sm:mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-4 sm:mb-6 text-foreground">
            选择您的 AI 助手
          </h2>
          <p className="text-base sm:text-lg text-muted-foreground max-w-2xl mx-auto">
            基于 Google ADK 构建，使用 Gemini 模型提供智能服务
          </p>
        </div>

        {/* Agent Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 sm:gap-6 mb-12 sm:mb-16 max-w-5xl mx-auto">
          {agents.map((agent) => (
            <Link key={agent.id} href={`/chat/${agent.id}`}>
              <Card
                className={`
                  group transition-all duration-200 cursor-pointer 
                  hover:shadow-lg hover:border-muted-foreground/20
                  ${selectedAgent === agent.id ? 'shadow-md border-muted-foreground/20' : ''}
                `}
                onMouseEnter={() => setSelectedAgent(agent.id)}
                onMouseLeave={() => setSelectedAgent(null)}
              >
                <CardHeader className="pb-4">
                  <div className="flex items-start gap-4">
                    <div className="text-4xl sm:text-5xl p-3 rounded-2xl bg-secondary flex-shrink-0">
                      {agent.icon}
                    </div>
                    <div className="flex-1 pt-1">
                      <CardTitle className="text-xl sm:text-2xl mb-2">
                        {agent.name}
                      </CardTitle>
                      <CardDescription className="text-sm sm:text-base">
                        {agent.description}
                      </CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center justify-end">
                    <span className="text-sm font-medium text-muted-foreground group-hover:translate-x-1 transition-transform duration-200">
                      开始对话 →
                    </span>
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>

        {/* Quick Start Guide */}
        <Card className="mb-8 sm:mb-12 max-w-5xl mx-auto">
          <CardHeader>
            <CardTitle className="text-xl sm:text-2xl flex items-center gap-2 sm:gap-3">
              <span className="text-2xl">🚀</span>
              <span>快速开始</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 sm:space-y-4">
            <div className="flex items-start gap-3 sm:gap-4">
              <div className="flex-shrink-0 w-8 h-8 rounded-full bg-secondary text-foreground flex items-center justify-center font-semibold text-sm">
                1
              </div>
              <p className="text-sm sm:text-base text-muted-foreground pt-1">
                选择上方的 AI 助手卡片进入对话界面
              </p>
            </div>
            <div className="flex items-start gap-3 sm:gap-4">
              <div className="flex-shrink-0 w-8 h-8 rounded-full bg-secondary text-foreground flex items-center justify-center font-semibold text-sm">
                2
              </div>
              <p className="text-sm sm:text-base text-muted-foreground pt-1">
                在对话框中输入您的问题或需求
              </p>
            </div>
            <div className="flex items-start gap-3 sm:gap-4">
              <div className="flex-shrink-0 w-8 h-8 rounded-full bg-secondary text-foreground flex items-center justify-center font-semibold text-sm">
                3
              </div>
              <p className="text-sm sm:text-base text-muted-foreground pt-1">
                AI 助手将实时为您提供专业的回答和建议
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Features */}
        <div className="max-w-5xl mx-auto">
          <h3 className="text-xl sm:text-2xl font-bold text-center mb-6 sm:mb-8 text-foreground">
            平台特性
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
            <Card className="hover:shadow-md transition-shadow">
              <CardHeader>
                <div className="text-3xl sm:text-4xl mb-2 sm:mb-3">⚡</div>
                <CardTitle className="text-base sm:text-lg">实时响应</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  基于 Google Gemini 2.0 的强大模型，提供快速准确的智能回复
                </p>
              </CardContent>
            </Card>
            <Card className="hover:shadow-md transition-shadow">
              <CardHeader>
                <div className="text-3xl sm:text-4xl mb-2 sm:mb-3">🎯</div>
                <CardTitle className="text-base sm:text-lg">专业分工</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  每个助手都针对特定场景优化，提供最专业的服务体验
                </p>
              </CardContent>
            </Card>
            <Card className="hover:shadow-md transition-shadow sm:col-span-2 lg:col-span-1">
              <CardHeader>
                <div className="text-3xl sm:text-4xl mb-2 sm:mb-3">🔧</div>
                <CardTitle className="text-base sm:text-lg">工具集成</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  集成 Google Search、网页抓取、地图生成等多种实用工具
                </p>
              </CardContent>
            </Card>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t bg-background mt-12 sm:mt-16">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-4 text-sm text-muted-foreground text-center sm:text-left">
            <div>
              <p>基于 Google ADK (Agent Development Kit) 构建</p>
              <p className="text-xs mt-1">Powered by Google Gemini 2.0</p>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
