'use client';

import { useState, useEffect } from 'react';
import { Check, Copy, Share2, X } from 'lucide-react';
import { Button } from '@/app/components/ui/button';
import { useAuth } from '@/app/contexts/AuthContext';

interface ShareModalProps {
  isOpen: boolean;
  onClose: () => void;
  sessionId: string;
  agentId: string;
}

interface ShareResponse {
  share_id: string;
  share_url: string;
  expires_at: string;
}

export function ShareModal({ isOpen, onClose, sessionId, agentId }: ShareModalProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [shareData, setShareData] = useState<ShareResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const { authenticatedFetch } = useAuth();

  useEffect(() => {
    if (!isOpen) {
      setShareData(null);
      setError(null);
      setCopied(false);
    }
  }, [isOpen]);

  const createShareLink = async () => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await authenticatedFetch('/api/assistant/share', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          session_id: sessionId,
          agent_id: agentId,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to create share link');
      }

      const data = await response.json();
      setShareData(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setIsLoading(false);
    }
  };

  const copyToClipboard = async () => {
    if (!shareData) return;

    const fullUrl = `${window.location.origin}${shareData.share_url}`;
    
    try {
      await navigator.clipboard.writeText(fullUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
            Share Conversation
          </h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {!shareData ? (
          <div className="space-y-4">
            <p className="text-sm text-gray-600 dark:text-gray-300">
              Create a shareable link that allows anyone to view this conversation without logging in.
              The link will expire in 7 days.
            </p>
            
            {error && (
              <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
              </div>
            )}

            <div className="flex gap-2 justify-end">
              <Button
                variant="outline"
                onClick={onClose}
                disabled={isLoading}
              >
                Cancel
              </Button>
              <Button
                onClick={createShareLink}
                disabled={isLoading}
                className="flex items-center gap-2"
              >
                {isLoading ? (
                  <>
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    Creating...
                  </>
                ) : (
                  <>
                    <Share2 className="w-4 h-4" />
                    Create Share Link
                  </>
                )}
              </Button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <p className="text-sm text-gray-600 dark:text-gray-300">
              Share link created successfully! Anyone with this link can view the conversation.
            </p>

            <div className="p-3 bg-gray-50 dark:bg-gray-700 rounded-md">
              <p className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                Share URL
              </p>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  readOnly
                  value={`${window.location.origin}${shareData.share_url}`}
                  className="flex-1 px-3 py-2 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <Button
                  size="sm"
                  onClick={copyToClipboard}
                  className="flex items-center gap-1"
                >
                  {copied ? (
                    <>
                      <Check className="w-4 h-4" />
                      Copied
                    </>
                  ) : (
                    <>
                      <Copy className="w-4 h-4" />
                      Copy
                    </>
                  )}
                </Button>
              </div>
            </div>

            <div className="p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-md">
              <p className="text-xs text-yellow-800 dark:text-yellow-200">
                This link will expire on{' '}
                <strong>{new Date(shareData.expires_at).toLocaleDateString()}</strong> at{' '}
                <strong>{new Date(shareData.expires_at).toLocaleTimeString()}</strong>
              </p>
            </div>

            <div className="flex justify-end">
              <Button onClick={onClose}>Close</Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
