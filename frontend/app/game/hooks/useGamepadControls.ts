'use client';

import { useEffect, useRef, useState } from 'react';
import type { GameSnapshot } from '../game/state';
import type { GameSceneHandle, GamepadDebugInfo } from '../types';

export function useGamepadControls(sceneRef: React.RefObject<GameSceneHandle | null>, gameState: GameSnapshot) {
  const animationFrameRef = useRef<number | null>(null);
  const lastDebugUpdateRef = useRef(0);
  const previousStartPressedRef = useRef(false);
  const gameStateRef = useRef<GameSnapshot>(gameState);
  const [gamepadConnected, setGamepadConnected] = useState(false);
  const [gamepadDebug, setGamepadDebug] = useState<GamepadDebugInfo[]>([]);

  useEffect(() => {
    gameStateRef.current = gameState;
  }, [gameState]);

  useEffect(() => {
    if (typeof window === 'undefined' || typeof navigator === 'undefined') {
      return;
    }

    let mounted = true;
    let latestScene: GameSceneHandle | null = null;

    const getScene = () => {
      latestScene = sceneRef.current;
      return latestScene;
    };

    const readActiveGamepad = () => {
      const pads = Array.from(navigator.getGamepads?.() ?? []);
      return pads.find((candidate) => candidate?.connected) ?? null;
    };

    const syncDebugState = () => {
      if (!mounted) {
        return;
      }

      const pads = Array.from(navigator.getGamepads?.() ?? []).filter(
        (candidate): candidate is Gamepad => Boolean(candidate?.connected)
      );

      setGamepadDebug(pads.map(toGamepadDebugInfo));
    };

    const syncConnectionState = () => {
      if (!mounted) {
        return null;
      }

      const activePad = readActiveGamepad();
      setGamepadConnected((current) => (current === Boolean(activePad) ? current : Boolean(activePad)));
      return activePad;
    };

    const updateGamepad = () => {
      const pad = syncConnectionState();
      const now = performance.now();
      const scene = getScene();

      if (now - lastDebugUpdateRef.current > 150) {
        syncDebugState();
        lastDebugUpdateRef.current = now;
      }

      if (!pad) {
        scene?.setGamepadInput({ left: false, right: false, jump: false });
        previousStartPressedRef.current = false;
        animationFrameRef.current = window.requestAnimationFrame(updateGamepad);
        return;
      }

      const horizontalAxis = pad.axes[0] ?? 0;
      const moveLeft = horizontalAxis < -0.35 || Boolean(pad.buttons[14]?.pressed);
      const moveRight = horizontalAxis > 0.35 || Boolean(pad.buttons[15]?.pressed);
      const jump = Boolean(pad.buttons[0]?.pressed || pad.buttons[1]?.pressed);
      const startPressed = Boolean(pad.buttons[9]?.pressed);

      scene?.setGamepadInput({ left: moveLeft, right: moveRight, jump });

      if (startPressed && !previousStartPressedRef.current) {
        const currentState = gameStateRef.current.status;
        if (currentState === 'paused') {
          scene?.resumeGame();
        } else if (currentState === 'running') {
          scene?.pauseGame();
        }
      }

      previousStartPressedRef.current = startPressed;
      animationFrameRef.current = window.requestAnimationFrame(updateGamepad);
    };

    const handleGamepadConnected = () => {
      syncConnectionState();
      syncDebugState();
    };

    const handleGamepadDisconnected = () => {
      syncConnectionState();
      syncDebugState();
    };

    syncConnectionState();
    syncDebugState();
    window.addEventListener('gamepadconnected', handleGamepadConnected);
    window.addEventListener('gamepaddisconnected', handleGamepadDisconnected);

    animationFrameRef.current = window.requestAnimationFrame(updateGamepad);

    return () => {
      mounted = false;
      window.removeEventListener('gamepadconnected', handleGamepadConnected);
      window.removeEventListener('gamepaddisconnected', handleGamepadDisconnected);
      if (animationFrameRef.current !== null) {
        window.cancelAnimationFrame(animationFrameRef.current);
      }
      setGamepadDebug([]);
      latestScene?.setGamepadInput({ left: false, right: false, jump: false });
    };
  }, [sceneRef]);

  return {
    gamepadConnected,
    gamepadDebug,
  };
}

function toGamepadDebugInfo(gamepad: Gamepad): GamepadDebugInfo {
  const axes = gamepad.axes.slice(0, 4).map((axis) => axis.toFixed(2)).join(', ');
  const pressedButtons = gamepad.buttons
    .map((button, index) => (button.pressed ? index : null))
    .filter((index): index is number => index !== null)
    .join(', ');

  return {
    index: gamepad.index,
    id: gamepad.id || 'Unknown Controller',
    mapping: gamepad.mapping || 'none',
    axes: axes || '0.00, 0.00, 0.00, 0.00',
    pressedButtons: pressedButtons || '无',
  };
}
