package engine

import "github.com/go-gl/mathgl/mgl32"

// Material задает параметры модели освещения (Phong/Blinn-Phong).
type Material struct {
	Ambient   mgl32.Vec3
	Diffuse   mgl32.Vec3
	Specular  mgl32.Vec3
	Emission  mgl32.Vec3
	Shininess float32
	// SpecularStrength — скалярный множитель бликов для тонкой настройки материала.
	SpecularStrength float32
}

func DefaultMaterial() Material {
	return MaterialPlastic
}

// SanitizeMaterial нормализует параметры материала перед отправкой в shader-layer.
func SanitizeMaterial(mat Material) Material {
	defaultMat := DefaultMaterial()

	mat.Ambient = clampMaterialColor(mat.Ambient)
	mat.Diffuse = clampMaterialColor(mat.Diffuse)
	mat.Specular = clampMaterialColor(mat.Specular)
	mat.Emission = clampMaterialColor(mat.Emission)

	if mat.Ambient.Len() < 0.0001 && mat.Diffuse.Len() < 0.0001 {
		mat.Ambient = defaultMat.Ambient
		mat.Diffuse = defaultMat.Diffuse
	}
	if mat.Specular.Len() < 0.0001 {
		mat.Specular = defaultMat.Specular
	}
	if mat.Shininess <= 0 {
		mat.Shininess = 1.0
	}
	if mat.SpecularStrength < 0 {
		mat.SpecularStrength = 0
	}
	return mat
}

// NewMaterial создает материал из явно заданных коэффициентов.
func NewMaterial(ambient, diffuse, specular mgl32.Vec3, shininess float32) Material {
	return Material{
		Ambient:          ambient,
		Diffuse:          diffuse,
		Specular:         specular,
		Emission:         mgl32.Vec3{0, 0, 0},
		Shininess:        shininess,
		SpecularStrength: 1.0,
	}
}

// NewMaterialWithEmission создает материал с эмиссией (для будущих эффектов типа биолюминесценции).
func NewMaterialWithEmission(ambient, diffuse, specular, emission mgl32.Vec3, shininess float32) Material {
	return Material{
		Ambient:          ambient,
		Diffuse:          diffuse,
		Specular:         specular,
		Emission:         emission,
		Shininess:        shininess,
		SpecularStrength: 1.0,
	}
}

// ShininessFromSmoothness конвертирует "гладкость" [0..1] в степень блика Phong/Blinn-Phong.
func ShininessFromSmoothness(smoothness float32) float32 {
	if smoothness < 0 {
		smoothness = 0
	}
	if smoothness > 1 {
		smoothness = 1
	}
	return 1.0 + smoothness*255.0
}

// Набор готовых материалов для быстрого прототипирования.
// Значения близки к распространенным таблицам Phong-материалов.
var (
	MaterialEmerald = Material{
		Ambient:          mgl32.Vec3{0.0215, 0.1745, 0.0215},
		Diffuse:          mgl32.Vec3{0.07568, 0.61424, 0.07568},
		Specular:         mgl32.Vec3{0.633, 0.727811, 0.633},
		Shininess:        76.8,
		SpecularStrength: 1.0,
	}

	MaterialJade = Material{
		Ambient:          mgl32.Vec3{0.135, 0.2225, 0.1575},
		Diffuse:          mgl32.Vec3{0.54, 0.89, 0.63},
		Specular:         mgl32.Vec3{0.316228, 0.316228, 0.316228},
		Shininess:        12.8,
		SpecularStrength: 0.78,
	}

	MaterialGold = Material{
		Ambient:          mgl32.Vec3{0.24725, 0.1995, 0.0745},
		Diffuse:          mgl32.Vec3{0.75164, 0.60648, 0.22648},
		Specular:         mgl32.Vec3{0.628281, 0.555802, 0.366065},
		Shininess:        51.2,
		SpecularStrength: 1.0,
	}

	MaterialSilver = Material{
		Ambient:          mgl32.Vec3{0.19225, 0.19225, 0.19225},
		Diffuse:          mgl32.Vec3{0.50754, 0.50754, 0.50754},
		Specular:         mgl32.Vec3{0.508273, 0.508273, 0.508273},
		Shininess:        51.2,
		SpecularStrength: 1.0,
	}

	MaterialRubber = Material{
		Ambient:          mgl32.Vec3{0.02, 0.02, 0.02},
		Diffuse:          mgl32.Vec3{0.01, 0.01, 0.01},
		Specular:         mgl32.Vec3{0.4, 0.4, 0.4},
		Shininess:        10.0,
		SpecularStrength: 0.25,
	}

	MaterialPlastic = Material{
		Ambient:          mgl32.Vec3{0.0, 0.0, 0.0},
		Diffuse:          mgl32.Vec3{0.55, 0.55, 0.55},
		Specular:         mgl32.Vec3{0.7, 0.7, 0.7},
		Shininess:        32.0,
		SpecularStrength: 0.58,
	}

	MaterialSand = Material{
		Ambient:          mgl32.Vec3{0.3, 0.25, 0.2},
		Diffuse:          mgl32.Vec3{0.82, 0.72, 0.52},
		Specular:         mgl32.Vec3{0.1, 0.1, 0.1},
		Shininess:        8.0,
		SpecularStrength: 0.18,
	}

	MaterialWater = Material{
		Ambient:          mgl32.Vec3{0.05, 0.15, 0.25},
		Diffuse:          mgl32.Vec3{0.1, 0.4, 0.7},
		Specular:         mgl32.Vec3{0.8, 0.9, 1.0},
		Shininess:        128.0,
		SpecularStrength: 1.35,
	}

	MaterialPlant = Material{
		Ambient:          mgl32.Vec3{0.1, 0.15, 0.1},
		Diffuse:          mgl32.Vec3{0.3, 0.6, 0.3},
		Specular:         mgl32.Vec3{0.2, 0.3, 0.2},
		Shininess:        16.0,
		SpecularStrength: 0.35,
	}
)

func clampMaterialColor(color mgl32.Vec3) mgl32.Vec3 {
	if color.X() < 0 {
		color[0] = 0
	}
	if color.Y() < 0 {
		color[1] = 0
	}
	if color.Z() < 0 {
		color[2] = 0
	}
	return color
}
