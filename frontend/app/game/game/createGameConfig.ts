import * as Phaser from 'phaser';
import { GameScene } from './GameScene';
import { GAME_HEIGHT, GAME_WIDTH } from './level';
import type { GameSnapshot } from './state';

export function createGameConfig(container: HTMLElement, onStateChange: (state: GameSnapshot) => void) {
  return {
    type: Phaser.AUTO,
    width: GAME_WIDTH,
    height: GAME_HEIGHT,
    parent: container,
    backgroundColor: '#4f46e5',
    physics: {
      default: 'arcade',
      arcade: {
        gravity: { y: 920, x: 0 },
        debug: false,
      },
    },
    scale: {
      mode: Phaser.Scale.FIT,
      autoCenter: Phaser.Scale.CENTER_BOTH,
      width: GAME_WIDTH,
      height: GAME_HEIGHT,
    },
    scene: [new GameScene(onStateChange)],
    render: {
      pixelArt: false,
      antialias: true,
    },
    callbacks: {
      postBoot: (game: Phaser.Game) => {
        game.scale.resize(GAME_WIDTH, GAME_HEIGHT);
        game.canvas.style.width = '100%';
        game.canvas.style.height = '100%';
        game.canvas.style.maxWidth = '100%';
        game.canvas.style.borderRadius = '20px';
      },
    },
  } satisfies Phaser.Types.Core.GameConfig;
}
