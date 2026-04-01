package hooks

import (
	"context"
	"time"
)

// HookEventType define los tipos de eventos que disparan hooks.
type HookEventType string

const (
	PreToolUse   HookEventType = "PreToolUse"
	PostToolUse  HookEventType = "PostToolUse"
	SessionStart HookEventType = "SessionStart"
	Stop         HookEventType = "Stop"
)

// AllHookEvents retorna la lista de eventos soportados.
func AllHookEvents() []HookEventType {
	return []HookEventType{PreToolUse, PostToolUse, SessionStart, Stop}
}

// HookConfig es la configuración de un hook individual desde crush.json.
type HookConfig struct {
	Matcher string `json:"matcher,omitempty"` // Pattern: "Bash", "Write", "Bash(rm *)", "*"
	Command string `json:"command"`           // Comando a ejecutar
	Shell   string `json:"shell,omitempty"`   // Shell a usar (default: /bin/sh)
	Timeout int    `json:"timeout,omitempty"` // Timeout en ms (default: 30000)
	Enabled *bool  `json:"enabled,omitempty"` // Habilitado/deshabilitado
	Once    bool   `json:"once,omitempty"`    // Ejecutar solo una vez
	Async   bool   `json:"async,omitempty"`   // Ejecutar en background
}

// HookConfigMap es el mapa de eventos → lista de hooks.
// Viene del campo "hooks" en crush.json.
type HookConfigMap map[HookEventType][]HookConfig

// HookEvent es el evento que se pasa a los hooks para ejecución.
type HookEvent struct {
	Type       HookEventType `json:"hook_event_name"`
	SessionID  string        `json:"session_id,omitempty"`
	Cwd        string        `json:"cwd,omitempty"`
	ToolName   string        `json:"tool_name,omitempty"`
	ToolInput  interface{}   `json:"tool_input,omitempty"`
	ToolOutput interface{}   `json:"tool_response,omitempty"`
	ToolUseID  string        `json:"tool_use_id,omitempty"`
	Source     string        `json:"source,omitempty"`      // SessionStart: "startup", "resume"
	Message    string        `json:"message,omitempty"`     // Stop: último mensaje
	StopReason string        `json:"stop_reason,omitempty"` // Stop: razón
}

// HookResult es la respuesta de un hook ejecutado.
type HookResult struct {
	// Continue indica si el flujo debe continuar (false = bloquear).
	Continue *bool `json:"continue,omitempty"`
	// Decision es la decisión para PreToolUse: "allow", "deny", "ask"
	Decision string `json:"decision,omitempty"`
	// Reason es el motivo de la decisión.
	Reason string `json:"reason,omitempty"`
	// SuppressOutput indica si se debe suprimir la salida del hook.
	SuppressOutput bool `json:"suppressOutput,omitempty"`
	// SystemMessage es un mensaje de advertencia para mostrar al usuario.
	SystemMessage string `json:"systemMessage,omitempty"`
	// UpdatedInput es el input modificado de la herramienta (PreToolUse).
	UpdatedInput interface{} `json:"updatedInput,omitempty"`
	// AdditionalContext es contexto adicional para el agente.
	AdditionalContext string `json:"additionalContext,omitempty"`

	// Metadata interno (no serializado)
	Stdout   string        `json:"-"`
	Stderr   string        `json:"-"`
	ExitCode int           `json:"-"`
	Duration time.Duration `json:"-"`
}

// ShouldBlock retorna true si el hook indica que el flujo debe detenerse.
func (r *HookResult) ShouldBlock() bool {
	if r.Continue != nil {
		return !*r.Continue
	}
	if r.Decision == "deny" {
		return true
	}
	return r.ExitCode == 2
}

// Hook es la interfaz que deben implementar todos los hooks.
type Hook interface {
	// Name retorna un identificador legible del hook.
	Name() string
	// Match evalúa si este hook debe ejecutarse para el evento dado.
	Match(event HookEvent) bool
	// Execute ejecuta el hook con el evento proporcionado.
	Execute(ctx context.Context, event HookEvent) (*HookResult, error)
}

// HookExecutor es la interfaz para ejecutar hooks concretos.
type HookExecutor interface {
	Execute(ctx context.Context, cfg HookConfig, event HookEvent) (*HookResult, error)
}
