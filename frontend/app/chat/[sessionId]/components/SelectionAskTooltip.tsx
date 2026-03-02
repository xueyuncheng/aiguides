'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { MessageSquare } from 'lucide-react';

interface TooltipState {
  visible: boolean;
  x: number;
  y: number;
  text: string;
}

interface SelectionAskTooltipProps {
  onAskAI: (text: string) => void;
}

export function SelectionAskTooltip({ onAskAI }: SelectionAskTooltipProps) {
  const [tooltip, setTooltip] = useState<TooltipState>({ visible: false, x: 0, y: 0, text: '' });
  const tooltipRef = useRef<HTMLButtonElement>(null);

  const updateTooltip = useCallback(() => {
    const selection = window.getSelection();
    if (!selection || selection.isCollapsed) {
      setTooltip(prev => (prev.visible ? { ...prev, visible: false } : prev));
      return;
    }

    const selectedText = selection.toString().trim();
    if (!selectedText) {
      setTooltip(prev => (prev.visible ? { ...prev, visible: false } : prev));
      return;
    }

    // Check if the selection is within a data-ai-message container
    const anchorNode = selection.anchorNode;
    if (!anchorNode) return;

    let currentNode: Node | null = anchorNode;
    let inAiMessage = false;
    while (currentNode) {
      if (currentNode instanceof Element && currentNode.hasAttribute('data-ai-message')) {
        inAiMessage = true;
        break;
      }
      currentNode = currentNode.parentNode;
    }

    if (!inAiMessage) {
      setTooltip(prev => (prev.visible ? { ...prev, visible: false } : prev));
      return;
    }

    const range = selection.getRangeAt(0);
    const rect = range.getBoundingClientRect();

    setTooltip({
      visible: true,
      x: rect.left + rect.width / 2,
      y: rect.top,
      text: selectedText,
    });
  }, []);

  useEffect(() => {
    document.addEventListener('mouseup', updateTooltip);
    return () => {
      document.removeEventListener('mouseup', updateTooltip);
    };
  }, [updateTooltip]);

  useEffect(() => {
    const handleSelectionChange = () => {
      const selection = window.getSelection();
      if (!selection || selection.isCollapsed) {
        setTooltip(prev => (prev.visible ? { ...prev, visible: false } : prev));
      }
    };
    document.addEventListener('selectionchange', handleSelectionChange);
    return () => {
      document.removeEventListener('selectionchange', handleSelectionChange);
    };
  }, []);

  const handleClick = useCallback(() => {
    onAskAI(tooltip.text);
    setTooltip(prev => ({ ...prev, visible: false }));
    window.getSelection()?.removeAllRanges();
  }, [onAskAI, tooltip.text]);

  if (!tooltip.visible) return null;

  return (
    <button
      ref={tooltipRef}
      // Prevent default on mousedown so the text selection isn't cleared before click fires
      onMouseDown={(e) => e.preventDefault()}
      onClick={handleClick}
      style={{
        position: 'fixed',
        left: `${tooltip.x}px`,
        top: `${tooltip.y}px`,
        transform: 'translate(-50%, calc(-100% - 8px))',
        zIndex: 50,
      }}
      className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 rounded-full shadow-lg hover:bg-zinc-700 dark:hover:bg-zinc-200 transition-colors select-none whitespace-nowrap"
      aria-label="问 AI"
    >
      <MessageSquare className="h-3.5 w-3.5" />
      问 AI
    </button>
  );
}
