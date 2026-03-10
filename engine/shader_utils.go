package engine

import (
	"os"
	"path/filepath"
	"strings"
)

// loadShaderSourceFile читает GLSL-файл и гарантирует нуль-терминатор,
// который ожидает `gl.Strs` в Go-биндингах OpenGL.
//
// Функция пробует несколько относительных путей, чтобы код работал
// из разных рабочих директорий (root проекта, examples и т.д.).
func loadShaderSourceFile(path string) string {
	candidates := []string{
		path,
		filepath.Join("..", path),
		filepath.Join("..", "..", path),
	}

	var data []byte
	var err error
	for _, candidate := range candidates {
		data, err = os.ReadFile(candidate)
		if err == nil {
			break
		}
	}
	if err != nil {
		panic("load shader source failed: " + err.Error())
	}

	src := string(data)
	if !strings.HasSuffix(src, "\x00") {
		src += "\x00"
	}
	return src
}
