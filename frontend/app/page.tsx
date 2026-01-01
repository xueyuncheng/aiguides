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
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50 dark:from-gray-900 dark:via-blue-950 dark:to-purple-950">
      {/* Header */}
      <header className="border-b border-white/20 bg-white/90 backdrop-blur-md dark:bg-gray-900/90 shadow-sm">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-4 sm:py-6 flex items-center justify-between">
          <div className="flex items-center gap-3 sm:gap-4">
            <div className="text-4xl sm:text-5xl">🤖</div>
            <div>
              <h1 className="text-2xl sm:text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 dark:from-blue-400 dark:to-purple-400 bg-clip-text text-transparent">
                AIGuide
              </h1>
              <p className="text-sm sm:text-base text-muted-foreground mt-0.5">
                智能 AI 助手平台
              </p>
            </div>
          </div>

          {/* User Menu */}
          {user && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" className="gap-2 sm:gap-3 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800">
                  <Avatar className="w-7 h-7 sm:w-8 sm:h-8">
                    <AvatarImage src={user.picture} alt={user.name} />
                    <AvatarFallback className="bg-gradient-to-br from-blue-500 to-purple-500 text-white text-sm">
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
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-4 sm:mb-6 bg-gradient-to-r from-blue-600 via-purple-600 to-pink-600 dark:from-blue-400 dark:via-purple-400 dark:to-pink-400 bg-clip-text text-transparent">
            选择您的专属 AI 助手
          </h2>
          <p className="text-base sm:text-lg text-muted-foreground max-w-2xl mx-auto leading-relaxed">
            基于 Google ADK 构建，集成先进的 Gemini 模型
            <br className="hidden sm:block" />
            为您提供专业、智能的 AI 服务体验
          </p>
        </div>

        {/* Agent Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-5 sm:gap-6 lg:gap-8 mb-12 sm:mb-16">
          {agents.map((agent) => (
            <Link key={agent.id} href={`/chat/${agent.id}`}>
              <Card
                className={`
                  group relative overflow-hidden transition-all duration-300 cursor-pointer 
                  hover:shadow-2xl hover:scale-[1.02] hover:-translate-y-1
                  bg-white/90 dark:bg-gray-800/90 backdrop-blur-sm
                  border-2 hover:border-purple-200 dark:hover:border-purple-700
                  ${selectedAgent === agent.id ? 'shadow-xl border-purple-300 dark:border-purple-600 scale-[1.01]' : 'border-gray-200 dark:border-gray-700'}
                `}
                onMouseEnter={() => setSelectedAgent(agent.id)}
                onMouseLeave={() => setSelectedAgent(null)}
              >
                {/* Background Gradient Effect */}
                <div className={`absolute inset-0 ${agent.color} opacity-0 group-hover:opacity-5 transition-opacity duration-300`}></div>
                
                <CardHeader className="pb-4">
                  <div className="flex items-start gap-4">
                    <div className={`text-5xl sm:text-6xl p-3 sm:p-4 rounded-2xl ${agent.color} bg-opacity-10 group-hover:scale-110 transition-transform duration-300`}>
                      {agent.icon}
                    </div>
                    <div className="flex-1 pt-1">
                      <CardTitle className="text-xl sm:text-2xl mb-2 sm:mb-3 group-hover:text-purple-600 dark:group-hover:text-purple-400 transition-colors">
                        {agent.name}
                      </CardTitle>
                      <CardDescription className="text-sm sm:text-base leading-relaxed">
                        {agent.description}
                      </CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center justify-end">
                    <span className="text-sm font-semibold text-purple-600 dark:text-purple-400 group-hover:translate-x-2 transition-transform duration-300 flex items-center gap-2">
                      开始对话
                      <span className="text-lg">→</span>
                    </span>
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>

        {/* Quick Start Guide */}
        <Card className="mb-8 sm:mb-12 bg-gradient-to-br from-blue-50 to-purple-50 dark:from-blue-950/50 dark:to-purple-950/50 border-blue-200 dark:border-blue-800 shadow-lg">
          <CardHeader>
            <CardTitle className="text-xl sm:text-2xl flex items-center gap-2 sm:gap-3">
              <span className="text-2xl sm:text-3xl">🚀</span>
              <span>快速开始</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 sm:space-y-4">
            <div className="flex items-start gap-3 sm:gap-4">
              <div className="flex-shrink-0 w-8 h-8 sm:w-10 sm:h-10 rounded-full bg-blue-500 text-white flex items-center justify-center font-bold text-sm sm:text-base">
                1
              </div>
              <p className="text-sm sm:text-base text-muted-foreground pt-1">
                选择上方的 AI 助手卡片进入对话界面
              </p>
            </div>
            <div className="flex items-start gap-3 sm:gap-4">
              <div className="flex-shrink-0 w-8 h-8 sm:w-10 sm:h-10 rounded-full bg-purple-500 text-white flex items-center justify-center font-bold text-sm sm:text-base">
                2
              </div>
              <p className="text-sm sm:text-base text-muted-foreground pt-1">
                在对话框中输入您的问题或需求
              </p>
            </div>
            <div className="flex items-start gap-3 sm:gap-4">
              <div className="flex-shrink-0 w-8 h-8 sm:w-10 sm:h-10 rounded-full bg-pink-500 text-white flex items-center justify-center font-bold text-sm sm:text-base">
                3
              </div>
              <p className="text-sm sm:text-base text-muted-foreground pt-1">
                AI 助手将实时为您提供专业的回答和建议
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Features */}
        <div>
          <h3 className="text-xl sm:text-2xl font-bold text-center mb-6 sm:mb-8">
            ✨ 平台特性
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
            <Card className="bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-950/30 dark:to-blue-900/30 border-blue-200 dark:border-blue-800 hover:shadow-xl transition-all duration-300 hover:scale-105">
              <CardHeader>
                <div className="text-3xl sm:text-4xl mb-2 sm:mb-3">⚡</div>
                <CardTitle className="text-base sm:text-lg">实时响应</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground leading-relaxed">
                  基于 Google Gemini 2.0 的强大模型，提供快速准确的智能回复
                </p>
              </CardContent>
            </Card>
            <Card className="bg-gradient-to-br from-green-50 to-emerald-100 dark:from-green-950/30 dark:to-emerald-900/30 border-green-200 dark:border-green-800 hover:shadow-xl transition-all duration-300 hover:scale-105">
              <CardHeader>
                <div className="text-3xl sm:text-4xl mb-2 sm:mb-3">🎯</div>
                <CardTitle className="text-base sm:text-lg">专业分工</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground leading-relaxed">
                  每个助手都针对特定场景优化，提供最专业的服务体验
                </p>
              </CardContent>
            </Card>
            <Card className="bg-gradient-to-br from-purple-50 to-pink-100 dark:from-purple-950/30 dark:to-pink-900/30 border-purple-200 dark:border-purple-800 hover:shadow-xl transition-all duration-300 hover:scale-105 sm:col-span-2 lg:col-span-1">
              <CardHeader>
                <div className="text-3xl sm:text-4xl mb-2 sm:mb-3">🔧</div>
                <CardTitle className="text-base sm:text-lg">工具集成</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground leading-relaxed">
                  集成 Google Search、网页抓取、地图生成等多种实用工具
                </p>
              </CardContent>
            </Card>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-white/20 bg-white/90 backdrop-blur-md dark:bg-gray-900/90 mt-12 sm:mt-16">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
            <div className="text-center sm:text-left">
              <p className="text-sm text-muted-foreground">
                基于 <span className="font-semibold text-blue-600 dark:text-blue-400">Google ADK</span> (Agent Development Kit) 构建
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                Powered by <span className="font-semibold">Google Gemini 2.0</span>
              </p>
            </div>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>Made with</span>
              <span className="text-red-500 text-lg animate-pulse">♥</span>
              <span>by AIGuide Team</span>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
