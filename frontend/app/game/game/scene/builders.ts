import * as Phaser from 'phaser';
import { GAME_HEIGHT, type LevelConfig, type PlatformTheme } from '../levels';
import type { CheckpointInstance, EnemyInstance, MovingPlatformInstance } from './types';

const PLAYER_SIZE = { width: 30, height: 44 };

export function drawBackdrop(scene: Phaser.Scene, level: LevelConfig, levelIndex: number) {
  scene.add.rectangle(level.worldWidth / 2, GAME_HEIGHT / 2, level.worldWidth, GAME_HEIGHT, level.theme.sky);
  scene.add.rectangle(level.worldWidth / 2, 110, level.worldWidth, 160, level.theme.mist, levelIndex === 0 ? 0.28 : 0.18);

  const sun = scene.add.circle(levelIndex === 0 ? 170 : 240, 108, levelIndex === 0 ? 54 : 44, 0xfff3a8, 0.96);
  sun.setStrokeStyle(10, 0xffd54d, 0.35);

  for (const [cloudX, cloudY] of [
    [170, 122],
    [520, 94],
    [900, 136],
    [1280, 108],
    [1660, 128],
    [2040, 102],
    [2420, 140],
  ]) {
    drawCloud(scene, cloudX, cloudY, levelIndex === 0 ? 0.95 : 0.72);
  }

  for (const hill of [250, 760, 1320, 1860, 2460]) {
    drawHill(scene, hill, level.floorY, level.theme.hillFar, 250, 124);
    drawHill(scene, hill + 170, level.floorY, level.theme.hillNear, 320, 172);
  }

  for (const bush of [140, 470, 920, 1430, 1810, 2240, 2690]) {
    drawBush(scene, bush, level.floorY - 14, level.theme.hillNear);
  }

  scene.add.rectangle(level.worldWidth / 2, level.floorY - 5, level.worldWidth, 14, 0x7ccc4b, 0.9);

  if (levelIndex === 1) {
    drawCastleSilhouette(scene, level.worldWidth - 240, level.floorY - 32, 0x3e241a);
  }
}

export function createPlatforms(scene: Phaser.Scene, level: LevelConfig) {
  ensurePlatformTextures(scene);

  const groundHeight = GAME_HEIGHT - level.floorY;
  const platforms = scene.physics.add.staticGroup();
  const movingPlatforms: MovingPlatformInstance[] = [];

  const floor = scene.add.tileSprite(
    level.worldWidth / 2,
    level.floorY + groundHeight / 2,
    level.worldWidth,
    groundHeight,
    'ground-tile'
  );
  scene.physics.add.existing(floor, true);
  platforms.add(floor as unknown as Phaser.Physics.Arcade.Image);

  for (const platform of level.platforms) {
    const textureKey = getPlatformTextureKey(platform.theme);

    if (platform.movement) {
      const block = scene.add.tileSprite(
        platform.x + platform.width / 2,
        platform.y + platform.height / 2,
        platform.width,
        platform.height,
        textureKey
      );
      scene.physics.add.existing(block, true);

      const body = block.body as Phaser.Physics.Arcade.StaticBody;

      movingPlatforms.push({
        block,
        body,
        config: platform,
        originX: block.x,
        originY: block.y,
        previousX: block.x,
        previousY: block.y,
        deltaX: 0,
        deltaY: 0,
      });
      continue;
    }

    const block = scene.add.tileSprite(
      platform.x + platform.width / 2,
      platform.y + platform.height / 2,
      platform.width,
      platform.height,
      textureKey
    );
    scene.physics.add.existing(block, true);
    platforms.add(block as unknown as Phaser.Physics.Arcade.Image);
  }

  return { platforms, movingPlatforms };
}

export function createGoal(scene: Phaser.Scene, level: LevelConfig) {
  ensurePlatformTextures(scene);

  scene.add.tileSprite(level.goalX + 54, level.floorY - 18, 120, 36, 'platform-ember');
  const tower = scene.add.tileSprite(level.goalX + 100, level.floorY - 106, 56, 140, 'platform-ember');
  const towerTrim = scene.add.rectangle(level.goalX + 100, level.floorY - 174, 68, 16, 0xc27a3c);

  for (const offset of [-18, 0, 18]) {
    scene.add.rectangle(level.goalX + 82 + offset, level.floorY - 190, 14, 18, 0xc27a3c);
  }

  const pole = scene.add.rectangle(level.goalX + 6, level.floorY - 112, 10, 208, 0xf8fafc).setOrigin(0, 0);
  pole.setStrokeStyle(2, 0xf8d16e, 0.72);

  const flag = scene.add.triangle(
    level.goalX + 70,
    level.floorY - 98,
    0,
    0,
    0,
    50,
    74,
    24,
    0xe55433,
    0.98
  );
  flag.setOrigin(0.1, 0.1);
  flag.setStrokeStyle(3, 0xfff8dd, 0.92);

  const badge = scene.add.circle(level.goalX + 44, level.floorY - 77, 9, 0xffef8f);
  badge.setStrokeStyle(2, 0xa8611d, 0.85);

  const glow = scene.add.circle(level.goalX + 42, level.floorY - 128, 34, level.theme.accent, 0.18);
  glow.setStrokeStyle(2, 0xfff8dd, 0.42);

  const finial = scene.add.circle(level.goalX + 5, level.floorY - 120, 8, 0xfff0a8);
  finial.setStrokeStyle(2, 0xb86b1d);

  const label = scene.add.text(level.goalX + 16, level.floorY - 154, 'FINISH', {
    color: '#fff8dd',
    fontFamily: 'monospace',
    fontSize: '13px',
    fontStyle: 'bold',
  });
  label.setShadow(0, 0, '#ff9b3d', 12, false, true);
  towerTrim.setDepth(tower.depth + 1);
}

export function createPlayer(
  scene: Phaser.Scene,
  level: LevelConfig,
  platforms: Phaser.Physics.Arcade.StaticGroup,
  movingPlatforms: MovingPlatformInstance[]
) {
  const textureKey = 'player-retro-runner';
  if (!scene.textures.exists(textureKey)) {
    const graphics = scene.make.graphics({ x: 0, y: 0, add: false });
    graphics.fillStyle(0xc63228);
    graphics.fillRect(7, 2, 22, 8);
    graphics.fillRect(5, 8, 26, 4);
    graphics.fillStyle(0x8f241d);
    graphics.fillRect(4, 10, 28, 3);
    graphics.fillStyle(0xf4c9a6);
    graphics.fillRect(10, 13, 16, 10);
    graphics.fillRect(8, 23, 20, 3);
    graphics.fillStyle(0x23273b);
    graphics.fillRect(13, 16, 2, 3);
    graphics.fillRect(21, 16, 2, 3);
    graphics.fillStyle(0xc63228);
    graphics.fillRect(6, 24, 7, 12);
    graphics.fillRect(23, 24, 7, 12);
    graphics.fillStyle(0x3263b4);
    graphics.fillRect(13, 24, 10, 14);
    graphics.fillRect(9, 28, 6, 10);
    graphics.fillRect(21, 28, 6, 10);
    graphics.fillStyle(0xf0d15b);
    graphics.fillRect(15, 28, 2, 2);
    graphics.fillRect(19, 28, 2, 2);
    graphics.fillStyle(0xffffff);
    graphics.fillRect(4, 28, 4, 8);
    graphics.fillRect(28, 28, 4, 8);
    graphics.fillStyle(0x6f4b30);
    graphics.fillRect(10, 38, 7, 8);
    graphics.fillRect(19, 38, 7, 8);
    graphics.generateTexture(textureKey, PLAYER_SIZE.width, PLAYER_SIZE.height);
    graphics.destroy();
  }

  const player = scene.physics.add.sprite(level.playerStart.x, level.playerStart.y, textureKey);
  player.setCollideWorldBounds(true);
  player.setBounce(0.03);
  player.body?.setSize(PLAYER_SIZE.width - 4, PLAYER_SIZE.height - 2);
  player.body?.setOffset(2, 2);

  scene.physics.add.collider(player, platforms);
  for (const platform of movingPlatforms) {
    scene.physics.add.collider(player, platform.block);
  }

  scene.cameras.main.startFollow(player, true, 0.1, 0.1, -160, 40);
  scene.cameras.main.setDeadzone(180, 140);

  return player;
}

export function createCoins(
  scene: Phaser.Scene,
  level: LevelConfig,
  player: Phaser.Physics.Arcade.Sprite,
  onCollect: () => void
) {
  ensureCoinTexture(scene);

  const coins = scene.physics.add.group({ allowGravity: false, immovable: true });

  for (const coinPosition of level.coins) {
    const coin = scene.physics.add.sprite(coinPosition.x, coinPosition.y, 'coin-retro');
    scene.physics.add.existing(coin);
    const coinBody = coin.body as Phaser.Physics.Arcade.Body;
    coinBody.setAllowGravity(false);
    coinBody.setCircle(9, 1, 1);
    coins.add(coin);

    scene.tweens.add({
      targets: coin,
      y: coin.y - 6,
      duration: 760,
      yoyo: true,
      repeat: -1,
      ease: 'Sine.InOut',
    });
  }

  scene.physics.add.overlap(player, coins, (_, coin) => {
    coin.destroy();
    onCollect();
  });

  return coins;
}

export function createCheckpoints(
  scene: Phaser.Scene,
  level: LevelConfig,
  player: Phaser.Physics.Arcade.Sprite,
  onActivate: (label: string) => void
) {
  const checkpoints: CheckpointInstance[] = [];

  for (const checkpoint of level.checkpoints) {
    const pole = scene.add.rectangle(checkpoint.x, checkpoint.y - 38, 8, 102, 0xf8fafc);
    pole.setStrokeStyle(2, 0xf8d16e, 0.7);

    const pennant = scene.add.triangle(
      checkpoint.x + 18,
      checkpoint.y - 78,
      0,
      0,
      0,
      28,
      38,
      14,
      0xe55433,
      0.96
    );
    pennant.setOrigin(0.08, 0.1);
    pennant.setStrokeStyle(2, 0xfff7dd, 0.85);

    const halo = scene.add.circle(checkpoint.x + 4, checkpoint.y - 104, 24, 0xffcf5b, 0.16);
    halo.setStrokeStyle(3, 0xffefaa, 0.52);

    const beacon = scene.add.circle(checkpoint.x + 4, checkpoint.y - 104, 10, 0xffcf5b, 0.96);
    beacon.setStrokeStyle(2, 0xfff7dd);

    const badge = scene.add.text(checkpoint.x + 16, checkpoint.y - 114, 'SAVE', {
      color: '#fff7dd',
      fontFamily: 'monospace',
      fontSize: '11px',
      fontStyle: 'bold',
      letterSpacing: 1,
    });
    badge.setShadow(0, 0, '#ff9b3d', 10, false, true);

    scene.tweens.add({
      targets: halo,
      alpha: { from: 0.1, to: 0.3 },
      scaleX: { from: 0.92, to: 1.18 },
      scaleY: { from: 0.92, to: 1.18 },
      duration: 950,
      yoyo: true,
      repeat: -1,
      ease: 'Sine.InOut',
    });

    scene.tweens.add({
      targets: [beacon, badge],
      y: '-=4',
      duration: 1200,
      yoyo: true,
      repeat: -1,
      ease: 'Sine.InOut',
    });

    const zone = scene.add.zone(checkpoint.x + 14, checkpoint.y - 40, 60, 92);
    scene.physics.add.existing(zone, true);
    checkpoints.push({ definition: checkpoint, zone, beacon, halo, active: false });

    scene.physics.add.overlap(player, zone, () => {
      onActivate(checkpoint.label);
    });
  }

  return checkpoints;
}

export function createHazards(
  scene: Phaser.Scene,
  level: LevelConfig,
  player: Phaser.Physics.Arcade.Sprite,
  onHit: () => void
) {
  ensureHazardTextures(scene);

  for (const hazard of level.hazards) {
    const key = hazard.kind === 'lava' ? 'hazard-lava' : 'hazard-spikes';
    const block = scene.add.tileSprite(
      hazard.x + hazard.width / 2,
      hazard.y + hazard.height / 2,
      hazard.width,
      hazard.height,
      key
    );
    scene.physics.add.existing(block, true);
    scene.physics.add.overlap(player, block, onHit);
  }
}

export function createEnemies(
  scene: Phaser.Scene,
  level: LevelConfig,
  player: Phaser.Physics.Arcade.Sprite,
  platforms: Phaser.Physics.Arcade.StaticGroup,
  movingPlatforms: MovingPlatformInstance[],
  collisionHandler: Phaser.Types.Physics.Arcade.ArcadePhysicsCallback,
  collisionContext: Phaser.Scene
) {
  const textureKey = 'patrol-critter';
  if (!scene.textures.exists(textureKey)) {
    const graphics = scene.make.graphics({ x: 0, y: 0, add: false });
    graphics.fillStyle(0x6d3d1f);
    graphics.fillRect(4, 4, 26, 12);
    graphics.fillStyle(0x895029);
    graphics.fillRect(6, 6, 22, 8);
    graphics.fillStyle(0xf7e2bc);
    graphics.fillRect(8, 16, 18, 8);
    graphics.fillStyle(0x2b1b13);
    graphics.fillRect(12, 17, 2, 4);
    graphics.fillRect(20, 17, 2, 4);
    graphics.fillStyle(0xb87137);
    graphics.fillRect(6, 23, 7, 5);
    graphics.fillRect(21, 23, 7, 5);
    graphics.generateTexture(textureKey, 34, 30);
    graphics.destroy();
  }

  const enemies: EnemyInstance[] = [];

  for (const definition of level.enemies) {
    const enemy = scene.physics.add.sprite(definition.x, definition.y, textureKey);
    enemy.setBounce(0);
    enemy.body?.setSize(28, 28);
    enemy.setCollideWorldBounds(false);

    scene.physics.add.collider(enemy, platforms);
    for (const platform of movingPlatforms) {
      scene.physics.add.collider(enemy, platform.block);
    }

    scene.physics.add.collider(player, enemy, collisionHandler, undefined, collisionContext);
    enemies.push({ sprite: enemy, config: definition, direction: 1 });
  }

  return enemies;
}

function drawCloud(scene: Phaser.Scene, x: number, y: number, alpha: number) {
  scene.add.rectangle(x, y + 8, 84, 20, 0xffffff, alpha);
  scene.add.rectangle(x - 28, y, 34, 22, 0xffffff, alpha);
  scene.add.rectangle(x + 6, y - 8, 42, 24, 0xffffff, alpha);
  scene.add.rectangle(x + 32, y + 2, 30, 18, 0xffffff, alpha);
  scene.add.rectangle(x, y + 18, 90, 6, 0xd7eef9, alpha * 0.9);
}

function drawHill(scene: Phaser.Scene, centerX: number, floorY: number, color: number, width: number, height: number) {
  const hill = scene.add.ellipse(centerX, floorY, width, height, color);
  hill.setOrigin(0.5, 1);

  const eyeLeft = scene.add.circle(centerX - 28, floorY - height * 0.48, 5, 0x152534, 0.56);
  const eyeRight = scene.add.circle(centerX + 10, floorY - height * 0.48, 5, 0x152534, 0.56);
  eyeLeft.setOrigin(0.5, 1);
  eyeRight.setOrigin(0.5, 1);
}

function drawBush(scene: Phaser.Scene, x: number, y: number, color: number) {
  scene.add.circle(x, y, 24, color);
  scene.add.circle(x + 26, y - 6, 20, color);
  scene.add.circle(x + 52, y, 24, color);
}

function drawCastleSilhouette(scene: Phaser.Scene, x: number, y: number, color: number) {
  scene.add.rectangle(x, y, 124, 108, color).setOrigin(0.5, 1);
  scene.add.rectangle(x - 42, y - 64, 34, 84, color).setOrigin(0.5, 1);
  scene.add.rectangle(x + 42, y - 52, 34, 72, color).setOrigin(0.5, 1);

  for (const offset of [-48, -22, 4, 30, 56]) {
    scene.add.rectangle(x + offset, y - 108, 14, 18, color).setOrigin(0.5, 1);
  }
}

function ensurePlatformTextures(scene: Phaser.Scene) {
  ensureBrickTexture(scene, 'platform-moss', {
    base: 0xb56a27,
    mortar: 0x75401e,
    highlight: 0xe6b15d,
    cap: 0x6fb34d,
    capShadow: 0x4d8f31,
  });
  ensureBrickTexture(scene, 'platform-ember', {
    base: 0x7b3b23,
    mortar: 0x4a2517,
    highlight: 0xb46d3f,
    cap: 0xc27a3c,
    capShadow: 0x8d562b,
  });

  if (!scene.textures.exists('platform-cloud')) {
    const graphics = scene.make.graphics({ x: 0, y: 0, add: false });
    graphics.fillStyle(0xb7ddff);
    graphics.fillRect(0, 20, 96, 12);
    graphics.fillStyle(0x8ec5ff);
    graphics.fillRect(0, 24, 96, 8);

    graphics.fillStyle(0xf6fbff);
    graphics.fillCircle(12, 18, 12);
    graphics.fillCircle(30, 12, 14);
    graphics.fillCircle(48, 17, 13);
    graphics.fillCircle(66, 11, 14);
    graphics.fillCircle(84, 18, 12);
    graphics.fillRect(8, 14, 80, 14);
    graphics.fillRect(2, 20, 92, 8);

    graphics.fillStyle(0xffffff, 0.9);
    graphics.fillRect(10, 10, 10, 3);
    graphics.fillRect(39, 8, 12, 3);
    graphics.fillRect(70, 9, 10, 3);
    graphics.fillRect(24, 18, 48, 2);

    graphics.fillStyle(0xdbefff);
    graphics.fillRect(6, 22, 84, 5);
    graphics.fillRect(0, 28, 96, 2);

    graphics.fillStyle(0xfff5bf, 0.65);
    graphics.fillRect(16, 24, 8, 2);
    graphics.fillRect(46, 23, 10, 2);
    graphics.fillRect(74, 24, 8, 2);

    graphics.lineStyle(2, 0xffffff, 0.75);
    graphics.strokeLineShape(new Phaser.Geom.Line(8, 27, 88, 27));
    graphics.lineStyle(2, 0x7fb4ef, 0.72);
    graphics.strokeLineShape(new Phaser.Geom.Line(0, 31, 96, 31));

    graphics.generateTexture('platform-cloud', 96, 32);
    graphics.destroy();
  }

  if (!scene.textures.exists('ground-tile')) {
    const graphics = scene.make.graphics({ x: 0, y: 0, add: false });
    graphics.fillStyle(0x7a4a20);
    graphics.fillRect(0, 0, 96, 64);
    graphics.fillStyle(0x8a5728);
    graphics.fillRect(0, 18, 96, 14);
    graphics.fillRect(0, 44, 96, 10);
    graphics.fillStyle(0x70ba49);
    graphics.fillRect(0, 0, 96, 10);
    graphics.fillStyle(0x4e9832);
    graphics.fillRect(0, 10, 96, 6);
    graphics.fillStyle(0xc98a4d);
    graphics.fillRect(8, 24, 12, 8);
    graphics.fillRect(38, 36, 16, 8);
    graphics.fillRect(72, 22, 12, 8);
    graphics.generateTexture('ground-tile', 96, 64);
    graphics.destroy();
  }
}

function ensureBrickTexture(
  scene: Phaser.Scene,
  key: string,
  colors: { base: number; mortar: number; highlight: number; cap: number; capShadow: number }
) {
  if (scene.textures.exists(key)) {
    return;
  }

  const graphics = scene.make.graphics({ x: 0, y: 0, add: false });
  graphics.fillStyle(colors.base);
  graphics.fillRect(0, 0, 64, 32);
  graphics.fillStyle(colors.cap);
  graphics.fillRect(0, 0, 64, 7);
  graphics.fillStyle(colors.capShadow);
  graphics.fillRect(0, 6, 64, 3);
  graphics.lineStyle(2, colors.mortar, 0.95);
  graphics.strokeRect(1, 1, 62, 30);
  graphics.beginPath();
  graphics.moveTo(0, 17);
  graphics.lineTo(64, 17);
  graphics.moveTo(16, 7);
  graphics.lineTo(16, 17);
  graphics.moveTo(48, 7);
  graphics.lineTo(48, 17);
  graphics.moveTo(32, 17);
  graphics.lineTo(32, 32);
  graphics.strokePath();
  graphics.fillStyle(colors.highlight, 0.4);
  graphics.fillRect(4, 10, 10, 4);
  graphics.fillRect(35, 20, 14, 4);
  graphics.generateTexture(key, 64, 32);
  graphics.destroy();
}

function ensureCoinTexture(scene: Phaser.Scene) {
  if (scene.textures.exists('coin-retro')) {
    return;
  }

  const graphics = scene.make.graphics({ x: 0, y: 0, add: false });
  graphics.fillStyle(0xa05b00);
  graphics.fillCircle(10, 10, 9);
  graphics.fillStyle(0xffd84f);
  graphics.fillCircle(10, 10, 7);
  graphics.fillStyle(0xfff2a8);
  graphics.fillRect(8, 4, 4, 12);
  graphics.fillRect(6, 6, 2, 4);
  graphics.generateTexture('coin-retro', 20, 20);
  graphics.destroy();
}

function ensureHazardTextures(scene: Phaser.Scene) {
  if (!scene.textures.exists('hazard-lava')) {
    const graphics = scene.make.graphics({ x: 0, y: 0, add: false });
    graphics.fillStyle(0xa62b18);
    graphics.fillRect(0, 0, 64, 16);
    graphics.fillStyle(0xe95533);
    graphics.fillRect(0, 2, 64, 14);
    graphics.fillStyle(0xffd24b);
    graphics.fillCircle(8, 6, 3);
    graphics.fillCircle(24, 10, 2);
    graphics.fillCircle(40, 5, 3);
    graphics.fillCircle(56, 9, 2);
    graphics.generateTexture('hazard-lava', 64, 16);
    graphics.destroy();
  }

  if (!scene.textures.exists('hazard-spikes')) {
    const graphics = scene.make.graphics({ x: 0, y: 0, add: false });
    graphics.fillStyle(0x6f0f15);
    graphics.fillRect(0, 10, 64, 6);
    graphics.fillStyle(0xff4d57);
    for (let index = 0; index < 8; index += 1) {
      const x = index * 8;
      graphics.fillTriangle(x, 10, x + 4, 0, x + 8, 10);
    }
    graphics.fillStyle(0xffb3b8, 0.92);
    for (let index = 0; index < 8; index += 1) {
      const x = index * 8;
      graphics.fillTriangle(x + 2, 10, x + 4, 3, x + 6, 10);
    }
    graphics.generateTexture('hazard-spikes', 64, 16);
    graphics.destroy();
  }
}

function getPlatformTextureKey(theme: PlatformTheme | undefined) {
  if (theme === 'ember') {
    return 'platform-ember';
  }
  if (theme === 'cloud') {
    return 'platform-cloud';
  }
  return 'platform-moss';
}
