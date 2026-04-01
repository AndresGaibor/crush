# Hooks System - Fase 2

El sistema de hooks permite ejecutar código personalizado en puntos específicos del ciclo de vida del agente Crush.

## Eventos Soportados

- **PreToolUse**: Antes de ejecutar una herramienta
- **PostToolUse**: Después de ejecutar una herramienta
- **SessionStart**: Al inicio de una nueva sesión
- **Stop**: Cuando el agente finaliza una respuesta

## Configuración

Los hooks se configuran en `crush.json` bajo la clave `hooks`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash(rm*)",
        "command": "echo '{\"continue\": false, \"reason\": \"Delete command blocked\"}'",
        "timeout": 5000
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Write",
        "command": "prettier --write $CRUSH_PROJECT_DIR/$FILE_PATH",
        "timeout": 15000
      }
    ],
    "SessionStart": [
      {
        "command": "echo 'Session started!'",
        "timeout": 5000
      }
    ],
    "Stop": [
      {
        "command": "echo 'Agent finished' >> /tmp/crush.log",
        "timeout": 3000
      }
    ]
  }
}
```

## Tipos de Matcher

- **Vacío o `*`**: Matchea todos los eventos
- **Pipe-separated**: `"Write|Edit|Bash"` - Matchea exactamente contra tool_name
- **Glob con paréntesis**: `"Bash(rm*)"` - Glob match contra tool_name + input
- **Regex**: Cualquier otro string se interpreta como expresión regular

## Respuestas de Hooks

Los hooks se comunican por stdin/stdout en JSON:

```json
{
  "continue": false,
  "decision": "deny",
  "reason": "Tool execution denied",
  "systemMessage": "Warning: dangerous operation",
  "suppressOutput": false,
  "additionalContext": "Extra info for the model"
}
```

### Códigos de Salida

- **0**: Éxito - stdout contiene respuesta JSON
- **2**: Error bloqueante - stderr se muestra y bloquea la herramienta
- Otros: Error estándar - se registra pero no bloquea

## Variables de Entorno

Los hooks reciben estas variables de entorno:

- `CRUSH_HOOK_EVENT`: Tipo de evento (PreToolUse, PostToolUse, etc.)
- `CRUSH_PROJECT_DIR`: Directorio del proyecto
- `CRUSH_TOOL_NAME`: Nombre de la herramienta (para eventos de tool)
- `CRUSH_SESSION_ID`: ID de la sesión

## Ejemplos

### Bloquear comandos peligrosos

```json
{
  "PreToolUse": [
    {
      "matcher": "Bash(rm -rf *)",
      "command": "echo '{\"continue\": false, \"reason\": \"Deletions not allowed\"}' && exit 2"
    }
  ]
}
```

### Ejecutar formateador después de escribir

```json
{
  "PostToolUse": [
    {
      "matcher": "Write",
      "command": "cd $CRUSH_PROJECT_DIR && prettier --write $TOOL_INPUT",
      "timeout": 15000
    }
  ]
}
```

### Ejecutar una sola vez

```json
{
  "SessionStart": [
    {
      "command": "cat .env.example > .env",
      "once": true
    }
  ]
}
```

## Implementación

El sistema de hooks está integrado en:

- `internal/personal/hooks/` - Implementación del sistema
- `internal/agent/agent.go` - Integración con el ciclo de vida del agente
- `internal/agent/event.go` - Hooks de eventos de sesión
- `internal/app/app.go` - Inicialización del sistema

## Testing

```bash
go test -v ./internal/personal/hooks/...
```
