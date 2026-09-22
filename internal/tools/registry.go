package tools

import (
	"fmt"
	"sync"

	"github.com/mtariq99/dispatchai/models"
)

// This file maintains the set of tool definitions available to the
// Assistant for a request or Host Project.
//
// This registry is NOT a registry of business implementations owned by
// DispatchAI.
//
// The Host Project provides the available capabilities.
//
// DispatchAI stores those definitions so they can be:
//
//   - validated
//   - looked up
//   - supplied to the LLM
//   - checked when the LLM returns a tool call
//
// The actual execution still happens inside the Host Project.

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("cannot register nil tool")
	}
	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q is already registered", name)
	}
	r.tools[name] = tool
	return nil
}

// Get returns a registered tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]

	return tool, exists
}

// Definitions returns the definitions of all registered tools.
//
// These definitions are later translated into the provider-specific tool
// format by the LLM adapter.
func (r *Registry) Definitions() []models.Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	definitions := make([]models.Definition, 0, len(r.tools))

	for _, tool := range r.tools {
		definitions = append(definitions, tool.Definition())
	}

	return definitions
}

// Has reports whether a tool is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tools[name]

	return exists
}

// Len returns the number of registered tools.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}
