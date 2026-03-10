package engine

import (
	"fmt"
)

// RenderPassType группирует проходы по роли в пайплайне.
type RenderPassType int

const (
	PassShadow RenderPassType = iota
	PassGeometry
	PassLighting
	PassPostProcess
	PassUI
)

// RenderResource — именованный runtime-ресурс между pass'ами.
// Data хранится как interface{}, ответственность за тип на стороне потребителя.
type RenderResource struct {
	Name string
	Data interface{}
}

// RenderPass описывает узел графа рендера.
// Inputs/Outputs используются для вычисления порядка выполнения.
type RenderPass interface {
	Name() string
	Type() RenderPassType
	Inputs() []string
	Outputs() []string
	Execute(ctx *RenderContext) error
}

// RenderContext содержит shared-ресурсы графа и прямой доступ к engine Context.
type RenderContext struct {
	Resources map[string]*RenderResource
	Context   *Context
}

func NewRenderContext(ctx *Context) *RenderContext {
	return &RenderContext{
		Resources: make(map[string]*RenderResource, 8),
		Context:   ctx,
	}
}

func (rc *RenderContext) SetResource(name string, data interface{}) {
	if resource, ok := rc.Resources[name]; ok {
		resource.Data = data
		return
	}
	rc.Resources[name] = &RenderResource{Name: name, Data: data}
}

func (rc *RenderContext) GetResource(name string) (interface{}, bool) {
	if res, ok := rc.Resources[name]; ok {
		return res.Data, true
	}
	return nil, false
}

// RenderGraph хранит набор проходов и их топологически отсортированный порядок.
type RenderGraph struct {
	passes []RenderPass
	sorted []RenderPass
}

func NewRenderGraph() *RenderGraph {
	return &RenderGraph{
		passes: make([]RenderPass, 0),
	}
}

func (rg *RenderGraph) AddPass(pass RenderPass) {
	rg.passes = append(rg.passes, pass)
	rg.sorted = nil // invalidate sort
}

// Build строит топологический порядок по зависимостям Inputs -> Outputs.
func (rg *RenderGraph) Build() error {
	visited := make(map[string]bool)
	temp := make(map[string]bool)
	sorted := make([]RenderPass, 0, len(rg.passes))

	passMap := make(map[string]RenderPass)
	for _, p := range rg.passes {
		passMap[p.Name()] = p
	}

	var visit func(string) error
	visit = func(name string) error {
		if temp[name] {
			return fmt.Errorf("circular dependency detected at pass: %s", name)
		}
		if visited[name] {
			return nil
		}

		temp[name] = true
		pass := passMap[name]

		for _, input := range pass.Inputs() {
			for _, p := range rg.passes {
				for _, output := range p.Outputs() {
					if output == input {
						if err := visit(p.Name()); err != nil {
							return err
						}
					}
				}
			}
		}

		temp[name] = false
		visited[name] = true
		sorted = append(sorted, pass)
		return nil
	}

	for _, pass := range rg.passes {
		if !visited[pass.Name()] {
			if err := visit(pass.Name()); err != nil {
				return err
			}
		}
	}

	rg.sorted = sorted
	return nil
}

// Execute выполняет проходы в вычисленном порядке.
func (rg *RenderGraph) Execute(ctx *RenderContext) error {
	if rg.sorted == nil {
		if err := rg.Build(); err != nil {
			return err
		}
	}

	for _, pass := range rg.sorted {
		if err := pass.Execute(ctx); err != nil {
			return fmt.Errorf("pass %s failed: %w", pass.Name(), err)
		}
	}

	return nil
}

func (rg *RenderGraph) Clear() {
	rg.passes = rg.passes[:0]
	rg.sorted = nil
}

// GetPassesByType возвращает подмножество проходов по типу.
func (rg *RenderGraph) GetPassesByType(passType RenderPassType) []RenderPass {
	result := make([]RenderPass, 0)
	for _, p := range rg.passes {
		if p.Type() == passType {
			result = append(result, p)
		}
	}
	return result
}

// PrintExecutionOrder выводит порядок выполнения и связи ресурсов.
func (rg *RenderGraph) PrintExecutionOrder() {
	if rg.sorted == nil {
		rg.Build()
	}
	fmt.Println("=== Render Graph Execution Order ===")
	for i, pass := range rg.sorted {
		fmt.Printf("%d. %s (Type: %d)\n", i+1, pass.Name(), pass.Type())
		fmt.Printf("   Inputs:  %v\n", pass.Inputs())
		fmt.Printf("   Outputs: %v\n", pass.Outputs())
	}
}
