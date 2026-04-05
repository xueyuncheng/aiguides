import type { LevelConfig } from './types';

export const sunsetSprintLevel: LevelConfig = {
  id: 'sunset-sprint',
  name: '第一关: 蘑菇大道',
  tagline: '蓝天砖块区先热身，踩过巡逻小怪和浮空奖励台。',
  worldWidth: 2400,
  floorY: 500,
  playerStart: { x: 140, y: 420 },
  goalX: 2230,
  platforms: [
    { x: 180, y: 450, width: 220, height: 20, theme: 'moss' },
    { x: 470, y: 390, width: 180, height: 20, theme: 'moss' },
    { x: 760, y: 330, width: 160, height: 20, theme: 'cloud' },
    { x: 1040, y: 410, width: 260, height: 20, theme: 'moss' },
    { x: 1410, y: 360, width: 180, height: 20, theme: 'cloud' },
    { x: 1670, y: 295, width: 150, height: 20, theme: 'cloud' },
    { x: 1890, y: 365, width: 220, height: 20, theme: 'moss' },
    {
      x: 1290,
      y: 295,
      width: 140,
      height: 18,
      theme: 'cloud',
      movement: { axis: 'x', distance: 120, duration: 3600, phase: 0.15 },
    },
  ],
  coins: [
    { x: 260, y: 395 },
    { x: 540, y: 335 },
    { x: 825, y: 275 },
    { x: 1125, y: 355 },
    { x: 1320, y: 235 },
    { x: 1495, y: 305 },
    { x: 1740, y: 240 },
    { x: 1960, y: 310 },
    { x: 2140, y: 445 },
  ],
  enemies: [
    { x: 620, y: 460, minX: 520, maxX: 760, speed: 60 },
    { x: 1540, y: 470, minX: 1480, maxX: 1810, speed: 75 },
  ],
  hazards: [
    { x: 915, y: 488, width: 74, height: 12, kind: 'spikes' },
    { x: 1815, y: 488, width: 86, height: 12, kind: 'spikes' },
  ],
  checkpoints: [
    { x: 1130, y: 430, label: '砖桥中段' },
    { x: 1885, y: 385, label: '旗台前哨' },
  ],
  theme: {
    sky: 0x7dd3ff,
    mist: 0xfff2c4,
    hillNear: 0x67b84d,
    hillFar: 0x4fa03f,
    accent: 0xe55433,
    ground: 0x7a4a20,
  },
};