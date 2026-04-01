package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// ShellExecutor ejecuta hooks de tipo "command" (comandos del shell).
type ShellExecutor struct {
	defaultShell   string
	defaultTimeout time.Duration
	cwd            string
	env            []string
}

// NewShellExecutor crea un nuevo executor de comandos shell.
func NewShellExecutor(cwd string, env []string) *ShellExecutor {
	shell := "/bin/sh"
	if s, err := exec.LookPath("bash"); err == nil {
		shell = s
	}
	return &ShellExecutor{
		defaultShell:   shell,
		defaultTimeout: 30 * time.Second,
		cwd:            cwd,
		env:            env,
	}
}

// Execute ejecuta un hook de comando shell.
// El evento se serializa a JSON y se envía por stdin.
// La salida del hook se parsea como JSON (si es válido) o se trata como texto plano.
func (e *ShellExecutor) Execute(ctx context.Context, cfg HookConfig, event HookEvent) (*HookResult, error) {
	start := time.Now()

	// Serializar evento a JSON
	inputJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshaling hook event: %w", err)
	}

	// Determinar shell y timeout
	shell := e.defaultShell
	if cfg.Shell != "" {
		shell = cfg.Shell
	}
	timeout := e.defaultTimeout
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Millisecond
	}

	// Crear contexto con timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Ejecutar comando
	cmd := exec.CommandContext(ctx, shell, "-c", cfg.Command)
	cmd.Dir = e.cwd
	cmd.Env = e.buildEnv(event)

	// Pipes para stdin/stdout/stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(inputJSON)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	result := &HookResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}

	// Determinar exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			// Error del sistema (no del comando) — probablemente timeout
			result.ExitCode = 1
			result.Stderr += "\nHook timed out or failed to start"
		}
	} else {
		result.ExitCode = 0
	}

	// Intentar parsear stdout como JSON
	if result.Stdout != "" {
		if parsed, parseErr := parseHookOutput(result.Stdout); parseErr == nil {
			mergeResult(result, parsed)
		}
		// Si no es JSON válido, el stdout se usa como contexto adicional
		if result.AdditionalContext == "" {
			result.AdditionalContext = result.Stdout
		}
	}

	// Default: permitir si no se especificó otra cosa
	if result.Continue == nil && result.ExitCode != 2 && result.Decision != "deny" {
		result.Continue = boolPtr(true)
	}

	return result, nil
}

// buildEnv construye las variables de entorno para el hook.
func (e *ShellExecutor) buildEnv(event HookEvent) []string {
	env := append([]string{}, e.env...)
	env = append(env,
		"CRUSH_HOOK_EVENT="+string(event.Type),
		"CRUSH_PROJECT_DIR="+event.Cwd,
	)
	if event.ToolName != "" {
		env = append(env, "CRUSH_TOOL_NAME="+event.ToolName)
	}
	if event.SessionID != "" {
		env = append(env, "CRUSH_SESSION_ID="+event.SessionID)
	}
	return env
}

// parseHookOutput intenta parsear la salida de un hook como JSON.
func parseHookOutput(output string) (*HookResult, error) {
	var result HookResult
	// Limpiar output (puede tener nueva línea al final)
	output = trimOutput(output)
	if output == "" {
		return nil, fmt.Errorf("empty output")
	}
	err := json.Unmarshal([]byte(output), &result)
	return &result, err
}

// mergeResult combina un resultado parseado con el resultado base.
func mergeResult(base, parsed *HookResult) {
	if parsed.Continue != nil {
		base.Continue = parsed.Continue
	}
	if parsed.Decision != "" {
		base.Decision = parsed.Decision
	}
	if parsed.Reason != "" {
		base.Reason = parsed.Reason
	}
	if parsed.SuppressOutput {
		base.SuppressOutput = true
	}
	if parsed.SystemMessage != "" {
		base.SystemMessage = parsed.SystemMessage
	}
	if parsed.UpdatedInput != nil {
		base.UpdatedInput = parsed.UpdatedInput
	}
	if parsed.AdditionalContext != "" {
		base.AdditionalContext = parsed.AdditionalContext
	}
}

// trimOutput limpia whitespace de la salida del hook.
func trimOutput(s string) string {
	s = trimLeadingNewlines(s)
	s = trimTrailingNewlines(s)
	return s
}

func trimLeadingNewlines(s string) string {
	for len(s) > 0 && (s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	return s
}

func trimTrailingNewlines(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func boolPtr(b bool) *bool { return &b }
