# Система светящихся растений

## Обзор

Система светящихся растений добавляет эмиссивные объекты с настраиваемым свечением и пульсацией.

## Компоненты

### GlowRenderer
Специализированный рендерер с эмиссивным шейдером:
- Поддержка нормалей для rim lighting эффекта
- Настраиваемый цвет свечения
- Регулируемая интенсивность
- Опциональная пульсация

### Transform
Компонент позиции и трансформации:
```go
type Transform struct {
    Position mgl32.Vec3
    Yaw      float32
    Scale    float32
}
```

### GlowPlant
Компонент светящегося растения:
```go
type GlowPlant struct {
    Type          PlantType      // PlantKelp, PlantBush, PlantCoral
    GlowColor     mgl32.Vec3     // RGB цвет свечения
    GlowIntensity float32        // Интенсивность (1.0-5.0)
    PulseSpeed    float32        // Скорость пульсации (0 = нет)
}
```

## Использование

### Простое создание
```go
objects.SpawnGlowingPlant(
    world,
    mgl32.Vec3{x, y, z},
    objects.GlowNeonGreen,
    2.5, // интенсивность
)
```

### Расширенное создание
```go
objects.SpawnGlowingPlantAdvanced(
    world,
    position,
    glowColor,
    intensity,
    pulseSpeed,  // 0 = без пульсации
    plantType,   // PlantKelp/PlantBush/PlantCoral
    scale,
    yaw,
)
```

### Случайная генерация
```go
objects.SpawnRandomGlowingPlants(world, 50, 40.0) // 50 растений в радиусе 40
```

## Предустановленные цвета

- `GlowNeonGreen` - неоновый зелёный
- `GlowNeonBlue` - неоновый синий
- `GlowNeonPurple` - неоновый фиолетовый
- `GlowNeonPink` - неоновый розовый
- `GlowNeonOrange` - неоновый оранжевый
- `GlowNeonCyan` - неоновый голубой

## Рендеринг

```go
ctx.GlowRenderer.Use()
ctx.GlowRenderer.SetTime(ctx.Time)
ctx.GlowRenderer.SetViewPos(ctx.Camera.Position)

engine.Each2(world, func(_ engine.Entity, tr *objects.Transform, gp *objects.GlowPlant) {
    mesh := plantMeshes[gp.Type]
    ctx.GlowRenderer.SetMVP(projection.Mul4(view).Mul4(tr.ModelMatrix()))
    ctx.GlowRenderer.SetModel(tr.ModelMatrix())
    ctx.GlowRenderer.SetGlowColor(gp.GlowColor)
    ctx.GlowRenderer.SetGlowIntensity(gp.GlowIntensity)
    ctx.GlowRenderer.SetPulseSpeed(gp.PulseSpeed)
    ctx.GlowRenderer.Draw(mesh)
})
```

## Запуск примера

```bash
go run ./examples/glowing_plants
```

## Особенности

1. **Эмиссивное свечение** - растения светятся независимо от освещения сцены
2. **Rim lighting** - усиление свечения на краях для объёмности
3. **Пульсация** - плавная анимация интенсивности (опционально)
4. **ECS интеграция** - полная совместимость с системой компонентов движка
5. **Производительность** - оптимизированный шейдер без лишних вычислений
