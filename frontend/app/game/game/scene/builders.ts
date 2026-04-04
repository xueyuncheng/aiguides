import * as Phaser from 'phaser';
import { GAME_HEIGHT, type LevelConfig, type PlatformTheme } from '../levels';
import type { CheckpointInstance, EnemyInstance, MovingPlatformInstance } from './types';

const PLAYER_SIZE = { width: 34, height: 48 };

export function drawBackdrop(scene: Phaser.Scene, level: LevelConfig, levelIndex: number) {
  scene.add.rectangle(level.worldWidth / 2, GAME_HEIGHT / 2, level.worldWidth, GAME_HEIGHT, level.theme.sky);
  scene.add.rectangle(level.worldWidth / 2, 96, level.worldWidth, 140, level.theme.mist, 0.18);

  for (const cloudX of [180, 540, 960, 1320, 1760, 2140, 2580]) {
    scene.add.ellipse(cloudX, 120, 92, 44, 0xffffff, levelIndex === 0 ? 0.85 : 0.22);
    scene.add.ellipse(cloudX + 40, 110, 76, 36, 0xffffff, levelIndex === 0 ? 0.85 : 0.22);
    scene.add.ellipse(cloudX - 36, 110, 70, 34, 0xffffff, levelIndex === 0 ? 0.85 : 0.22);
  }

  for (const hill of [320, 860, 1540, 2050, 2670]) {
    scene.add.triangle(hill, level.floorY, 0, 160, 160, 0, 320, 160, level.theme.hillNear).setOrigin(0.5, 1);
    scene.add.triangle(hill + 190, level.floorY, 0, 120, 120, 0, 240, 120, level.theme.hillFar).setOrigin(0.5, 1);
  }
}

export function createPlatforms(scene: Phaser.Scene, level: LevelConfig) {
  const groundHeight = GAME_HEIGHT - level.floorY;
  const platforms = scene.physics.add.staticGroup();
  const movingPlatforms: MovingPlatformInstance[] = [];

  const floor = scene.add.rectangle(
    level.worldWidth / 2,
    level.floorY + groundHeight / 2,
    level.worldWidth,
    groundHeight,
    level.theme.ground
  );
  scene.physics.add.existing(floor, true);
  platforms.add(floor as unknown as Phaser.Physics.Arcade.Image);

  for (const platform of level.platforms) {
    if (platform.movement) {
      const block = scene.add.rectangle(
        platform.x + platform.width / 2,
        platform.y + platform.height / 2,
        platform.width,
        platform.height,
        getPlatformFill(platform.theme)
      );
      block.setStrokeStyle(2, getPlatformStroke(platform.theme));
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

    const block = scene.add.rectangle(
      platform.x + platform.width / 2,
      platform.y + platform.height / 2,
      platform.width,
      platform.height,
      getPlatformFill(platform.theme)
    );
    block.setStrokeStyle(2, getPlatformStroke(platform.theme));
    scene.physics.add.existing(block, true);
    platforms.add(block as unknown as Phaser.Physics.Arcade.Image);
  }

  return { platforms, movingPlatforms };
}

export function createGoal(scene: Phaser.Scene, level: LevelConfig) {
  const pole = scene.add.rectangle(level.goalX, level.floorY - 104, 12, 204, 0xe2e8f0).setOrigin(0, 0);
  pole.setStrokeStyle(2, 0xf8fafc, 0.8);

  const glow = scene.add.circle(level.goalX + 58, level.floorY - 124, 30, level.theme.accent, 0.18);
  glow.setStrokeStyle(2, 0xfef3c7, 0.35);

  const flag = scene.add.triangle(
    level.goalX + 66,
    level.floorY - 92,
    0,
    0,
    0,
    46,
    68,
    23,
    level.theme.accent,
    0.98
  );
  flag.setOrigin(0.12, 0.1);
  flag.setStrokeStyle(2, 0xfffbeb, 0.9);

  const stripe = scene.add.rectangle(level.goalX + 43, level.floorY - 70, 28, 6, 0xfffbeb, 0.92);
  stripe.setAngle(-16);

  const finial = scene.add.circle(level.goalX + 6, level.floorY - 114, 9, 0xfef3c7);
  finial.setStrokeStyle(2, 0xf59e0b);

  const label = scene.add.text(level.goalX + 20, level.floorY - 146, 'GOAL', {
    color: '#fff7ed',
    fontFamily: 'monospace',
    fontSize: '13px',
    fontStyle: 'bold',
  });
  label.setShadow(0, 0, '#fb923c', 12, false, true);
}

export function createPlayer(
  scene: Phaser.Scene,
  level: LevelConfig,
  platforms: Phaser.Physics.Arcade.StaticGroup,
  movingPlatforms: MovingPlatformInstance[]
) {
  const textureKey = 'player-block';
  if (!scene.textures.exists(textureKey)) {
    const graphics = scene.add.graphics();
    graphics.fillStyle(0xf97316);
    graphics.fillRoundedRect(0, 0, PLAYER_SIZE.width, PLAYER_SIZE.height, 8);
    graphics.fillStyle(0xfef3c7);
    graphics.fillRoundedRect(7, 8, PLAYER_SIZE.width - 14, PLAYER_SIZE.height - 20, 6);
    graphics.fillStyle(0x111827);
    graphics.fillCircle(11, 18, 2);
    graphics.fillCircle(23, 18, 2);
    graphics.generateTexture(textureKey, PLAYER_SIZE.width, PLAYER_SIZE.height);
    graphics.destroy();
  }

  const player = scene.physics.add.sprite(level.playerStart.x, level.playerStart.y, textureKey);
  player.setCollideWorldBounds(true);
  player.setBounce(0.03);
  player.body?.setSize(PLAYER_SIZE.width, PLAYER_SIZE.height);

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
  const coins = scene.physics.add.group({ allowGravity: false, immovable: true });

  for (const coinPosition of level.coins) {
    const coin = scene.add.circle(coinPosition.x, coinPosition.y, 10, 0xfacc15);
    coin.setStrokeStyle(3, 0xfde68a);
    scene.physics.add.existing(coin);
    const coinBody = coin.body as Phaser.Physics.Arcade.Body;
    coinBody.setAllowGravity(false);
    coinBody.setCircle(10);
    coins.add(coin as unknown as Phaser.Physics.Arcade.Sprite);
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
    const pole = scene.add.rectangle(checkpoint.x, checkpoint.y - 42, 6, 104, 0xe2e8f0);
    pole.setStrokeStyle(2, 0xf8fafc, 0.65);

    const halo = scene.add.circle(checkpoint.x, checkpoint.y - 106, 22, 0x38bdf8, 0.16);
    halo.setStrokeStyle(3, 0x7dd3fc, 0.48);

    const beacon = scene.add.circle(checkpoint.x, checkpoint.y - 106, 9, 0x38bdf8, 0.96);
    beacon.setStrokeStyle(2, 0xe0f2fe);

    const badge = scene.add.text(checkpoint.x + 12, checkpoint.y - 114, 'CHECKPOINT', {
      color: '#e0f2fe',
      fontFamily: 'monospace',
      fontSize: '11px',
      fontStyle: 'bold',
      letterSpacing: 1,
    });
    badge.setShadow(0, 0, '#38bdf8', 10, false, true);

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
  for (const hazard of level.hazards) {
    const color = hazard.kind === 'lava' ? 0xfb7185 : 0xf43f5e;
    const block = scene.add.rectangle(
      hazard.x + hazard.width / 2,
      hazard.y + hazard.height / 2,
      hazard.width,
      hazard.height,
      color,
      hazard.kind === 'lava' ? 0.95 : 0.88
    );
    block.setStrokeStyle(2, 0xffedd5);
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
  const textureKey = 'patrol-bot';
  if (!scene.textures.exists(textureKey)) {
    const graphics = scene.add.graphics();
    graphics.fillStyle(0x0f172a);
    graphics.fillRoundedRect(0, 0, 34, 30, 10);
    graphics.fillStyle(0xf97316);
    graphics.fillRoundedRect(4, 4, 26, 22, 8);
    graphics.fillStyle(0xf8fafc);
    graphics.fillCircle(11, 14, 3);
    graphics.fillCircle(23, 14, 3);
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

function getPlatformFill(theme: PlatformTheme | undefined) {
  if (theme === 'ember') {
    return 0x9a3412;
  }
  if (theme === 'cloud') {
    return 0x94a3b8;
  }
  return 0x65a30d;
}

function getPlatformStroke(theme: PlatformTheme | undefined) {
  if (theme === 'ember') {
    return 0xfdba74;
  }
  if (theme === 'cloud') {
    return 0xe2e8f0;
  }
  return 0xfacc15;
}
