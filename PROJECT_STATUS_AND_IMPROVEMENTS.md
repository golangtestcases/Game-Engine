# Subnautica Lite: Текущее Состояние Проекта И Улучшения

## Что Уже Реализовано

- Базовый движок: цикл `Run`, `Context`, камера, рендерер, ECS-lite.
  - `engine/app.go`
  - `engine/ecs.go`
- Полноценный runtime примера `subnautica`: `Init`, `Update`, `Render`.
  - `examples/subnautica/game.go`
  - `examples/subnautica/update.go`
- Архитектура под кооп: authoritative state, команды симуляции, события.
  - `examples/subnautica/coop_state.go`
  - `examples/subnautica/coop_simulation.go`
  - `examples/subnautica/coop_commands.go`
- Cell streaming + LOD по террейну и декору (hysteresis, culling, density).
  - `examples/subnautica/world_streaming.go`
  - `examples/subnautica/world_streaming_update.go`
  - `examples/subnautica/world_streaming_terrain.go`
  - `examples/subnautica/world_streaming_plants.go`
- Редактируемый террейн: кисть, raycast, сохранение/загрузка JSON.
  - `objects/editable_terrain.go`
  - `examples/subnautica/editor_mode*.go`
  - `examples/subnautica/terrain_runtime.go`
- Продвинутый рендер: render graph, shadow pass, ocean system, туман, glow-растения, quality presets.
  - `engine/render_graph.go`
  - `engine/render_passes.go`
  - `examples/subnautica/render.go`
  - `examples/subnautica/graphics_quality.go`
- Существа: FSM-поведение, параметры движения, GLB-модель, рендер в сцене и в тенях.
  - `objects/creature.go`
  - `examples/subnautica/creatures.go`
- Иммерсивный first-person слой: процедурные руки и анимация плавания.
  - `examples/subnautica/first_person_hands.go`
  - `examples/subnautica/first_person_hand_mesh.go`
- Инструменты разработчика: dev-console, FPS overlay, streaming debug HUD, editor HUD.
  - `examples/subnautica/console.go`
  - `examples/subnautica/fps_overlay.go`
  - `examples/subnautica/streaming_debug.go`

## Текущее Состояние Качества

- Проект компилируется по пакетам:
  - `go test ./... -run TestDoesNotExist -count=1` проходит.
- При этом отсутствуют автоматические тесты (`[no test files]`), что повышает риск регрессий при развитии.

## Приоритетные Улучшения

1. Перевести симуляцию на fixed tick (например, 60 Гц) + interpolation рендера.
   - Зачем: стабильная физика/движение и подготовка к netcode.
2. Оптимизировать терраформинг в editor режиме.
   - Сейчас изменение террейна тянет тяжелые операции (пересборка и ресинк декора), что может давать просадки.
   - Цель: сделать обновление по dirty-cell вместо широкого перерасчета.
3. Ввести budgeted jobs для стриминга/LOD.
   - Не строить тяжелые LOD-данные сразу в критическом кадре.
   - Лимитировать объем работы на кадр.
4. Добавить unit-тесты на чистые функции.
   - `selectLODWithHysteresis`
   - `distanceToggleWithHysteresis`
   - `classifyWaterModeByLevel`
   - `viewDistanceToDepth`
   - Логику кисти террейна
5. Добавить явный lifecycle освобождения ресурсов (`Shutdown`/`Dispose`).
   - Для GPU/аудио/оверлеев лучше иметь предсказуемое завершение, а не только cleanup при выходе процесса.
6. Расширить dev-console через реестр команд.
   - Добавить `help`, группировку команд, алиасы, при необходимости автодополнение.

## Почему Проект Уже Сильный

- Архитектура не выглядит как прототип-однодневка: есть разделение gameplay/simulation/presentation.
- Хороший фундамент для масштабирования: streaming, quality presets, ocean pipeline, editor.
- Уже есть задел на кооп, а не просто singleplayer-логика в одном большом update.

## Рекомендованный Порядок Следующих Шагов

1. Fixed tick + interpolation.
2. Dirty-cell терраформинг.
3. Budgeted streaming jobs.
4. Набор базовых unit-тестов.
5. Расширение консоли и инструментов диагностики.
