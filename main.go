package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jakecoffman/cp/v2"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

// 웨이브당 적 스펙 향상 비율 (예: 매 웨이브마다 5% 증가)
const enemyScalingFactor = 0.05

// ----------------------------------------------------------------------
// 프리셋 구조체 정의
// ----------------------------------------------------------------------
type TowerPreset struct {
	Range    float32 // 기본 타워 공격 범위 (픽셀)
	FireRate float32 // 기본 초당 발사 횟수
	Attack   int     // 기본 타워 공격력 (총알 데미지)
	Health   int     // 기본 타워 체력
	Cost     int     // 타워 구매 비용 (경험치)
}

type EnemyPreset struct {
	Health int     // 적 체력
	Speed  float32 // 적 이동 속도 (픽셀/초)
	Damage int     // 적이 타워에 입힐 데미지
}

// 타워 프리셋 (3종류)
var towerPresets = []TowerPreset{
	{Range: 150, FireRate: 1.0, Attack: 50, Health: 100, Cost: 20},
	{Range: 200, FireRate: 0.8, Attack: 40, Health: 150, Cost: 30},
	{Range: 100, FireRate: 2.0, Attack: 70, Health: 80, Cost: 25},
}

// 적 프리셋 (4종류; 이동속도는 낮추고 데미지 추가)
var enemyPresets = []EnemyPreset{
	{Health: 100, Speed: 25, Damage: 20},
	{Health: 150, Speed: 20, Damage: 30},
	{Health: 80, Speed: 35, Damage: 15},
	{Health: 200, Speed: 15, Damage: 40},
}

// ----------------------------------------------------------------------
// 게임 오브젝트 구조체 정의
// ----------------------------------------------------------------------
// Enemy 구조체에 maxHealth 필드 추가
type Enemy struct {
	pos        rl.Vector2
	health     int
	maxHealth  int     // 최대 체력 (생성 시 설정)
	Speed      float32 // 적 이동 속도 (프리셋 값에 scaling 적용)
	damage     int     // 적이 타워에 입힐 데미지 (scaling 적용)
	body       *cp.Body
	expCounted bool // 이미 경험치 증가 처리 여부
}

// Tower는 기본 체력(baseHealth)와 현재 체력(currentHealth)을 구분합니다.
type Tower struct {
	pos           rl.Vector2
	Range         float32   // 기본 공격 사거리
	fireRate      float32   // 기본 발사 속도
	attack        int       // 기본 공격력 (총알 데미지)
	baseHealth    int       // 기본 체력
	currentHealth int       // 현재 체력
	lastShot      time.Time // 마지막 발사 시간
}

type Bullet struct {
	pos      rl.Vector2
	velocity rl.Vector2
	body     *cp.Body
	damage   int // 총알 데미지
}

// ----------------------------------------------------------------------
// 글로벌 변수
// ----------------------------------------------------------------------
var (
	enemies   []Enemy
	towers    []Tower
	bullets   []Bullet
	playerExp int = 100 // 테스트를 위해 100xp로 시작
)

// 웨이브 관련 변수 (웨이브당 몬스터 수를 3배로 늘리고 웨이브 수도 늘림)
var waveEnemyCounts = []int{24, 30, 36, 42, 48, 54, 60, 66, 72, 78}
var currentWave int = 0         // 현재 웨이브 인덱스 (0부터 시작)
var spawnedEnemiesCount int = 0 // 이번 웨이브에서 스폰된 적 수
var spawnInterval float32 = 0.1 // 스폰 간격: 0.1초 (100ms)
var spawnTimer float32 = 0      // 스폰 타이머 누적
var waveWaitTime float32 = 3.0  // 웨이브 종료 후 대기 시간 (초)
var waveWaitTimer float32 = 0   // 웨이브 대기 타이머

// 적 이동 대상: 기지(화면 중앙)
var basePos = rl.Vector2{X: screenWidth / 2, Y: screenHeight / 2}

// 타워 구매를 위한 선택된 프리셋 (초기값은 0번 프리셋)
var selectedTowerPreset int = 0

// ----------------------------------------------------------------------
// 전역 스탯 업그레이드 변수 (각각 1레벨 당 1% 증가)
// ----------------------------------------------------------------------
var hpUpgradeLevel int = 0          // 타워 체력 업그레이드 레벨
var damageUpgradeLevel int = 0      // 공격력 업그레이드 레벨
var attackSpeedUpgradeLevel int = 0 // 공격 속도 업그레이드 레벨
var rangeUpgradeLevel int = 0       // 사거리 업그레이드 레벨

// 업그레이드 비용 및 증분 (각 업그레이드 당 초기 비용 50xp, 레벨당 10xp 증가)
var hpUpgradeCost int = 50
var damageUpgradeCost int = 50
var attackSpeedUpgradeCost int = 50
var rangeUpgradeCost int = 50

const upgradeCostIncrement int = 10

// ----------------------------------------------------------------------
// 유틸리티: 두 점 사이의 거리를 계산하는 함수
// ----------------------------------------------------------------------
func distance(a, b rl.Vector2) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// ----------------------------------------------------------------------
// createEnemy: 지정된 좌표에 적 생성 (enemyPreset 데이터를 복사)
// 웨이브 번호(currentWave)에 따라 적 스펙을 일정 비율로 향상시킵니다.
// ----------------------------------------------------------------------
func createEnemy(x, y float32, preset EnemyPreset) {
	// 웨이브가 증가할 때마다 스펙을 (1 + enemyScalingFactor*currentWave) 배로 향상
	scaledHealth := int(float32(preset.Health) * (1 + enemyScalingFactor*float32(currentWave)))
	scaledSpeed := preset.Speed * (1 + enemyScalingFactor*float32(currentWave))
	scaledDamage := int(float32(preset.Damage) * (1 + enemyScalingFactor*float32(currentWave)))

	body := cp.NewBody(1.0, cp.MomentForCircle(1.0, 0, 10, cp.Vector{X: 0, Y: 0}))
	body.SetPosition(cp.Vector{X: float64(x), Y: float64(y)})
	enemy := Enemy{
		pos:        rl.Vector2{X: x, Y: y},
		health:     scaledHealth,
		maxHealth:  scaledHealth, // 최대 체력 저장
		Speed:      scaledSpeed,
		damage:     scaledDamage,
		body:       body,
		expCounted: false,
	}
	enemies = append(enemies, enemy)
}

// ----------------------------------------------------------------------
// 메인 함수
// ----------------------------------------------------------------------
func main() {
	rl.InitWindow(screenWidth, screenHeight, "Tower Defense Game - Tower Placement Minimum Spacing")
	rl.SetTargetFPS(60)
	rand.Seed(time.Now().UnixNano())

	space := cp.NewSpace()
	space.SetGravity(cp.Vector{X: 0, Y: 0})

	// 메인 게임 루프
	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()
		space.Step(float64(dt))

		// ----- 웨이브 스폰 로직 (생략: 이전과 동일) -----
		if currentWave < len(waveEnemyCounts) {
			if spawnedEnemiesCount < waveEnemyCounts[currentWave] {
				spawnTimer += dt
				if spawnTimer >= spawnInterval {
					angle := 2 * math.Pi * float64(spawnedEnemiesCount) / float64(waveEnemyCounts[currentWave])
					spawnRadius := float32(300)
					x := basePos.X + spawnRadius*float32(math.Cos(angle))
					y := basePos.Y + spawnRadius*float32(math.Sin(angle))
					presetEnemy := enemyPresets[rand.Intn(len(enemyPresets))]
					createEnemy(x, y, presetEnemy)
					spawnedEnemiesCount++
					spawnTimer = 0
				}
			} else {
				if len(enemies) == 0 {
					waveWaitTimer += dt
					if waveWaitTimer >= waveWaitTime {
						currentWave++
						if currentWave >= len(waveEnemyCounts) {
							currentWave = 0
						}
						spawnedEnemiesCount = 0
						spawnTimer = 0
						waveWaitTimer = 0
					}
				}
			}
		}

		// ----- 적 이동 로직 (이전과 동일) -----
		for i := range enemies {
			enemy := &enemies[i]
			var targetPos rl.Vector2
			if len(towers) > 0 {
				closestTower := towers[0]
				minDist := distance(enemy.pos, towers[0].pos)
				for j := 1; j < len(towers); j++ {
					d := distance(enemy.pos, towers[j].pos)
					if d < minDist {
						minDist = d
						closestTower = towers[j]
					}
				}
				targetPos = closestTower.pos
			} else {
				targetPos = basePos
			}
			dx := targetPos.X - enemy.pos.X
			dy := targetPos.Y - enemy.pos.Y
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			if dist > 0 {
				enemy.pos.X += dx / dist * enemy.Speed * dt
				enemy.pos.Y += dy / dist * enemy.Speed * dt
				enemy.body.SetPosition(cp.Vector{X: float64(enemy.pos.X), Y: float64(enemy.pos.Y)})
			}
		}

		// ----- 적과 타워 간 충돌 체크 (이전과 동일) -----
		collisionThreshold := float32(30)
		var remainingEnemies []Enemy
		for i := range enemies {
			enemy := &enemies[i]
			collided := false
			for j := 0; j < len(towers); j++ {
				if distance(enemy.pos, towers[j].pos) < collisionThreshold {
					towers[j].currentHealth -= enemy.damage
					collided = true
					break
				}
			}
			if !collided {
				remainingEnemies = append(remainingEnemies, *enemy)
			}
		}
		enemies = remainingEnemies

		// 체력이 0 이하인 타워 제거
		var remainingTowers []Tower
		for i := range towers {
			if towers[i].currentHealth > 0 {
				remainingTowers = append(remainingTowers, towers[i])
			}
		}
		towers = remainingTowers

		// ----- 타워 발사 로직 (업그레이드 적용) -----
		for ti := range towers {
			tower := &towers[ti]
			effectiveRange := tower.Range * (1 + float32(rangeUpgradeLevel)*0.01)
			effectiveFireRate := tower.fireRate * (1 + float32(attackSpeedUpgradeLevel)*0.01)
			effectiveAttack := int(float32(tower.attack) * (1 + float32(damageUpgradeLevel)*0.01))
			if time.Since(tower.lastShot).Seconds() >= float64(1.0/effectiveFireRate) {
				for ei := range enemies {
					enemy := &enemies[ei]
					dx := enemy.pos.X - tower.pos.X
					dy := enemy.pos.Y - tower.pos.Y
					dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
					if dist <= effectiveRange {
						dir := rl.Vector2{X: dx / dist, Y: dy / dist}
						bulletSpeed := float32(300)
						bulletBody := cp.NewBody(1.0, cp.MomentForCircle(1.0, 0, 5, cp.Vector{X: 0, Y: 0}))
						bulletBody.SetPosition(cp.Vector{X: float64(tower.pos.X), Y: float64(tower.pos.Y)})
						bullet := Bullet{
							pos:      tower.pos,
							velocity: rl.Vector2{X: dir.X * bulletSpeed, Y: dir.Y * bulletSpeed},
							body:     bulletBody,
							damage:   effectiveAttack,
						}
						bullets = append(bullets, bullet)
						tower.lastShot = time.Now()
						break
					}
				}
			}
		}

		// ----- 총알 이동 및 적과 충돌 체크 (이전과 동일) -----
		for bi := 0; bi < len(bullets); bi++ {
			bullet := &bullets[bi]
			bullet.pos.X += bullet.velocity.X * dt
			bullet.pos.Y += bullet.velocity.Y * dt
			bullet.body.SetPosition(cp.Vector{X: float64(bullet.pos.X), Y: float64(bullet.pos.Y)})
			for ei := 0; ei < len(enemies); ei++ {
				enemy := &enemies[ei]
				dx := enemy.pos.X - bullet.pos.X
				dy := enemy.pos.Y - bullet.pos.Y
				if float32(math.Sqrt(float64(dx*dx+dy*dy))) < 10 {
					enemy.health -= bullet.damage
					if enemy.health <= 0 && !enemy.expCounted {
						playerExp += 10
						enemy.expCounted = true
					}
					bullet.pos = rl.Vector2{X: -100, Y: -100}
				}
			}
		}
		var activeBullets []Bullet
		for _, bullet := range bullets {
			if bullet.pos.X >= 0 && bullet.pos.X <= screenWidth && bullet.pos.Y >= 0 && bullet.pos.Y <= screenHeight {
				activeBullets = append(activeBullets, bullet)
			}
		}
		bullets = activeBullets

		var activeEnemies []Enemy
		for _, enemy := range enemies {
			if enemy.health > 0 {
				activeEnemies = append(activeEnemies, enemy)
			}
		}
		enemies = activeEnemies

		// ----- 업그레이드 구매 (경험치 사용) -----
		if rl.IsKeyPressed(rl.KeyF1) { // HP 업그레이드
			if playerExp >= hpUpgradeCost {
				playerExp -= hpUpgradeCost
				hpUpgradeLevel++
				hpUpgradeCost += upgradeCostIncrement
				// 모든 타워의 체력 1% 증가
				for i := range towers {
					towers[i].baseHealth = int(float32(towers[i].baseHealth) * 1.01)
					towers[i].currentHealth = int(float32(towers[i].currentHealth) * 1.01)
				}
			}
		}
		if rl.IsKeyPressed(rl.KeyF2) { // Damage 업그레이드
			if playerExp >= damageUpgradeCost {
				playerExp -= damageUpgradeCost
				damageUpgradeLevel++
				damageUpgradeCost += upgradeCostIncrement
			}
		}
		if rl.IsKeyPressed(rl.KeyF3) { // Attack Speed 업그레이드
			if playerExp >= attackSpeedUpgradeCost {
				playerExp -= attackSpeedUpgradeCost
				attackSpeedUpgradeLevel++
				attackSpeedUpgradeCost += upgradeCostIncrement
			}
		}
		if rl.IsKeyPressed(rl.KeyF4) { // Range 업그레이드
			if playerExp >= rangeUpgradeCost {
				playerExp -= rangeUpgradeCost
				rangeUpgradeLevel++
				rangeUpgradeCost += upgradeCostIncrement
			}
		}

		// ----- 타워 구매 및 설치 (경험치 사용) -----
		// 키 1,2,3로 타워 종류 선택 후 좌클릭하여 설치
		if rl.IsKeyPressed(rl.KeyOne) {
			selectedTowerPreset = 0
		} else if rl.IsKeyPressed(rl.KeyTwo) {
			selectedTowerPreset = 1
		} else if rl.IsKeyPressed(rl.KeyThree) {
			selectedTowerPreset = 2
		}
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			mousePos := rl.GetMousePosition()
			// 최소 설치 영역 검사 (예: 50픽셀)
			minSpacing := float32(50)
			validPlacement := true
			for _, t := range towers {
				if distance(mousePos, t.pos) < minSpacing {
					validPlacement = false
					break
				}
			}
			if validPlacement {
				cost := towerPresets[selectedTowerPreset].Cost
				if playerExp >= cost {
					playerExp -= cost
					preset := towerPresets[selectedTowerPreset]
					newTower := Tower{
						pos:           mousePos,
						Range:         preset.Range,
						fireRate:      preset.FireRate,
						attack:        preset.Attack,
						baseHealth:    preset.Health,
						currentHealth: preset.Health,
						lastShot:      time.Now(),
					}
					towers = append(towers, newTower)
				}
			}
		}

		// ----- 렌더링 -----
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		// 기지(베이스) 그리기 (파란 원)
		rl.DrawCircle(int32(basePos.X), int32(basePos.Y), 15, rl.Blue)

		// 타워 그리기 (사거리 영역은 윤곽선으로 표시)
		for _, tower := range towers {
			effectiveRange := tower.Range * (1 + float32(rangeUpgradeLevel)*0.01)
			rl.DrawCircleLines(int32(tower.pos.X), int32(tower.pos.Y), effectiveRange, rl.LightGray)
			rl.DrawCircle(int32(tower.pos.X), int32(tower.pos.Y), 10, rl.DarkGray)

			// --- 타워 체력바 그리기 ---
			// 체력바 크기 (폭 30, 높이 5)
			healthBarWidth := int32(30)
			healthBarHeight := int32(5)
			// 체력 비율 계산
			healthRatio := float32(tower.currentHealth) / float32(tower.baseHealth)
			// 체력바 위치: 타워 중심 위쪽 (예: Y offset -20)
			barX := int32(tower.pos.X) - int32(healthBarWidth/2)
			barY := int32(tower.pos.Y) - 20
			// 배경(회색)과 체력(주황색) 그리기
			rl.DrawRectangle(barX, barY, healthBarWidth, healthBarHeight, rl.NewColor(128, 128, 128, 255))
			rl.DrawRectangle(barX, barY, int32(float32(healthBarWidth)*healthRatio), healthBarHeight, rl.Orange)
		}
		// 적 그리기 (빨간 사각형) 및 체력바 추가
		for _, enemy := range enemies {
			rl.DrawRectangle(int32(enemy.pos.X)-10, int32(enemy.pos.Y)-10, 20, 20, rl.Red)
			// --- 적 체력바 그리기 ---
			healthBarWidth := int32(20)
			healthBarHeight := int32(4)
			healthRatio := float32(enemy.health) / float32(enemy.maxHealth)
			barX := int32(enemy.pos.X) - int32(healthBarWidth/2)
			barY := int32(enemy.pos.Y) - 15
			rl.DrawRectangle(barX, barY, healthBarWidth, healthBarHeight, rl.NewColor(128, 128, 128, 255))
			rl.DrawRectangle(barX, barY, int32(float32(healthBarWidth)*healthRatio), healthBarHeight, rl.Orange)
		}
		// 총알 그리기 (작은 검은 원)
		for _, bullet := range bullets {
			rl.DrawCircle(int32(bullet.pos.X), int32(bullet.pos.Y), 5, rl.Black)
		}

		// UI 출력
		rl.DrawText(fmt.Sprintf("Exp: %d", playerExp), 10, 10, 20, rl.Black)
		rl.DrawText(fmt.Sprintf("Wave: %d", currentWave+1), 10, 40, 20, rl.Black)
		rl.DrawText(fmt.Sprintf("Selected Tower: %d (Cost: %d)", selectedTowerPreset+1, towerPresets[selectedTowerPreset].Cost), 10, 70, 20, rl.Black)
		rl.DrawText("Press 1,2,3 to select tower type, Left click to place tower", 10, 100, 20, rl.Black)
		rl.DrawText(fmt.Sprintf("HP Upgrade (F1): Level %d, Cost: %d", hpUpgradeLevel, hpUpgradeCost), 10, 130, 20, rl.Black)
		rl.DrawText(fmt.Sprintf("Damage Upgrade (F2): Level %d, Cost: %d", damageUpgradeLevel, damageUpgradeCost), 10, 160, 20, rl.Black)
		rl.DrawText(fmt.Sprintf("Attack Speed Upgrade (F3): Level %d, Cost: %d", attackSpeedUpgradeLevel, attackSpeedUpgradeCost), 10, 190, 20, rl.Black)
		rl.DrawText(fmt.Sprintf("Range Upgrade (F4): Level %d, Cost: %d", rangeUpgradeLevel, rangeUpgradeCost), 10, 220, 20, rl.Black)

		rl.EndDrawing()
	}

	rl.CloseWindow()
}
