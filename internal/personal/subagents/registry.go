package subagents

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Registry mantiene subagentes cargados y permite consultas rápidas.
type Registry struct {
	mu        sync.RWMutex
	paths     []string
	subagents map[string]*Subagent
}

// NewRegistry crea un registro vacío.
func NewRegistry(paths []string) *Registry {
	return &Registry{
		paths:     append([]string(nil), paths...),
		subagents: make(map[string]*Subagent),
	}
}

// Reload vuelve a leer los subagentes desde los paths actuales o los
// proporcionados.
func (r *Registry) Reload(paths []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if paths != nil {
		r.paths = append([]string(nil), paths...)
	}

	loaded := Discover(r.paths)
	subagentsByName := make(map[string]*Subagent, len(loaded))
	for _, subagent := range loaded {
		key := strings.ToLower(strings.TrimSpace(subagent.Name))
		if key == "" {
			continue
		}
		subagentsByName[key] = subagent
	}
	r.subagents = subagentsByName
	return nil
}

// All devuelve todos los subagentes cargados.
func (r *Registry) All() []*Subagent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	loaded := make([]*Subagent, 0, len(r.subagents))
	for _, subagent := range r.subagents {
		loaded = append(loaded, subagent)
	}
	sort.Slice(loaded, func(i, j int) bool {
		return strings.ToLower(loaded[i].Name) < strings.ToLower(loaded[j].Name)
	})
	return loaded
}

// Get busca un subagente por nombre exacto.
func (r *Registry) Get(name string) (*Subagent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	subagent, ok := r.subagents[strings.ToLower(strings.TrimSpace(name))]
	return subagent, ok
}

// Match intenta encontrar el mejor subagente para una tarea.
func (r *Registry) Match(task string) (*Subagent, bool) {
	task = strings.ToLower(strings.TrimSpace(task))
	if task == "" {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var (
		best      *Subagent
		bestScore int
	)
	for _, subagent := range r.subagents {
		if !subagent.AutoDelegate {
			continue
		}
		score := scoreSubagent(task, subagent)
		if score > bestScore {
			bestScore = score
			best = subagent
		}
	}

	if best == nil {
		return nil, false
	}
	return best, true
}

// Find resuelve un nombre exacto y, si no existe, hace una búsqueda por tarea.
func (r *Registry) Find(query string) (*Subagent, bool) {
	if subagent, ok := r.Get(query); ok {
		return subagent, true
	}
	return r.Match(query)
}

func scoreSubagent(task string, subagent *Subagent) int {
	score := 0
	name := strings.ToLower(subagent.Name)
	desc := strings.ToLower(subagent.Description)

	switch {
	case task == name:
		score += 100
	case strings.Contains(task, name):
		score += 70
	case strings.Contains(name, task):
		score += 50
	}

	if strings.Contains(desc, task) {
		score += 40
	}

	for _, word := range strings.Fields(task) {
		if len(word) < 3 {
			continue
		}
		if strings.Contains(name, word) {
			score += 10
		}
		if strings.Contains(desc, word) {
			score += 5
		}
	}

	return score
}

// ErrNotInitialized se retorna cuando el singleton todavía no se cargó.
var ErrNotInitialized = fmt.Errorf("subagents registry is not initialized")
