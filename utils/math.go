package utils

import (
	"math"
	"math/rand"
	"time"
)

func init() {
	// Инициализируем генератор случайных чисел один раз при старте процесса.
	// Это важно, чтобы объекты (растения, существа) не спавнились в одних и тех же местах.
	rand.Seed(time.Now().UnixNano())
}

// SinDeg вычисляет синус угла, заданного в градусах.
func SinDeg(a float32) float32 {
	return float32(math.Sin(float64(a) * math.Pi / 180.0))
}

// CosDeg вычисляет косинус угла, заданного в градусах.
func CosDeg(a float32) float32 {
	return float32(math.Cos(float64(a) * math.Pi / 180.0))
}

// RandomRange возвращает равномерно распределённое значение в диапазоне [min, max).
func RandomRange(min, max float32) float32 {
	return min + rand.Float32()*(max-min)
}
