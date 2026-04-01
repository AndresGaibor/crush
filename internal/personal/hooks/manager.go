package hooks

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Manager es el gestor central del sistema de hooks.
// Se encarga de registrar hooks, hacer matching de eventos,
// ejecutar hooks y recolectar resultados.
type Manager struct {
	config    HookConfigMap
	executor  *ShellExecutor
	cwd       string
	mu        sync.RWMutex
	onceRun   map[string]bool // Para hooks con Once=true
}

// NewManager crea un nuevo HookManager.
func NewManager(cwd string, config HookConfigMap) *Manager {
	env := []string{} // Se puede pasar el entorno actual si es necesario
	// Nota: en la integración final, pasa os.Environ() o filtra las vars necesarias.
	return &Manager{
		config:   config,
		executor: NewShellExecutor(cwd, env),
		cwd:      cwd,
		onceRun:  make(map[string]bool),
	}
}

// LoadConfig reemplaza la configuración de hooks.
// Se puede llamar para recargar la configuración sin reiniciar.
func (m *Manager) LoadConfig(config HookConfigMap) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
	m.onceRun = make(map[string]bool) // Reset once counters
	slog.Info("Hooks config reloaded")
}

// Fire dispara todos los hooks registrados para un evento dado.
// Retorna el primer resultado que bloquea (si lo hay) y los resultados
// de todos los hooks ejecutados.
func (m *Manager) Fire(ctx context.Context, event HookEvent) *FireResult {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()

	hooks, ok := config[event.Type]
	if !ok || len(hooks) == 0 {
		return &FireResult{Fired: false}
	}

	results := make([]FireResultEntry, 0, len(hooks))
	var blocking *HookResult

	for i, hookCfg := range hooks {
		// Verificar si ya se ejecutó (Once=true)
		onceKey := fmt.Sprintf("%s/%d/%s", event.Type, i, hookCfg.Command)
		if hookCfg.Once {
			if m.wasOnceRun(onceKey) {
				continue
			}
			m.markOnceRun(onceKey)
		}

		// Verificar matcher
		if !MatchHook(hookCfg.Matcher, event) {
			continue
		}

		// Ejecutar hook
		slog.Debug("Executing hook",
			"event", event.Type,
			"tool", event.ToolName,
			"matcher", hookCfg.Matcher,
			"command", hookCfg.Command,
			"index", i,
		)

		result, err := m.executor.Execute(ctx, hookCfg, event)

		entry := FireResultEntry{
			HookConfig: hookCfg,
			Result:     result,
			Error:      err,
			Index:      i,
		}
		results = append(results, entry)

		if err != nil {
			slog.Warn("Hook execution failed",
				"event", event.Type,
				"command", hookCfg.Command,
				"error", err,
			)
			continue
		}

		// Log del resultado
		if result.ShouldBlock() {
			slog.Info("Hook blocked execution",
				"event", event.Type,
				"tool", event.ToolName,
				"command", hookCfg.Command,
				"reason", result.Reason,
				"exit_code", result.ExitCode,
			)
			if blocking == nil {
				blocking = result
			}
			// Para PreToolUse, detener en el primer bloqueo
			if event.Type == PreToolUse {
				break
			}
		} else {
			slog.Debug("Hook passed",
				"event", event.Type,
				"tool", event.ToolName,
				"command", hookCfg.Command,
				"duration", result.Duration,
			)
		}
	}

	return &FireResult{
		Fired:    len(results) > 0,
		Results:  results,
		Blocking: blocking,
	}
}

// FirePreToolUse es un helper específico para PreToolUse.
// Retorna el resultado consolidado: nil si no hay bloqueo.
func (m *Manager) FirePreToolUse(ctx context.Context, sessionID, toolName, toolUseID string, toolInput interface{}) *HookResult {
	event := HookEvent{
		Type:      PreToolUse,
		SessionID: sessionID,
		Cwd:       m.cwd,
		ToolName:  toolName,
		ToolInput: toolInput,
		ToolUseID: toolUseID,
	}
	fireResult := m.Fire(ctx, event)

	// Recolectar contexto adicional de todos los hooks
	var additionalContexts []string
	for _, entry := range fireResult.Results {
		if entry.Result != nil && entry.Result.AdditionalContext != "" {
			additionalContexts = append(additionalContexts, entry.Result.AdditionalContext)
		}
	}

	if fireResult.Blocking != nil {
		fireResult.Blocking.AdditionalContext = joinNonEmpty("\n", additionalContexts...)
		return fireResult.Blocking
	}

	// Si no hay bloqueo pero hay contexto adicional, retornar resultado con contexto
	if len(additionalContexts) > 0 {
		return &HookResult{
			Continue:          boolPtr(true),
			AdditionalContext: joinNonEmpty("\n", additionalContexts...),
		}
	}

	return nil // No hooks ejecutados o todos pasaron
}

// FirePostToolUse es un helper específico para PostToolUse.
func (m *Manager) FirePostToolUse(ctx context.Context, sessionID, toolName, toolUseID string, toolInput, toolOutput interface{}) *HookResult {
	event := HookEvent{
		Type:       PostToolUse,
		SessionID:  sessionID,
		Cwd:        m.cwd,
		ToolName:   toolName,
		ToolInput:  toolInput,
		ToolOutput: toolOutput,
		ToolUseID:  toolUseID,
	}
	fireResult := m.Fire(ctx, event)

	var additionalContexts []string
	for _, entry := range fireResult.Results {
		if entry.Result != nil && entry.Result.AdditionalContext != "" {
			additionalContexts = append(additionalContexts, entry.Result.AdditionalContext)
		}
	}

	if fireResult.Blocking != nil {
		fireResult.Blocking.AdditionalContext = joinNonEmpty("\n", additionalContexts...)
		return fireResult.Blocking
	}

	if len(additionalContexts) > 0 {
		return &HookResult{
			Continue:          boolPtr(true),
			AdditionalContext: joinNonEmpty("\n", additionalContexts...),
		}
	}

	return nil
}

// FireSessionStart es un helper específico para SessionStart.
func (m *Manager) FireSessionStart(ctx context.Context, sessionID, source string) *HookResult {
	event := HookEvent{
		Type:      SessionStart,
		SessionID: sessionID,
		Cwd:       m.cwd,
		Source:    source,
	}
	fireResult := m.Fire(ctx, event)

	var additionalContexts []string
	for _, entry := range fireResult.Results {
		if entry.Result != nil && entry.Result.AdditionalContext != "" {
			additionalContexts = append(additionalContexts, entry.Result.AdditionalContext)
		}
	}

	if fireResult.Blocking != nil {
		fireResult.Blocking.AdditionalContext = joinNonEmpty("\n", additionalContexts...)
		return fireResult.Blocking
	}

	if len(additionalContexts) > 0 {
		return &HookResult{
			Continue:          boolPtr(true),
			AdditionalContext: joinNonEmpty("\n", additionalContexts...),
		}
	}

	return nil
}

// FireStop es un helper específico para Stop.
func (m *Manager) FireStop(ctx context.Context, sessionID string, lastMessage string) *HookResult {
	event := HookEvent{
		Type:      Stop,
		SessionID: sessionID,
		Cwd:       m.cwd,
		Message:   lastMessage,
	}
	fireResult := m.Fire(ctx, event)

	if fireResult.Blocking != nil {
		return fireResult.Blocking
	}

	return nil
}

// HookCount retorna la cantidad de hooks registrados por tipo.
func (m *Manager) HookCount() map[HookEventType]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counts := make(map[HookEventType]int)
	for event, hooks := range m.config {
		counts[event] = len(hooks)
	}
	return counts
}

func (m *Manager) wasOnceRun(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.onceRun[key]
}

func (m *Manager) markOnceRun(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onceRun[key] = true
}

// FireResult contiene el resultado de disparar hooks para un evento.
type FireResult struct {
	Fired    bool              // Si se ejecutó al menos un hook
	Results  []FireResultEntry // Todos los resultados
	Blocking *HookResult       // Primer resultado bloqueante (o nil)
}

// FireResultEntry contiene el resultado individual de un hook.
type FireResultEntry struct {
	HookConfig HookConfig
	Result     *HookResult
	Error      error
	Index      int
}

func joinNonEmpty(sep string, parts ...string) string {
	var filtered []string
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	switch len(filtered) {
	case 0:
		return ""
	case 1:
		return filtered[0]
	default:
		result := filtered[0]
		for _, p := range filtered[1:] {
			result += sep + p
		}
		return result
	}
}
