'use client';

import { useState, useEffect, useRef } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '../../contexts/AuthContext';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import SessionSidebar from '../../components/SessionSidebar';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}



interface AgentInfo {
  id: string;
  name: string;
  description: string;
  icon: string;
  color: string;
  examples: string[];
}

const agentInfoMap: Record<string, AgentInfo> = {
  assistant: {
    id: 'assistant',
    name: 'AI Assistant',
    description: '信息检索和事实核查',
    icon: '🔍',
    color: 'bg-blue-500',
    examples: [
      '什么是量子计算？',
      '请帮我查找关于人工智能的最新研究',
      '核查一下这个新闻是否准确...',
    ],
  },
  web_summary: {
    id: 'web_summary',
    name: 'WebSummary Agent',
    description: '网页内容分析',
    icon: '🌐',
    color: 'bg-green-500',
    examples: [
      '请帮我总结这个网页：https://example.com',
      '分析这篇文章的主要内容',
      '提取网页的关键信息',
    ],
  },
  email_summary: {
    id: 'email_summary',
    name: 'EmailSummary Agent',
    description: '邮件智能总结',
    icon: '📧',
    color: 'bg-purple-500',
    examples: [
      '请帮我总结收件箱中的重要邮件',
      '获取最近20封邮件并总结',
      '分析哪些邮件需要优先处理',
    ],
  },
  travel: {
    id: 'travel',
    name: 'Travel Agent',
    description: '旅游规划助手',
    icon: '✈️',
    color: 'bg-orange-500',
    examples: [
      '我计划去日本东京旅游5天，请帮我制定详细的旅游计划',
      '想在泰国曼谷玩3天，预算有限，请推荐经济实惠的行程',
      '帮我规划一个巴黎7日游，我对艺术和美食特别感兴趣',
    ],
  },
};

export default function ChatPage() {
  const params = useParams();
  const router = useRouter();
  const { user, loading } = useAuth();
  const agentId = params.agentId as string;
  const agentInfo = agentInfoMap[agentId];

  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string>('');
  const [isLoadingHistory, setIsLoadingHistory] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
      return;
    }

    if (!agentInfo) {
      router.push('/');
      return;
    }
    // Generate a simple session ID
    setSessionId(`session-${Date.now()}-${Math.random().toString(36).substring(7)}`);
  }, [agentId, agentInfo, router]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const sendMessage = async (content: string) => {
    if (!content.trim() || isLoading) return;

    const userMessage: Message = {
      id: `msg-${Date.now()}`,
      role: 'user',
      content: content.trim(),
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInputValue('');
    setIsLoading(true);

    try {
      const response = await fetch(`/api/${agentId}/chats/${sessionId}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include', // Include cookies for authentication
        body: JSON.stringify({
          user_id: user?.user_id,
          session_id: sessionId,
          message: content.trim(),
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const reader = response.body?.getReader();
      const decoder = new TextDecoder();
      let assistantContent = '';

      if (reader) {
        // 1. 初始化 AI 的空消息占位
        const assistantMessage: Message = {
          id: `msg-${Date.now()}-assistant`,
          role: 'assistant',
          content: '',
          timestamp: new Date(),
        };
        setMessages((prev) => [...prev, assistantMessage]);

        // 2. 核心修复：定义缓冲区，用于处理 TCP 分包导致的数据截断
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          // 3. 解码数据块 (stream: true 保持流式解码状态)
          const chunk = decoder.decode(value, { stream: true });
          buffer += chunk;

          // 4. 按换行符分割数据
          const lines = buffer.split('\n');

          // 5. 将最后一行（可能不完整）留到下一次循环处理
          // pop() 会移除数组最后一个元素并返回它
          buffer = lines.pop() || '';

          for (const line of lines) {
            const trimmedLine = line.trim();
            // 6. 解析 SSE 格式：只处理以 "data:" 开头的行
            if (trimmedLine.startsWith('data:')) {
              try {
                // 去掉 "data:" 前缀并解析 JSON
                const jsonStr = trimmedLine.substring(5).trim();
                if (!jsonStr) continue;

                const data = JSON.parse(jsonStr);

                // 7. 更新 UI 状态
                if (data.content) {
                  assistantContent += data.content;

                  // 使用函数式更新，确保总是获取到最新的 messages 数组
                  setMessages((prev) => {
                    const newMessages = [...prev];
                    // 找到最后一条消息（即当前正在生成的 AI 消息）并更新它
                    const lastIndex = newMessages.length - 1;
                    if (lastIndex >= 0 && newMessages[lastIndex].role === 'assistant') {
                      newMessages[lastIndex] = {
                        ...newMessages[lastIndex],
                        content: assistantContent,
                      };
                    }
                    return newMessages;
                  });
                }
              } catch (e) {
                console.warn('JSON parse error, skipping line:', trimmedLine, e);
              }
            }
          }
        }
      }
    } catch (error) {
      console.error('Error sending message:', error);
      const errorMessage: Message = {
        id: `msg-${Date.now()}-error`,
        role: 'assistant',
        content: '抱歉，发生了错误。请确保后端服务正在运行，并稍后重试。\n\n错误详情：' + (error instanceof Error ? error.message : String(error)),
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, errorMessage]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    sendMessage(inputValue);
  };

  const handleExampleClick = (example: string) => {
    setInputValue(example);
  };

  if (loading || !agentInfo) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="w-16 h-16 border-4 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
      {/* Session Sidebar */}
      <SessionSidebar
        agentId={agentId}
        userId={user?.user_id || ''}
        currentSessionId={sessionId}
        onSessionSelect={handleSessionSelect}
        onNewSession={handleNewSession}
        onDeleteSession={handleDeleteSession}
      />

      {/* Main Content */}
      <div className="flex flex-col flex-1 ml-80">
        {/* Header */}
        <header className="border-b bg-white dark:bg-gray-800 shadow-sm">
          <div className="container mx-auto px-4 py-4 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <button
                onClick={() => router.push('/')}
                className="text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
              >
                ← 返回
              </button>
              <div className="flex items-center gap-3">
                <div className={`text-3xl p-2 rounded-lg ${agentInfo.color} bg-opacity-10`}>
                  {agentInfo.icon}
                </div>
                <div>
                  <h1 className="text-xl font-bold text-gray-900 dark:text-white">
                    {agentInfo.name}
                  </h1>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    {agentInfo.description}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </header>

        {/* Loading History Overlay */}
        {isLoadingHistory && (
          <div className="absolute inset-0 bg-gray-900/50 flex items-center justify-center z-40">
            <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow-xl">
              <div className="flex items-center gap-3">
                <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
                <span className="text-gray-900 dark:text-white">加载会话历史...</span>
              </div>
            </div>
          </div>
        )}

        {/* Messages Area */}
        <div className="flex-1 overflow-y-auto">
          <div className="container mx-auto px-4 py-6 max-w-4xl">
            {messages.length === 0 ? (
              <div className="text-center py-12">
                <div className="text-6xl mb-4">{agentInfo.icon}</div>
                <h2 className="text-2xl font-semibold text-gray-800 dark:text-gray-200 mb-2">
                  开始与 {agentInfo.name} 对话
                </h2>
                <p className="text-gray-600 dark:text-gray-400 mb-8">
                  尝试以下示例问题，或输入您自己的问题
                </p>
                <div className="grid grid-cols-1 gap-3 max-w-2xl mx-auto">
                  {agentInfo.examples.map((example, index) => (
                    <button
                      key={index}
                      onClick={() => handleExampleClick(example)}
                      className="p-4 text-left bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-gray-400 dark:hover:border-gray-500 transition-colors"
                    >
                      <p className="text-gray-700 dark:text-gray-300">{example}</p>
                    </button>
                  ))}
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                {messages.map((message) => (
                  <div
                    key={message.id}
                    className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
                  >
                    <div
                      className={`max-w-[80%] rounded-lg px-4 py-3 ${message.role === 'user'
                        ? 'bg-blue-500 text-white'
                        : 'bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border border-gray-200 dark:border-gray-700'
                        }`}
                    >
                      <div className="break-words">
                        {message.role === 'assistant' ? (
                          <div className="prose prose-sm dark:prose-invert max-w-none">
                            <ReactMarkdown
                              remarkPlugins={[remarkGfm]}
                              components={{
                                // Customize link rendering to open in new tab
                                a: ({ ...props }) => (
                                  <a {...props} target="_blank" rel="noopener noreferrer" className="text-blue-600 dark:text-blue-400 hover:underline" />
                                ),
                                // Customize code blocks
                                code: (props) => {
                                  const { children, className, ...rest } = props;
                                  // Code blocks have language classes like 'language-javascript'
                                  const isInline = !className || !className.startsWith('language-');
                                  return isInline ? (
                                    <code {...rest} className="bg-gray-100 dark:bg-gray-700 px-1 py-0.5 rounded text-sm">
                                      {children}
                                    </code>
                                  ) : (
                                    <pre className="bg-gray-100 dark:bg-gray-700 p-2 rounded text-sm overflow-x-auto">
                                      <code {...rest} className={className}>
                                        {children}
                                      </code>
                                    </pre>
                                  );
                                },
                                // Customize list styling
                                ul: ({ ...props }) => (
                                  <ul {...props} className="list-disc list-inside space-y-1" />
                                ),
                                ol: ({ ...props }) => (
                                  <ol {...props} className="list-decimal list-inside space-y-1" />
                                ),
                                // Customize heading styles
                                h1: ({ ...props }) => (
                                  <h1 {...props} className="text-2xl font-bold mt-4 mb-2" />
                                ),
                                h2: ({ ...props }) => (
                                  <h2 {...props} className="text-xl font-bold mt-3 mb-2" />
                                ),
                                h3: ({ ...props }) => (
                                  <h3 {...props} className="text-lg font-bold mt-2 mb-1" />
                                ),
                                // Customize paragraph spacing
                                p: ({ ...props }) => (
                                  <p {...props} className="mb-2" />
                                ),
                              }}
                            >
                              {message.content}
                            </ReactMarkdown>
                          </div>
                        ) : (
                          <div className="whitespace-pre-wrap">{message.content}</div>
                        )}
                      </div>
                      <div
                        className={`text-xs mt-2 ${message.role === 'user' ? 'text-blue-100' : 'text-gray-500 dark:text-gray-400'
                          }`}
                      >
                        {message.timestamp.toLocaleTimeString('zh-CN', {
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </div>
                    </div>
                  </div>
                ))}
                {isLoading && (
                  <div className="flex justify-start">
                    <div className="max-w-[80%] rounded-lg px-4 py-3 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700">
                      <div className="flex items-center gap-2">
                        <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }}></div>
                        <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }}></div>
                        <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }}></div>
                      </div>
                    </div>
                  </div>
                )}
                <div ref={messagesEndRef} />
              </div>
            )}
          </div>
        </div>

        {/* Input Area */}
        <div className="border-t bg-white dark:bg-gray-800 shadow-lg">
          <div className="container mx-auto px-4 py-4 max-w-4xl">
            <form onSubmit={handleSubmit} className="flex gap-2">
              <input
                type="text"
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                placeholder="输入您的消息..."
                className="flex-1 px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                disabled={isLoading}
              />
              <button
                type="submit"
                disabled={isLoading || !inputValue.trim()}
                className="px-6 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-300 dark:disabled:bg-gray-600 disabled:cursor-not-allowed transition-colors font-medium"
              >
                {isLoading ? '发送中...' : '发送'}
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
