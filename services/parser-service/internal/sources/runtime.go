package sources

import (
	"strings"
	"sync"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type RuntimeRegistry struct {
	mu      sync.RWMutex
	sources map[string]parserv1.ParserSource
	order   []string
}

func NewRuntimeRegistry(sourceList []parserv1.ParserSource) *RuntimeRegistry {
	registry := &RuntimeRegistry{
		sources: make(map[string]parserv1.ParserSource, len(sourceList)),
		order:   make([]string, 0, len(sourceList)),
	}
	for _, source := range sourceList {
		name := strings.TrimSpace(source.Name)
		if name == "" {
			continue
		}
		source.Name = name
		if _, exists := registry.sources[name]; !exists {
			registry.order = append(registry.order, name)
		}
		registry.sources[name] = source
	}
	return registry
}

func (registry *RuntimeRegistry) List() []parserv1.ParserSource {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	result := make([]parserv1.ParserSource, 0, len(registry.order))
	for _, name := range registry.order {
		result = append(result, registry.sources[name])
	}
	return result
}

func (registry *RuntimeRegistry) SetEnabled(name string, enabled bool) (parserv1.ParserSource, bool) {
	name = strings.TrimSpace(name)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	source, ok := registry.sources[name]
	if !ok {
		return parserv1.ParserSource{}, false
	}
	source.Enabled = enabled
	registry.sources[name] = source
	return source, true
}

func (registry *RuntimeRegistry) Enabled(name string) bool {
	name = strings.TrimSpace(name)
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	source, ok := registry.sources[name]
	return ok && source.Enabled && source.Searchable
}
