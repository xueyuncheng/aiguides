'use client';

import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Clock, AlertCircle, Lock } from 'lucide-react';
import { AIAvatar, AIMessageContent, UserMessage } from '@/app/chat/[sessionId]/components';
import { agentInfoMap } from '@/app/chat/[sessionId]/constants';
import type { Message } from '@/app/chat/[sessionId]/types';

interface SharedConversationResponse {
  share_id: string;
  session_id: string;
  app_name: string;
  messages: Message[];
  expires_at: string;
  is_expired: boolean;
}

export default function SharedConversationPage() {
  const params = useParams();
  const router = useRouter();
  const shareId = params.shareId as string;
  const [data, setData] = useState<SharedConversationResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchSharedConversation = async () => {
      try {
        const response = await fetch(`/api/share/${shareId}`);
        
        if (!response.ok) {
          if (response.status === 404) {
            setError('This shared conversation was not found.');
          } else if (response.status === 410) {
            const errorData = await response.json();
            setError(`This shared link has expired. It expired on ${new Date(errorData.expires_at).toLocaleDateString()}.`);
          } else {
            setError('Failed to load shared conversation.');
          }
          return;
        }

        const result = await response.json();
        setData(result);
      } catch (err) {
        console.error('Error fetching shared conversation:', err);
        setError('An error occurred while loading the conversation.');
      } finally {
        setIsLoading(false);
      }
    };

    if (shareId) {
      fetchSharedConversation();
    }
  }, [shareId]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="text-center">
          <div className="w-16 h-16 border-4 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">Loading shared conversation...</p>
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 p-4">
        <div className="max-w-md w-full bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8 text-center">
          <div className="w-16 h-16 bg-red-100 dark:bg-red-900/20 rounded-full flex items-center justify-center mx-auto mb-4">
            <AlertCircle className="w-8 h-8 text-red-600 dark:text-red-400" />
          </div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
            Unable to Load Conversation
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mb-6">
            {error || 'This shared conversation could not be found.'}
          </p>
          <button
            onClick={() => router.push('/')}
            className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
          >
            Go to Home
          </button>
        </div>
      </div>
    );
  }

  const agentInfo = agentInfoMap[data.app_name] || agentInfoMap['assistant'];
  const expiresAt = new Date(data.expires_at);

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <header className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 sticky top-0 z-10">
        <div className="max-w-5xl mx-auto px-4 py-4 sm:px-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/20 rounded-lg flex items-center justify-center">
                <span className="text-2xl">{agentInfo.icon}</span>
              </div>
              <div>
                <h1 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Shared Conversation
                </h1>
                <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                  <Lock className="w-3 h-3" />
                  <span>Read-only view</span>
                </div>
              </div>
            </div>
            <button
              onClick={() => router.push('/')}
              className="px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
            >
              Visit AIGuides
            </button>
          </div>
        </div>
      </header>

      {/* Expiry Notice */}
      <div className="bg-yellow-50 dark:bg-yellow-900/20 border-b border-yellow-200 dark:border-yellow-800">
        <div className="max-w-5xl mx-auto px-4 py-3 sm:px-6">
          <div className="flex items-center gap-2 text-sm text-yellow-800 dark:text-yellow-200">
            <Clock className="w-4 h-4" />
            <span>
              This shared link will expire on{' '}
              <strong>{expiresAt.toLocaleDateString()}</strong> at{' '}
              <strong>{expiresAt.toLocaleTimeString()}</strong>
            </span>
          </div>
        </div>
      </div>

      {/* Messages */}
      <main className="max-w-5xl mx-auto px-4 py-8 sm:px-6">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {data.messages.length === 0 ? (
              <div className="p-12 text-center text-gray-500 dark:text-gray-400">
                This conversation has no messages.
              </div>
            ) : (
              data.messages.map((message, index) => (
                <div
                  key={message.id || index}
                  className="p-6 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
                >
                  {message.role === 'user' ? (
                    <UserMessage
                      content={message.content}
                      images={message.images}
                      fileNames={message.fileNames}
                    />
                  ) : (
                    <div className="flex gap-4">
                      <div className="flex-shrink-0">
                        <AIAvatar icon={agentInfo.icon} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <AIMessageContent
                          content={message.content}
                          thought={message.thought}
                          images={message.images}
                        />
                      </div>
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>

        {/* Footer Info */}
        <div className="mt-8 text-center text-sm text-gray-500 dark:text-gray-400">
          <p>
            This is a readonly view of a shared conversation from{' '}
            <button
              onClick={() => router.push('/')}
              className="text-blue-600 dark:text-blue-400 hover:underline"
            >
              AIGuides
            </button>
          </p>
        </div>
      </main>
    </div>
  );
}
