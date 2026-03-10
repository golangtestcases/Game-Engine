package engine

import "reflect"

// Entity — уникальный идентификатор сущности в ECS-мире.
// ID только растет; повторного использования удаленных ID нет.
type Entity uint64

// World — минимальное ECS-хранилище.
// Компоненты лежат в map по типу компонента, затем по entity.
type World struct {
	nextID Entity
	alive  map[Entity]struct{}
	store  map[reflect.Type]map[Entity]any
}

// NewWorld создает пустой ECS-мир.
func NewWorld() *World {
	return &World{
		nextID: 1,
		alive:  make(map[Entity]struct{}),
		store:  make(map[reflect.Type]map[Entity]any),
	}
}

// CreateEntity регистрирует новую сущность и возвращает ее ID.
func (w *World) CreateEntity() Entity {
	e := w.nextID
	w.nextID++
	w.alive[e] = struct{}{}
	return e
}

// DestroyEntity удаляет сущность и все её компоненты.
func (w *World) DestroyEntity(e Entity) {
	if !w.isAlive(e) {
		return
	}
	delete(w.alive, e)
	for _, components := range w.store {
		delete(components, e)
	}
}

// EntityCount возвращает число активных сущностей.
func (w *World) EntityCount() int {
	return len(w.alive)
}

// AddComponent добавляет/перезаписывает компонент указанного типа для сущности.
// Компонент копируется, чтобы внешние изменения исходной переменной не влияли на ECS.
func AddComponent[T any](w *World, e Entity, component T) {
	w.requireAlive(e)
	typeID := componentType[T]()
	components := w.ensureStore(typeID)
	componentCopy := component
	components[e] = &componentCopy
}

// RemoveComponent удаляет компонент типа T у сущности.
func RemoveComponent[T any](w *World, e Entity) {
	typeID := componentType[T]()
	components, ok := w.store[typeID]
	if !ok {
		return
	}
	delete(components, e)
}

// HasComponent проверяет наличие компонента типа T.
func HasComponent[T any](w *World, e Entity) bool {
	typeID := componentType[T]()
	components, ok := w.store[typeID]
	if !ok {
		return false
	}
	_, ok = components[e]
	return ok
}

// GetComponent возвращает указатель на компонент типа T.
// Изменения через указатель сразу отражаются в ECS-хранилище.
func GetComponent[T any](w *World, e Entity) (*T, bool) {
	typeID := componentType[T]()
	components, ok := w.store[typeID]
	if !ok {
		return nil, false
	}
	raw, ok := components[e]
	if !ok {
		return nil, false
	}
	comp, ok := raw.(*T)
	return comp, ok
}

// Each1 итерирует все живые сущности с компонентом T.
func Each1[T any](w *World, fn func(e Entity, c *T)) {
	typeID := componentType[T]()
	components, ok := w.store[typeID]
	if !ok {
		return
	}
	for e, raw := range components {
		if !w.isAlive(e) {
			continue
		}
		comp, ok := raw.(*T)
		if !ok {
			continue
		}
		fn(e, comp)
	}
}

// Each2 итерирует живые сущности, у которых одновременно есть A и B.
// Для производительности обходится меньшая из двух component-map.
func Each2[A any, B any](w *World, fn func(e Entity, a *A, b *B)) {
	aType := componentType[A]()
	bType := componentType[B]()
	aStore, okA := w.store[aType]
	bStore, okB := w.store[bType]
	if !okA || !okB {
		return
	}

	if len(aStore) <= len(bStore) {
		for e, rawA := range aStore {
			if !w.isAlive(e) {
				continue
			}
			rawB, ok := bStore[e]
			if !ok {
				continue
			}
			a, okA := rawA.(*A)
			b, okB := rawB.(*B)
			if !okA || !okB {
				continue
			}
			fn(e, a, b)
		}
		return
	}

	for e, rawB := range bStore {
		if !w.isAlive(e) {
			continue
		}
		rawA, ok := aStore[e]
		if !ok {
			continue
		}
		a, okA := rawA.(*A)
		b, okB := rawB.(*B)
		if !okA || !okB {
			continue
		}
		fn(e, a, b)
	}
}

func (w *World) isAlive(e Entity) bool {
	_, ok := w.alive[e]
	return ok
}

func (w *World) requireAlive(e Entity) {
	if !w.isAlive(e) {
		panic("ecs: entity does not exist")
	}
}

func (w *World) ensureStore(typeID reflect.Type) map[Entity]any {
	components, ok := w.store[typeID]
	if ok {
		return components
	}
	components = make(map[Entity]any)
	w.store[typeID] = components
	return components
}

func componentType[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}
