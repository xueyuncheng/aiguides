'use client';

import { useEffect, useRef } from 'react';
import type * as Phaser from 'phaser';
import { createGameConfig } from '../game/createGameConfig';
import type { GameScene } from '../game/GameScene';
import type { GameSnapshot } from '../game/state';
import type { GameSceneHandle } from '../types';

interface PhaserGameCanvasProps {
  onStateChange: (state: GameSnapshot) => void;
  sceneRef: React.MutableRefObject<GameSceneHandle | null>;
}

export function PhaserGameCanvas({ onStateChange, sceneRef }: PhaserGameCanvasProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const gameRef = useRef<Phaser.Game | null>(null);

  useEffect(() => {
    if (!containerRef.current || gameRef.current) {
      return;
    }

    let destroyed = false;

    const loadGame = async () => {
      const PhaserModule = await import('phaser');
      if (destroyed || !containerRef.current) {
        return;
      }

      const config = createGameConfig(containerRef.current, onStateChange);
      const game = new PhaserModule.Game(config);
      gameRef.current = game;

      game.events.once(PhaserModule.Core.Events.READY, () => {
        sceneRef.current = game.scene.getScene('GameScene') as GameScene & GameSceneHandle;
      });
    };

    void loadGame();

    return () => {
      destroyed = true;
      sceneRef.current = null;
      gameRef.current?.destroy(true);
      gameRef.current = null;
    };
  }, [onStateChange, sceneRef]);

  return <div ref={containerRef} className="h-full w-full" />;
}
