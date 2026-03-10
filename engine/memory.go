package engine

import (
	"fmt"
	"runtime"
)

// MemStats содержит компактную статистику памяти процесса в мегабайтах.
type MemStats struct {
	Alloc      uint64 // Текущий объем живых аллокаций.
	TotalAlloc uint64 // Суммарный объем всех аллокаций за время работы.
	Sys        uint64 // Память, запрошенная рантаймом у ОС.
	NumGC      uint32 // Количество циклов сборки мусора.
}

// GetMemStats считывает статистику из runtime.
func GetMemStats() MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return MemStats{
		Alloc:      m.Alloc / 1024 / 1024,
		TotalAlloc: m.TotalAlloc / 1024 / 1024,
		Sys:        m.Sys / 1024 / 1024,
		NumGC:      m.NumGC,
	}
}

// String форматирует статистику для заголовка окна/логов.
func (m MemStats) String() string {
	return fmt.Sprintf("Mem: %dMB | Total: %dMB | Sys: %dMB | GC: %d",
		m.Alloc, m.TotalAlloc, m.Sys, m.NumGC)
}
