'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from './contexts/AuthContext';

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
  const [showUserMenu, setShowUserMenu] = useState(false);

  // Check if authentication is required by checking if backend has auth enabled
  const [authRequired, setAuthRequired] = useState(false);
  const [checkingAuth, setCheckingAuth] = useState(true);

  useEffect(() => {
    // Check if backend requires authentication
    const checkAuthRequirement = async () => {
      try {
        // Use dedicated config endpoint
        const response = await fetch('/config');
        if (response.ok) {
          const config = await response.json();
          setAuthRequired(config.authentication_enabled);

          // If auth is required and user is not logged in, redirect to login
          if (config.authentication_enabled && !loading && !user) {
            router.push('/login');
          }
        } else {
          setAuthRequired(false);
        }
      } catch (error) {
        console.error('Failed to check auth requirement:', error);
        setAuthRequired(false);
      } finally {
        setCheckingAuth(false);
      }
    };

    checkAuthRequirement();
  }, [user, loading, router]);

  const handleLogout = async () => {
    await logout();
    router.push('/login');
  };

  if (checkingAuth || loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
        <div className="text-center">
          <div className="w-16 h-16 border-4 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">加载中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
      {/* Header */}
      <header className="border-b bg-white/80 backdrop-blur-sm dark:bg-gray-800/80 dark:border-gray-700">
        <div className="container mx-auto px-4 py-6 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
              🤖 AIGuide - AI 助手平台
            </h1>
            <p className="mt-2 text-gray-600 dark:text-gray-300">
              基于 Google ADK 构建的智能助手服务
            </p>
          </div>

          {/* User Menu */}
          {authRequired && user && (
            <div className="relative">
              <button
                onClick={() => setShowUserMenu(!showUserMenu)}
                className="flex items-center gap-3 px-4 py-2 rounded-lg bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
              >
                <div className="w-8 h-8 rounded-full bg-blue-500 overflow-hidden flex items-center justify-center text-white font-semibold">
                  {user.picture ? (
                    <img src={user.picture} alt={user.name} className="w-full h-full object-cover" />
                  ) : (
                    user.name.charAt(0).toUpperCase()
                  )}
                </div>
                <div className="text-left hidden sm:block">
                  <p className="text-sm font-medium text-gray-900 dark:text-white">
                    {user.name}
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    {user.email}
                  </p>
                </div>
                <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              {/* Dropdown Menu */}
              {showUserMenu && (
                <div className="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-2 z-10">
                  <button
                    onClick={handleLogout}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                  >
                    退出登录
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-12">
        <div className="mb-8">
          <h2 className="text-2xl font-semibold text-gray-800 dark:text-gray-200 mb-4">
            选择您的 AI 助手
          </h2>
          <p className="text-gray-600 dark:text-gray-400">
            点击下方卡片与不同的 AI 助手交互
          </p>
        </div>

        {/* Agent Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-2 gap-6 mb-12">
          {agents.map((agent) => (
            <Link key={agent.id} href={`/chat/${agent.id}`}>
              <div
                className={`
                  p-6 rounded-xl border-2 bg-white dark:bg-gray-800 
                  transition-all duration-300 cursor-pointer
                  hover:shadow-xl hover:scale-105 hover:border-gray-400
                  ${selectedAgent === agent.id ? 'border-gray-400 shadow-lg' : 'border-gray-200 dark:border-gray-700'}
                `}
                onMouseEnter={() => setSelectedAgent(agent.id)}
                onMouseLeave={() => setSelectedAgent(null)}
              >
                <div className="flex items-start gap-4">
                  <div className={`text-4xl p-3 rounded-lg ${agent.color} bg-opacity-10`}>
                    {agent.icon}
                  </div>
                  <div className="flex-1">
                    <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
                      {agent.name}
                    </h3>
                    <p className="text-gray-600 dark:text-gray-400 text-sm leading-relaxed">
                      {agent.description}
                    </p>
                  </div>
                </div>
                <div className="mt-4 flex items-center justify-end text-sm font-medium text-gray-500 dark:text-gray-400">
                  开始对话 →
                </div>
              </div>
            </Link>
          ))}
        </div>

        {/* Quick Start Guide */}
        <div className="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
          <h3 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
            🚀 快速开始
          </h3>
          <div className="space-y-3 text-gray-600 dark:text-gray-400">
            <p>
              <strong>1.</strong> 选择上方的 AI 助手卡片进入对话界面
            </p>
            <p>
              <strong>2.</strong> 在对话框中输入您的问题或需求
            </p>
            <p>
              <strong>3.</strong> AI 助手将实时为您提供专业的回答和建议
            </p>
          </div>
        </div>

        {/* Features */}
        <div className="mt-8 grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
            <div className="text-2xl mb-2">⚡</div>
            <h4 className="font-semibold text-gray-900 dark:text-white mb-1">实时响应</h4>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              基于 Google Gemini 的强大模型，快速准确的回复
            </p>
          </div>
          <div className="bg-green-50 dark:bg-green-900/20 rounded-lg p-4">
            <div className="text-2xl mb-2">🎯</div>
            <h4 className="font-semibold text-gray-900 dark:text-white mb-1">专业分工</h4>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              每个助手都针对特定场景优化，提供专业服务
            </p>
          </div>
          <div className="bg-purple-50 dark:bg-purple-900/20 rounded-lg p-4">
            <div className="text-2xl mb-2">🔧</div>
            <h4 className="font-semibold text-gray-900 dark:text-white mb-1">工具集成</h4>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              集成 Google Search、网页抓取、地图等实用工具
            </p>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t bg-white/80 backdrop-blur-sm dark:bg-gray-800/80 dark:border-gray-700 mt-12">
        <div className="container mx-auto px-4 py-6 text-center text-gray-600 dark:text-gray-400">
          <p>基于 Google ADK (Agent Development Kit) 构建 | Powered by Google Gemini</p>
        </div>
      </footer>
    </div>
  );
}
