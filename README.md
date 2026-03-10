# game-engine-golang

Проект теперь разделён на движок, стартовый шаблон и полноценный пример игры.

## Что теперь является движком

Пакет [`engine`](./engine) содержит:
- `Config` - система конфигурации с загрузкой из JSON (окно, графика)
- `Run(cfg, game)` - главный цикл движка
- `Context` - объекты, доступные игре (`Window`, `Camera`, `Renderer`, `Projection`, `Time`, `DeltaTime`)
- `Game` - интерфейс игры:
  - `Init(ctx)`
  - `Update(ctx)`
  - `Render(ctx)`
- `Renderer` - обёртка над OpenGL-шейдером и рисованием мешей
- `Mesh` - VAO/VBO + количество вершин
- `World` (ECS-lite) - сущности и компоненты (`CreateEntity`, `engine.AddComponent`, `engine.GetComponent`, `engine.Each1`, `engine.Each2`)

## Конфигурация

Настройки движка хранятся в `config.json`:

```json
{
  "window": {
    "width": 1920,
    "height": 1080,
    "title": "Go Game Engine",
    "fullscreen": true,
    "vsync": true
  },
  "graphics": {
    "fov": 60,
    "near": 0.1,
    "far": 100
  }
}
```

Файл создаётся автоматически при первом запуске с дефолтными значениями.
Используй `config.example.json` как шаблон.

## Как пользоваться движком

1. Создай структуру игры и реализуй `engine.Game`.
2. Загрузи конфигурацию через `engine.LoadConfig("config.json")`.
3. В `Init` создай меши через `ctx.Renderer.NewMesh(...)` и подготовь данные.
4. В `Update` обрабатывай ввод и игровую логику, используя `ctx.DeltaTime`.
5. В `Render` выставляй цвет/туман/матрицы и вызывай `ctx.Renderer.Draw(...)`.
6. Запусти игру через `engine.Run(cfg, yourGame)`.

Минимальный шаблон уже находится в корневом [`main.go`](./main.go).

## Пример с ECS (подводная сцена)

Полный пример игры вынесен в [`examples/subnautica/main.go`](./examples/subnautica/main.go).

Запуск:

```bash
go run ./examples/subnautica
```

В этом примере:
- растения создаются как ECS-сущности
- `transform` и `plantVisual` хранятся как компоненты
- отрисовка проходит через `engine.Each2(world, ...)`

## Запуск стартового шаблона

```bash
go run .
```

## Управление в подводном примере

- `W A S D` - движение
- `Mouse` - обзор
- `Space` - прыжок/всплытие
- `Left Shift` - ускорение
- `Esc` - выход

## Структура проекта

- `engine/` - ядро движка (окно, цикл, камера, рендер, ECS-lite)
- `objects/` - геометрия и генерация сцены
- `utils/` - математика и случайные значения
- `examples/subnautica/` - полноценный пример игры на движке
- `main.go` - минимальный стартовый шаблон
