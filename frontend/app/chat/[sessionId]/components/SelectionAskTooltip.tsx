'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { MessageSquare } from 'lucide-react';

interface TooltipState {
  visible: boolean;
  x: number;
  y: number;
  text: string;
  placement: 'top' | 'bottom';
}

interface SelectionAskTooltipProps {
  onAskAI: (text: string) => void;
}

export function SelectionAskTooltip({ onAskAI }: SelectionAskTooltipProps) {
  const [tooltip, setTooltip] = useState<TooltipState>({ visible: false, x: 0, y: 0, text: '', placement: 'top' });
  const tooltipRef = useRef<HTMLDivElement>(null);

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
    const estimatedWidth = tooltipRef.current?.offsetWidth ?? 240;
    const viewportPadding = 16;
    const placement = rect.top < 96 ? 'bottom' : 'top';
    const nextX = Math.min(
      Math.max(rect.left + rect.width / 2, viewportPadding + estimatedWidth / 2),
      window.innerWidth - viewportPadding - estimatedWidth / 2,
    );
    const nextY = placement === 'top' ? rect.top : rect.bottom;

    setTooltip({
      visible: true,
      x: nextX,
      y: nextY,
      text: selectedText,
      placement,
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

  const isTop = tooltip.placement === 'top';

  return (
    <div
      ref={tooltipRef}
      style={{
        position: 'fixed',
        left: `${tooltip.x}px`,
        top: `${tooltip.y}px`,
        transform: isTop ? 'translate(-50%, calc(-100% - 14px))' : 'translate(-50%, 14px)',
        zIndex: 50,
      }}
    >
      <button
        // Prevent default on mousedown so the text selection isn't cleared before click fires
        onMouseDown={(e) => e.preventDefault()}
        onClick={handleClick}
        className="group relative flex items-center gap-1.5 rounded-xl border border-zinc-300/70 bg-white/96 px-3 py-2 text-left shadow-[0_10px_24px_-18px_rgba(0,0,0,0.28)] backdrop-blur-md transition-all duration-150 hover:border-zinc-400/80 hover:bg-white dark:border-zinc-700/80 dark:bg-zinc-900/96 dark:hover:border-zinc-600/90 dark:hover:bg-zinc-900"
        aria-label="问 AI"
      >
        <MessageSquare className="h-3.5 w-3.5 text-zinc-600 dark:text-zinc-300" />
        <span className="pr-0.5 text-sm font-medium text-zinc-800 dark:text-zinc-100">问 AI</span>
      </button>

      <div
        className="pointer-events-none absolute left-1/2 h-2.5 w-2.5 -translate-x-1/2 rotate-45 border-zinc-300/70 bg-white/96 dark:border-zinc-700/80 dark:bg-zinc-900/96"
        style={isTop ? { bottom: '-6px', borderRightWidth: '1px', borderBottomWidth: '1px' } : { top: '-6px', borderTopWidth: '1px', borderLeftWidth: '1px' }}
      />
    </div>
  );
}
