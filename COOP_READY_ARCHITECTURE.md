# Coop-Ready Lite Architecture

Этот проект остаётся singleplayer, но теперь использует заготовку под будущий host/client coop.

## 1) Authoritative state (игровая симуляция)

Код: `examples/subnautica/coop_state.go`, `examples/subnautica/coop_simulation.go`.

Что входит:
- `authoritativeState` (tick/time/water level/players/entities).
- `playerAuthoritativeState` (позиция, режим воды, noclip, скорость по Y, health, oxygen).
- `EntityID` + `entityMeta` (тип сущности, owner, authority, scope репликации).

Принцип:
- gameplay-состояние мутируется только через `simulationCommand` в `runSimulationFrame`.
- `Update` собирает intents и enqueue-команды, но не пишет напрямую в gameplay-state.

## 2) Local-only state

Что входит:
- `localControlState` — edge-состояние ввода (`SpaceWasDown`, последний вертикальный axis).
- `localPresentationState` — чисто визуальные значения (`ProceduralYOffset`, `UnderwaterIdleMix`).

Принцип:
- local-only данные не являются authoritative.
- они не требуются для репликации и могут пересчитываться на клиенте.

## 3) Что потенциально реплицируется позже

Минимальный набор для coop:
- `authoritativeState.Players[*].Position/LookDir/Health/Oxygen/WaterMode`.
- `authoritativeState.Entities` (`EntityID`, `Kind`, `Owner`, `Scope`, `Dynamic`).
- spawn/despawn события (`simulationEventEntitySpawned` и будущие despawn events).
- изменения мира (например, терраформинг через `simulationCommandApplyTerrainBrush`).

Не реплицируется:
- HUD/console/fps overlay.
- camera bob, first person hand sway, debug/editor overlay.
- GL/mesh/shader ресурсы.

## 4) Где подключать будущий transport (host/client)

Точка входа входящих сетевых действий:
- преобразовывать network packets в `simulationCommand` и подавать в очередь перед `runSimulationFrame`.

Точка выхода исходящего состояния:
- сериализовать `authoritativeState` и `simulationEvent` после `runSimulationFrame`.

Переключение режимов:
- `sessionModeSingleplayer` = local simulation + local presentation.
- `sessionModeHost` = authoritative simulation + local presentation + отправка snapshots.
- `sessionModeClient` = применение входящих authoritative snapshots + local presentation/prediction.

## 5) Текущие ограничения (осознанно не реализовано)

- нет сетевого транспорта, lobby/matchmaking/NAT/dedicated server.
- нет клиентского prediction/rollback reconciliation.
- нет снапшотной компрессии/delta protocol.
- нет полноценного despawn/event-history потока для late join.
