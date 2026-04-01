# Plan de Migración y Mejora: Crush (Go) + Arquitectura Claude Code (TypeScript)

> **Base:** Fork de [github.com/charmbracelet/crush](https://github.com/charmbracelet/crush) → [github.com/AndresGaibor/crush](https://github.com/AndresGaibor/crush)
> **Referencia:** Claude Code (TypeScript/Bun) de Anthropic
> **Objetivo:** Migrar características clave de Claude Code a Go, manteniendo Crush como base
> **Estrategia:** Fork + Módulo personal aislado + Rebase periódico de upstream
> **Fecha:** Abril 2026 | **Versión:** 1.0

---

## Tabla de Contenidos

1. [Resumen Ejecutivo](#1-resumen-ejecutivo)
2. [Análisis Comparativo de Arquitecturas](#2-análisis-comparativo-de-arquitecturas)
   - 2.1 [Visión General de Crush (Go)](#21-visión-general-de-crush-go)
   - 2.2 [Visión General de Claude Code (TypeScript)](#22-visión-general-de-claude-code-typescript)
3. [Estrategia de Fork y Mantenimiento](#3-estrategia-de-fork-y-mantenimiento)
   - 3.1 [Estructura del Repositorio](#31-estructura-del-repositorio)
   - 3.2 [Aislamiento de Módulos Personales](#32-aislamiento-de-módulos-personales)
4. [Plan de Migración por Fases](#4-plan-de-migración-por-fases)
   - 4.1 [Fase 1: Sistema de Memoria Persistente](#41-fase-1-sistema-de-memoria-persistente)
   - 4.2 [Fase 2: Sistema de Hooks](#42-fase-2-sistema-de-hooks)
   - 4.3 [Fase 3: Sistema de Plugins](#43-fase-3-sistema-de-plugins)
   - 4.4 [Fase 4: Compresión Inteligente de Contexto](#44-fase-4-compresión-inteligente-de-contexto)
   - 4.5 [Fase 5: Modo Plan y Tareas v2](#45-fase-5-modo-plan-y-tareas-v2)
   - 4.6 [Fase 6: Extensiones Avanzadas](#46-fase-6-extensiones-avanzadas)
5. [Nuevas Herramientas a Implementar](#5-nuevas-herramientas-a-implementar)
6. [Mapa de Archivos y Directorios Propuestos](#6-mapa-de-archivos-y-directorios-propuestos)
7. [Decisiones Técnicas Clave](#7-decisiones-técnicas-clave)
   - 7.1 [Dependencias Go Seleccionadas](#71-dependencias-go-seleccionadas)
   - 7.2 [Estrategia de Testing](#72-estrategia-de-testing)
   - 7.3 [Compatibilidad con Actualizaciones de Upstream](#73-compatibilidad-con-actualizaciones-de-upstream)
8. [Cronograma Estimado](#8-cronograma-estimado)
9. [Riesgos y Mitigaciones](#9-riesgos-y-mitigaciones)
10. [Conclusión y Próximos Pasos](#10-conclusión-y-próximos-pasos)

---

## 1. Resumen Ejecutivo

Este documento presenta un plan integral para migrar y mejorar **Crush**, el asistente de codificación con inteligencia artificial de terminal construido en Go por Charm, incorporando las características y patrones arquitectónicos avanzados de **Claude Code** de Anthropic (originalmente escrito en TypeScript con Bun). La meta es crear un fork personal que combine la solidez, rendimiento y multi-proveedor de Crush con la sofisticación en gestión de contexto, sistema de plugins, memoria persistente, hooks extensibles y orquestación multi-agente que Claude Code ha demostrado en producción.

Crush ya proporciona una base excepcional: un TUI completo con Bubble Tea v2, soporte para más de 10 proveedores LLM (Anthropic, OpenAI, Gemini, Bedrock, Azure, OpenRouter, Ollama, etc.), persistencia en SQLite mediante sqlc, integración LSP para inteligencia de código, soporte MCP para extensiones, y un sistema de herramientas con aproximadamente 20 herramientas integradas. Sin embargo, Claude Code introduce conceptos arquitectónicos que representan el estado del arte en asistentes de codificación AI: un sistema de memoria semántica con CLAUDE.md, un sistema de plugins con marketplace y hot-reload, hooks pre/post ejecución de herramientas, compresión inteligente de contexto (compact), modo plan para razonamiento estructurado, orquestación de enjambres de agentes, y un sistema de tareas v2 estructurado.

La estrategia fundamental se basa en **mantener un fork limpio de Crush**, desarrollando las nuevas funcionalidades como módulos internos que se integran de forma no invasiva con la arquitectura existente, y realizando **rebase periódicos de upstream** para absorber las actualizaciones de Charm sin conflictos excesivos. Este enfoque garantiza que el fork se beneficie de las mejoras continuas de Crush mientras incorpora las capacidades avanzadas inspiradas en Claude Code, todo escrito en Go idiomático para mantener la coherencia del proyecto.

---

## 2. Análisis Comparativo de Arquitecturas

### 2.1 Visión General de Crush (Go)

Crush es un asistente de codificación AI de terminal (TUI) construido completamente en **Go 1.26** por Charm. Utiliza Bubble Tea v2 como framework de interfaz de usuario, siguiendo el patrón arquitectónico Elm (Model-Update-View). Su objetivo principal es proporcionar una experiencia de codificación con AI directamente en la terminal, con soporte para múltiples proveedores LLM y herramientas de desarrollo integradas. El proyecto se distribuye como binarios estáticos multiplataforma mediante GoReleaser, sin dependencias de CGO, utilizando drivers SQLite puros en Go.

La arquitectura de Crush se organiza en aproximadamente **25 paquetes internos** bajo `internal/`, con una separación clara de responsabilidades:

| Paquete | Responsabilidad |
|---------|----------------|
| `agent/` | Orquestación de agentes y bucle de ejecución de herramientas |
| `ui/` | TUI completa con Bubble Tea (chat, sidebar, dialogs, diffview) |
| `config/` | Configuración multi-scope (proyecto, global, datos) |
| `db/` | Persistencia SQLite via sqlc + goose migrations |
| `lsp/` | Cliente LSP para inteligencia de código |
| `pubsub/` | Eventos desacoplados con Broker[T] genérico |
| `shell/` | Ejecución de comandos del shell con background jobs |
| `fsext/` | Extensiones del filesystem (ls, search, ignore) |
| `permission/` | Control de permisos de herramientas |
| `skills/` | Descubrimiento y carga de skills |
| `mcp/` | Integración Model Context Protocol |

**Estructura de directorios clave de Crush:**

```
crush/
├── main.go                     # Entry point
├── go.mod                      # Go 1.26.1, CGO_ENABLED=0
├── Taskfile.yaml               # Build, test, lint, release
├── .goreleaser.yml             # Cross-platform builds
└── internal/
    ├── app/                    # App wiring (DB, config, agents, LSP, MCP)
    ├── cmd/                    # Cobra CLI commands (root, run, login, session, etc.)
    ├── agent/                  # Agent loop, coordinator, prompts, tools
    │   ├── agent.go            # SessionAgent: LLM streaming + tool execution
    │   ├── coordinator.go      # Coordinator: named agents management
    │   ├── templates/          # Go template system prompts
    │   └── tools/              # ~20 built-in tools + MCP client
    ├── ui/                     # Bubble Tea v2 TUI
    │   ├── model/              # Main UI model
    │   ├── chat/               # Chat message rendering
    │   ├── dialog/             # Dialog boxes
    │   ├── diffview/           # Diff viewer (split & unified)
    │   └── common/             # Shared utilities (markdown, highlight, etc.)
    ├── config/                 # Configuration management
    ├── db/                     # SQLite via sqlc (sessions, messages, files, read_files)
    ├── lsp/                    # LSP client
    ├── pubsub/                 # Broker[T] event system
    ├── session/                # Session CRUD service
    ├── message/                # Message model + content types
    └── ...                     # shell, fsext, permission, skills, oauth, etc.
```

### 2.2 Visión General de Claude Code (TypeScript)

Claude Code es el asistente de codificación AI de Anthropic, construido con **TypeScript** y ejecutado mediante el runtime **Bun**. Su arquitectura es significativamente más compleja que la de Crush, con más de 30 directorios de primer nivel y cientos de archivos fuente. Utiliza un framework TUI propio basado en **Ink** (React para terminal), que ha sido profundamente personalizado con un layout engine propio basado en Yoga (Flexbox), un sistema de renderizado optimizado, y soporte avanzado para scrolling virtual, selección de texto, y atajos de teclado extensibles incluyendo modo Vim completo.

El patrón arquitectónico fundamental de Claude Code se basa en **QueryEngine**, una clase que owns el ciclo de vida completo de una conversación. QueryEngine encapsula los mensajes mutables, el estado de permisos, el seguimiento de uso (tokens/costo), y coordina el bucle de query que alterna entre llamadas al LLM y ejecución de herramientas. Cada herramienta se define como un objeto `Tool` con métodos para `call()`, `description()`, `checkPermissions()`, `prompt()`, y múltiples métodos de renderizado para la UI.

**Estructura de directorios clave de Claude Code:**

```
claude-code-main/
├── main.tsx                    # Entry point
├── setup.ts                    # App initialization
├── QueryEngine.ts              # Core query lifecycle management
├── Tool.ts                     # Tool interface + buildTool factory
├── tools.ts                    # Tool registry (~40 tools)
├── query.ts                    # Query loop (LLM ↔ tools)
├── tools/                      # Individual tool implementations
│   ├── AgentTool/              # Sub-agent spawning
│   ├── BashTool/               # Shell execution
│   ├── FileEditTool/           # Find/replace in files
│   ├── FileReadTool/           # Read files
│   ├── FileWriteTool/          # Write files
│   ├── GrepTool/               # Search with grep
│   ├── GlobTool/               # File pattern matching
│   ├── WebFetchTool/           # URL fetching
│   ├── WebSearchTool/          # Web search
│   ├── NotebookEditTool/       # Jupyter notebook editing
│   ├── TodoWriteTool/          # Todo management
│   ├── TaskCreateTool/         # Task CRUD
│   ├── TaskUpdateTool/
│   ├── TaskGetTool/
│   ├── TaskListTool/
│   ├── AskUserQuestionTool/    # Interactive user questions
│   ├── EnterPlanModeTool/      # Plan mode entry
│   ├── ExitPlanModeTool/       # Plan mode exit
│   ├── LSPTool/                # LSP integration
│   ├── SkillTool/              # Skill execution
│   ├── ConfigTool/             # Runtime config modification
│   ├── ToolSearchTool/         # Lazy tool schema search
│   ├── ScheduleCronTool/       # Cron scheduling
│   ├── SendMessageTool/        # Inter-agent messaging
│   └── ...                     # BriefTool, SleepTool, MCP tools, etc.
├── services/
│   ├── compact/                # Context compression (auto, micro, snip)
│   ├── mcp/                    # MCP client + auth
│   ├── lsp/                    # LSP manager
│   ├── tools/                  # Tool orchestration + streaming
│   ├── SessionMemory/          # Session memory persistence
│   └── api/                    # API client + auth + retry
├── hooks/                      # ~100 React hooks for UI
├── memdir/                     # Memory directory system (CLAUDE.md)
├── plugins/                    # Plugin system
├── skills/bundled/             # ~15 built-in skills
├── coordinator/                # Multi-agent coordinator mode
├── bridge/                     # Remote bridge (IDE, mobile, SDK)
├── ink/                        # Custom Ink (React) TUI framework
├── vim/                        # Vim mode implementation
├── state/                      # App state management
├── migrations/                 # Config migration utilities
├── keybindings/                # Extensible keybinding system
├── utils/                      # ~150 utility modules
│   ├── hooks/                  # Hook system (pre/post tool, session, file)
│   ├── plugins/                # Plugin loader, validator, hot-reload
│   ├── permissions/            # Permission system (allow/deny/ask)
│   ├── swarm/                  # Agent swarms (tmux, iTerm2, in-process)
│   ├── compact/                # Compact utilities
│   └── ...                     # git, shell, memory, model, etc.
├── commands/                   # CLI slash commands
├── screens/                    # UI screens (REPL, Doctor, etc.)
└── entrypoints/                # CLI, SDK, MCP entrypoints
```

### Comparativa Directa

| Componente | Crush (Go) | Claude Code (TS) |
|------------|-----------|-----------------|
| **Lenguaje** | Go 1.26 (CGO=0) | TypeScript / Bun |
| **TUI Framework** | Bubble Tea v2 (Charm) | Ink (React para terminal, propio) |
| **CLI Framework** | Cobra | Propio con entrypoints múltiples |
| **Persistencia** | SQLite (sqlc + goose) | JSONL transcripciones + state |
| **LLM Providers** | fantasy (10+ providers) | Anthropic SDK directo + bridges |
| **Herramientas** | ~20 built-in + MCP | ~40 built-in + MCP + Plugins |
| **Plugin System** | No existe | Marketplace + hot-reload + versioning |
| **Memory System** | Skills básicos | CLAUDE.md + memdir + semantic scan |
| **Hooks** | No existe | Pre/Post tool + Session + FileChange |
| **Context Mgmt** | Auto-summarization | Compact + MicroCompact + Snip |
| **Multi-Agent** | Coordinator básico | Coordinator + Swarm + Teammates |
| **Plan Mode** | No existe | Enter/Exit Plan Mode + Plan Agent |
| **Task System** | Todo básico | Tasks v2 (CRUD estructurado) |
| **Cron/Schedule** | No existe | CronCreate/CronDelete/CronList |
| **Keybindings** | Fijo | Extensible con archivo de config |
| **Vim Mode** | No existe | Vim completo (motions, operators, text objects) |

---

## 3. Estrategia de Fork y Mantenimiento

### 3.1 Estructura del Repositorio

El repositorio personal se mantiene como un fork de `github.com/charmbracelet/crush` en `github.com/AndresGaibor/crush`. La estrategia clave es mantener los cambios personales en **ramas de características separadas** (feature branches) que se rebasan periódicamente contra la rama `main` de upstream (charmbracelet/crush).

| Concepto | Implementación |
|----------|---------------|
| **Remote upstream** | `git remote add upstream https://github.com/charmbracelet/crush.git` |
| **Fork personal** | `github.com/AndresGaibor/crush` (origin) |
| **Branch principal** | `main` rebasada periódicamente sobre `upstream/main` |
| **Feature branches** | `feat/memory`, `feat/plugins`, `feat/hooks`, etc. |
| **Módulo personal** | `internal/personal/` (aislado del core de Crush) |
| **Rebase cadencia** | Semanal o quincenal, dependiendo de actividad upstream |

### 3.2 Aislamiento de Módulos Personales

Todas las nuevas funcionalidades inspiradas en Claude Code se desarrollan dentro del directorio **`internal/personal/`**, completamente aislado del código core de Crush. Este directorio contiene subpaquetes para cada subsistema nuevo:

```
internal/personal/
├── memory/       # Sistema de memoria persistente
├── hooks/        # Sistema de hooks pre/post ejecución
├── plugins/      # Sistema de plugins con hot-reload
├── compact/      # Compresión inteligente de contexto
├── planmode/     # Modo plan para razonamiento estructurado
├── tasks/        # Sistema de tareas v2
├── coordinator/  # Orquestación multi-agente
├── cron/         # Sistema de programación de tareas
└── tools/        # Nuevas herramientas individuales
```

**Puntos de extensión utilizados para integrarse con Crush:**

- **`pubsub/`** → Suscribirse a eventos de herramientas y sesiones
- **`agent/`** → Registrar nuevas herramientas
- **`config/`** → Agregar opciones de configuración personalizadas
- **`ui/`** → Agregar componentes de interfaz adicionales
- **`agent/templates/`** → Modificar system prompts para incluir contexto de memoria

Cada módulo personal registra sus extensiones durante la inicialización de la aplicación en `app.go`, mediante una función `Init()` que se llama condicionalmente si el módulo está habilitado en la configuración del usuario. El core de Crush **nunca importa directamente** los módulos personales → inversión de dependencias.

---

## 4. Plan de Migración por Fases

### 4.1 Fase 1: Sistema de Memoria Persistente (Prioridad: CRÍTICA)

El sistema de memoria es la característica más transformadora de Claude Code. Permite que el asistente recuerde contexto entre sesiones, proyectos y conversaciones, sin necesidad de repetir instrucciones. En Claude Code, este sistema se basa en archivos **CLAUDE.md jerárquicos** que se cargan automáticamente como contexto adicional.

#### 4.1.1 Archivos de Memoria Jerárquicos

Se implementa un sistema de archivos de memoria que sigue la jerarquía de Claude Code pero adaptada al estilo de Crush. Se utiliza un directorio `.crush/memory/` en la raíz del proyecto que contiene archivos Markdown con instrucciones, preferencias y contexto acumulado:

```
.crush/memory/
├── CLAUDE.md              # Instrucciones y convenciones del proyecto
├── preferences.md          # Preferencias personales del usuario
├── architecture.md         # Decisiones arquitectónicas importantes
└── patterns.md             # Patrones de uso detectados automáticamente

~/.config/crush/memory/
├── global.md              # Memoria global del usuario (cross-proyecto)
└── skills/                # Skills memorizadas aprendidas por el agente
```

**Integración con el system prompt:** El sistema carga automáticamente estos archivos como parte del system prompt del agente, utilizando los templates Go existentes en `internal/agent/templates/`. Se agrega una nueva sección al template `coder.md.tpl` que inyecta las memorias relevantes.

**Implementación en Go:** `internal/personal/memory/`

| Archivo | Responsabilidad |
|---------|----------------|
| `memory.go` | MemoryManager: Load, Save, Search, Delete memorias |
| `scanner.go` | Escaneo del filesystem para encontrar memorias relevantes |
| `loader.go` | Carga de memorias al system prompt via templates |
| `search.go` | Búsqueda por relevancia (TF-IDF / embeddings) |
| `autogen.go` | Auto-generación de memorias semilla |
| `age.go` | Envejecimiento y consolidación de memorias antiguas |
| `config.go` | Opciones de configuración del sistema de memoria |

El MemoryManager se integra con el App en `app.go` a través del pubsub, suscribiéndose a eventos de tipo `ToolExecuted` y `SessionEnded` para actualizar memorias automáticamente.

#### 4.1.2 Escaneo Semántico de Recuerdos

Claude Code implementa `findRelevantMemories()` que busca en el directorio de memorias archivos cuyo contenido es relevante para la consulta actual del usuario. En Go, esto se implementa como un **sistema híbrido**:

1. **Filtro rápido** basado en palabras clave y metadatos (tags en el frontmatter de los archivos Markdown)
2. **Ranking por relevancia** utilizando:
   - Opción A: Embeddings locales (small BGE model via ONNX Runtime en Go)
   - Opción B: Aproximación TF-IDF con el paquete `go-tfidf`

El resultado es un conjunto de memorias relevantes que se inyectan como contexto adicional en el system prompt con una sección dedicada.

#### 4.1.3 Auto-Generación de Memorias

Cuando el agente detecta patrones repetitivos en las interacciones (por ejemplo, el usuario siempre prefiere un framework específico, o corrige recurrentemente un estilo de código), genera automáticamente una nueva memoria semilla. Funcionalidad implementada como un hook post-tool que:

1. Analiza las correcciones y preferencias del usuario
2. Las compara con memorias existentes
3. Si detecta un patrón nuevo con suficiente frecuencia (configurable, por defecto 3 ocurrencias)
4. Propone al usuario la creación de una nueva memoria
5. El usuario puede aceptar, editar o rechazar la propuesta

**Tareas de la Fase 1:**

| Tarea | Archivo(s) | Esfuerzo | Prioridad |
|-------|-----------|----------|-----------|
| Memory Manager core | `memory/memory.go` | Alto | P0 |
| File scanner + loader | `memory/scanner.go`, `loader.go` | Medio | P0 |
| Template integration | `memory/loader.go` + templates | Bajo | P0 |
| Relevance search | `memory/search.go` | Alto | P1 |
| Auto-generation | `memory/autogen.go` | Alto | P1 |
| Memory aging | `memory/age.go` | Bajo | P2 |
| Config options | `memory/config.go` | Bajo | P0 |

---

### 4.2 Fase 2: Sistema de Hooks (Prioridad: ALTA)

El sistema de hooks de Claude Code es uno de sus pilares de extensibilidad. Permite a los usuarios ejecutar código personalizado antes y después de eventos clave. Los hooks se definen en el archivo de configuración y pueden ejecutar comandos del shell, scripts, o comunicarse con el agente a través de stdin/stdout con un protocolo estructurado basado en JSONL.

#### 4.2.1 Tipos de Eventos Hook

| Evento | Descripción | Input | Output |
|--------|-------------|-------|--------|
| `PreToolUse` | Antes de ejecutar una herramienta | `tool_name, tool_input` | `approve/deny/modify` |
| `PostToolUse` | Después de ejecutar una herramienta | `tool_name, tool_input, output` | `feedback/modificación` |
| `SessionStart` | Al inicio de una nueva sesión | `session_id, cwd` | `mensaje inicial, contexto` |
| `FileChanged` | Cuando un archivo es modificado por el agente | `file_path, change_type` | `acciones post-cambio` |
| `Stop` | Cuando el agente finaliza una respuesta | `messages, summary` | `notificación, log` |
| `Notification` | Notificaciones del sistema | `type, message` | `acción personalizada` |

#### 4.2.2 Implementación en Go

En Go, el sistema de hooks se implementa en `internal/personal/hooks/` con una arquitectura basada en interfaces:

```go
// internal/personal/hooks/types.go

// Hook es la interfaz base que todos los hooks deben implementar
type Hook interface {
    // Name retorna el nombre identificador del hook
    Name() string
    // Match evalúa si este hook debe ejecutarse para el evento dado
    Match(event HookEvent) bool
    // Execute ejecuta el hook con el evento proporcionado
    Execute(ctx context.Context, event HookEvent) (*HookResult, error)
}

// HookEvent representa un evento que dispara hooks
type HookEvent struct {
    Type      HookEventType   // PreToolUse, PostToolUse, etc.
    ToolName  string          // Nombre de la herramienta (si aplica)
    ToolInput interface{}     // Input de la herramienta
    Output    interface{}     // Output de la herramienta (PostToolUse)
    SessionID string          // ID de la sesión
    FilePath  string          // Ruta del archivo (FileChanged)
    Message   string          // Mensaje (Notification)
}

// HookResult es el resultado de ejecutar un hook
type HookResult struct {
    Decision  string // "approve", "deny", "modify"
    Reason    string
    Modified  interface{} // Input modificado (si Decision == "modify")
    Message   string      // Mensaje para el usuario
}
```

**Tipos de hooks concretos:**

```go
// ShellHook ejecuta un comando del shell
type ShellHook struct {
    name    string
    command string
    pattern string // Pattern matching (ej: "Bash(git *)")
    timeout time.Duration
}

// ScriptHook ejecuta un script con el evento como argumento
type ScriptHook struct {
    name    string
    script  string
    pattern string
}

// AgentHook comunica con el agente via stdin/stdout en formato JSONL
type AgentHook struct {
    name    string
    command string
    pattern string
}
```

#### 4.2.3 Configuración de Hooks

Los hooks se configuran en `crush.json` bajo una nueva sección `"hooks"`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash(rm *)",
        "command": "echo '⚠️  About to delete files. Reviewing...'",
        "timeout": 5000
      }
    ],
    "PostToolUse": [
      {
        "matcher": "FileEdit",
        "command": "./scripts/format-check.sh"
      }
    ],
    "SessionStart": [
      {
        "command": "cat .crush/memory/CLAUDE.md"
      }
    ]
  }
}
```

| Archivo | Responsabilidad |
|---------|----------------|
| `manager.go` | HookManager: registro, ejecución, matching |
| `types.go` | Interface Hook + tipos concretos (Shell, Script, Agent) |
| `events.go` | Definición de eventos (PreToolUse, PostToolUse, etc.) |
| `config.go` | Configuración de hooks desde crush.json |
| `matcher.go` | Pattern matching de hooks contra eventos |

---

### 4.3 Fase 3: Sistema de Plugins (Prioridad: ALTA)

Claude Code tiene un sistema de plugins maduro con marketplace oficial, instalación versionada, hot-reload mediante file watchers, y una API para que los plugins registren herramientas, hooks, comandos, skills, y estilos de output.

#### 4.3.1 Arquitectura del Plugin Manager

Dado que Go no tiene soporte nativo para plugins dinámicos (a diferencia de TypeScript con `require/import` dinámico), se utilizan **dos estrategias complementarias**:

1. **Plugins Markdown**: Definen herramientas, hooks y comandos mediante declaración en YAML frontmatter + prompt templates (similar a las skills existentes de Crush pero más poderosas)
2. **Plugins binarios**: Se cargan mediante `hashicorp/go-plugin` usando RPC sobre `net/rpc`, permitiendo plugins en cualquier lenguaje

**Formato de plugin Markdown:**

```markdown
---
name: my-tool
version: 1.0.0
description: "Herramienta personalizada para análisis de código"
tools:
  - name: AnalyzeCode
    description: "Analiza la calidad del código"
    input_schema:
      type: object
      properties:
        path:
          type: string
          description: "Ruta del archivo a analizar"
hooks:
  - event: PostToolUse
    matcher: "FileWrite"
    command: "python3 lint.py {{.FilePath}}"
---

## Instrucciones

Cuando se te pida analizar código, sigue estas reglas:
1. Lee el archivo completo
2. Identifica problemas de estilo
3. Sugiere mejoras específicas
```

**Componentes del sistema de plugins:**

| Componente | Ubicación | Responsabilidad |
|-----------|-----------|----------------|
| PluginManager | `plugins/manager.go` | Ciclo de vida, registro, hot-reload |
| PluginLoader | `plugins/loader.go` | Carga y parseo de plugins Markdown/binary |
| PluginRegistry | `plugins/registry.go` | Almacenamiento y consulta de extensiones |
| PluginValidator | `plugins/validator.go` | Validación de esquemas y seguridad |
| PluginWatcher | `plugins/watcher.go` | File watcher para hot-reload |
| PluginConfig | `plugins/config.go` | Configuración de plugins por proyecto |

#### 4.3.2 Integración con Crush

El PluginManager se integra con los puntos de extensión de Crush:

- **Herramientas**: Los plugins pueden registrar nuevas herramientas que se agregan al set de herramientas del agente
- **Hooks**: Los plugins pueden registrar hooks que se ejecutan en los puntos del ciclo de vida
- **Commands**: Los plugins pueden agregar nuevos slash commands al REPL
- **Skills**: Los plugins pueden proporcionar skills que el agente puede descubrir y usar
- **System Prompt**: Los plugins pueden inyectar secciones adicionales al system prompt

---

### 4.4 Fase 4: Compresión Inteligente de Contexto (Prioridad: MEDIA)

Claude Code implementa un sistema de compresión de contexto sofisticado con múltiples niveles:

#### 4.4.1 Niveles de Compresión

| Nivel | Nombre | Trigger | Método | Latencia |
|-------|--------|---------|--------|----------|
| 1 | **Micro-compact** | Cada tool result | Reglas heurísticas (sin LLM) | < 10ms |
| 2 | **Session Memory Compact** | Antes de auto-compact | LLM extrae memorias persistentes | 2-5s |
| 3 | **Auto-compact** | ~80% contexto usado | LLM genera resumen | 3-8s |
| 4 | **Snip** | Contexto crítico | Recorte agresivo con boundary markers | < 50ms |

#### 4.4.2 Micro-Compact (Sin LLM)

El micro-compact es una optimización que no requiere llamadas al LLM y por lo tanto es extremadamente rápida. Funciona aplicando reglas heurísticas a los resultados de herramientas:

1. **Truncar logs repetitivos**: Conservar solo la primera y última ocurrencia con un contador intermedio
2. **Colapsar stack traces**: Mantener solo las primeras 5 y últimas 5 líneas
3. **Limitar outputs de búsqueda**: Mostrar los primeros N resultados con indicador de conteo
4. **Eliminar ANSI escapes**: Remover secuencias de control innecesarias
5. **Truncar líneas largas**: Cortar líneas individuales que excedan un umbral configurable

```go
// internal/personal/compact/micro.go

type MicroCompactRule func(content string) string

func TruncateRepetitiveLogs(maxRepeat int) MicroCompactRule { ... }
func CollapseStackTraces(headLines, tailLines int) MicroCompactRule { ... }
func LimitSearchResults(maxResults int) MicroCompactRule { ... }
func StripANSIEscapes() MicroCompactRule { ... }
func TruncateLongLines(maxLen int) MicroCompactRule { ... }
```

#### 4.4.3 Auto-Compact con Memoria

El auto-compact se activa cuando el porcentaje de contexto utilizado supera el umbral configurado (por defecto 80%). El flujo es:

1. **Evaluar urgencia**: Calcular porcentaje de contexto usado
2. **Session Memory Compact**: Extraer memorias persistentes antes de resumir
3. **Generar resumen**: Usar el small model del usuario para generar un resumen conciso
4. **Insertar resumen**: Reemplazar mensajes antiguos con un mensaje de sistema que contiene el resumen + metadata

```go
// internal/personal/compact/manager.go

type CompactManager struct {
    agent       *agent.SessionAgent
    memoryMgr   *memory.MemoryManager
    threshold   float64  // Default: 0.8
    smallModel  string
}

func (cm *CompactManager) Evaluate(messages []message.Message) CompactAction { ... }
func (cm *CompactManager) MicroCompact(content string) string { ... }
func (cm *CompactManager) MemoryCompact(messages []message.Message) ([]memory.Memory, error) { ... }
func (cm *CompactManager) AutoCompact(messages []message.Message) ([]message.Message, error) { ... }
```

---

### 4.5 Fase 5: Modo Plan y Tareas v2 (Prioridad: MEDIA)

#### 4.5.1 Modo Plan

El modo plan permite que el agente primero genere un plan detallado antes de ejecutar cualquier acción destructiva. El flujo es:

1. **EnterPlanMode**: El usuario o agente entra al modo plan
2. **Generación del plan**: El agente genera un plan estructurado en Markdown con:
   - Objetivo general
   - Pasos numerados con dependencias
   - Consideraciones y riesgos
   - Archivos afectados
3. **Revisión del usuario**: El usuario revisa, modifica o rechaza el plan
4. **ExitPlanMode**: El plan es aprobado y el agente lo ejecuta paso a paso

```go
// internal/personal/planmode/state.go

type PlanState struct {
    Active      bool
    Plan       *Plan
    CurrentStep int
    Approved   bool
}

type Plan struct {
    Goal         string
    Steps        []PlanStep
    Considerations []string
    AffectedFiles []string
    CreatedAt    time.Time
}

type PlanStep struct {
    Number      int
    Description string
    Status      PlanStepStatus // Pending, InProgress, Completed, Failed
    Dependencies []int
}
```

**Herramientas nuevas:** `EnterPlanMode` y `ExitPlanModeV2` que cambian el estado del agente.

**UI:** Un panel dedicado en Bubble Tea que muestra el plan con controles de aprobación/rechazo y progreso de cada paso.

#### 4.5.2 Tareas v2 (Task System)

Claude Code tiene un sistema de tareas v2 con CRUD completo que va más allá del simple `TodoWrite` de Crush:

| Herramienta | Descripción |
|-------------|-------------|
| `TaskCreate` | Crear nueva tarea con título, descripción, prioridad |
| `TaskGet` | Obtener detalle de una tarea por ID |
| `TaskUpdate` | Actualizar estado, título, descripción de una tarea |
| `TaskList` | Listar tareas con filtros (estado, prioridad) |

**Nueva tabla SQLite:**

```sql
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, in_progress, completed
    priority TEXT NOT NULL DEFAULT 'medium', -- low, medium, high
    parent_id TEXT REFERENCES tasks(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);
```

---

### 4.6 Fase 6: Extensiones Avanzadas (Prioridad: BAJA)

#### 4.6.1 Orquestación Multi-Agente (Coordinator Mode)

Claude Code implementa un modo coordinador donde un agente principal (coordinator) delega tareas a agentes trabajadores (workers) especializados. En Go, esto se implementa **naturalmente con goroutines y canales**:

```go
// internal/personal/coordinator/coordinator.go

type Coordinator struct {
    agents    map[string]*Worker
    taskQueue chan Task
    results   chan WorkerResult
    mu        sync.RWMutex
}

type Worker struct {
    ID       string
    Agent    *agent.SessionAgent
    Context  ToolUseContext
    Cancel   context.CancelFunc
}

func (c *Coordinator) SpawnWorker(task Task) (*Worker, error) {
    // Ejecutar como goroutine independiente
    go func() {
        result, err := w.Agent.Run(task.Prompt)
        c.results <- WorkerResult{WorkerID: w.ID, Result: result, Err: err}
    }()
}
```

**Backends de visualización:**

| Backend | Descripción | Complejidad |
|---------|-------------|-------------|
| In-process | Workers como goroutines, panel en TUI | Media |
| tmux | Cada worker en un pane separado | Alta |
| iTerm2 | Tabs nativas de iTerm2 | Alta (solo macOS) |

#### 4.6.2 Sistema de Cron

Herramientas para programar tareas periódicas: `CronCreate`, `CronDelete`, `CronList`. Se implementa utilizando `go-co-op/gocron` como scheduler, integrado con el sistema de tareas v2 y persistencia en SQLite.

```go
// internal/personal/cron/scheduler.go

type CronScheduler struct {
    scheduler *gocron.Scheduler
    db        *db.Queries
    agent     *agent.SessionAgent
}
```

#### 4.6.3 Tool Search y Diferimiento Lazy

Cuando el número de herramientas disponibles supera un umbral (incluyendo herramientas MCP y plugins), en lugar de enviar los schemas de todas al LLM, se envía solo nombres y descripciones breves, y se proporciona una herramienta `ToolSearch` para obtener el schema completo bajo demanda. Esto reduce dramáticamente el uso del contexto.

---

## 5. Nuevas Herramientas a Implementar

| Herramienta | Equivalente en CC | Descripción | Prioridad |
|-------------|-----------------|-------------|-----------|
| **AskUser** | `AskUserQuestionTool` | Permite al agente hacer preguntas interactivas al usuario durante la ejecución | P0 |
| **NotebookEdit** | `NotebookEditTool` | Edición de celdas en notebooks Jupyter (.ipynb) | P1 |
| **ConfigTool** | `ConfigTool` | Modificar configuración en runtime desde el agente | P1 |
| **TaskCreate** | `TaskCreateTool` | Crear tareas estructuradas con estados | P0 |
| **TaskUpdate** | `TaskUpdateTool` | Actualizar estado y contenido de tareas | P0 |
| **TaskGet** | `TaskGetTool` | Obtener detalle de una tarea | P0 |
| **TaskList** | `TaskListTool` | Listar tareas con filtros | P0 |
| **EnterPlanMode** | `EnterPlanModeTool` | Entrar al modo plan | P1 |
| **ExitPlanMode** | `ExitPlanModeV2Tool` | Salir del modo plan y ejecutar | P1 |
| **CronCreate** | `CronCreateTool` | Programar tarea periódica | P2 |
| **CronDelete** | `CronDeleteTool` | Eliminar tarea programada | P2 |
| **CronList** | `CronListTool` | Listar tareas programadas | P2 |
| **ToolSearch** | `ToolSearchTool` | Búsqueda lazy de schemas de herramientas | P2 |
| **SendMessage** | `SendMessageTool` | Enviar mensajes entre agentes (multi-agente) | P2 |
| **FileHistory** | N/A (builtin) | Historial de cambios por archivo con snapshots | P2 |

---

## 6. Mapa de Archivos y Directorios Propuestos

```
internal/personal/
├── init.go                        # Inicialización de todos los módulos personales
│
├── memory/                        # Sistema de memoria persistente
│   ├── memory.go                  # MemoryManager: Load, Save, Search, Delete
│   ├── scanner.go                 # Escaneo del filesystem para memorias relevantes
│   ├── loader.go                  # Carga de memorias al system prompt via templates
│   ├── search.go                  # Búsqueda por relevancia (TF-IDF / embeddings)
│   ├── autogen.go                 # Auto-generación de memorias semilla
│   ├── age.go                     # Envejecimiento y consolidación de memorias
│   └── config.go                  # Opciones de configuración
│
├── hooks/                         # Sistema de hooks pre/post ejecución
│   ├── manager.go                 # HookManager: registro, ejecución, matching
│   ├── types.go                   # Interface Hook + tipos (Shell, Script, Agent)
│   ├── events.go                  # Definición de eventos
│   ├── matcher.go                 # Pattern matching de hooks
│   └── config.go                  # Configuración desde crush.json
│
├── plugins/                       # Sistema de plugins con hot-reload
│   ├── manager.go                 # PluginManager: descubrimiento, carga, registro
│   ├── loader.go                  # Carga de plugins Markdown y binarios
│   ├── registry.go                # Registro de extensiones (tools, hooks, skills)
│   ├── watcher.go                 # File watcher para hot-reload
│   ├── validator.go               # Validación de esquemas y seguridad
│   └── config.go                  # Configuración de plugins
│
├── compact/                       # Compresión inteligente de contexto
│   ├── manager.go                 # CompactManager: evaluación y ejecución
│   ├── micro.go                   # Micro-compact: reglas heurísticas sin LLM
│   ├── auto.go                    # Auto-compact: resumen via LLM small model
│   ├── memory.go                  # Session memory compact: extracción de memorias
│   ├── snip.go                    # Snip: recorte agresivo con boundary markers
│   └── config.go                  # Umbrales y configuración
│
├── planmode/                      # Modo plan para razonamiento estructurado
│   ├── state.go                   # Estado del modo plan en SessionAgent
│   ├── tools.go                   # EnterPlanMode + ExitPlanMode tools
│   ├── plan.go                    # Struct Plan + PlanStep
│   └── renderer.go                # Renderizado del plan en la UI
│
├── tasks/                         # Sistema de tareas v2
│   ├── service.go                 # TaskService: CRUD con SQLite
│   ├── models.go                  # Task, TaskStatus, TaskPriority
│   ├── tools.go                   # TaskCreate, TaskGet, TaskUpdate, TaskList
│   └── migrations/                # SQL migration para tabla tasks
│       └── 20260401_add_tasks.sql
│
├── coordinator/                   # Orquestación multi-agente
│   ├── coordinator.go             # Coordinator: delegación a workers via goroutines
│   ├── worker.go                  # Worker: agente especializado con contexto propio
│   └── backend.go                 # Backend de visualización (in-process, tmux)
│
├── cron/                          # Sistema de programación de tareas
│   ├── scheduler.go               # Scheduler basado en gocron
│   ├── tools.go                   # CronCreate, CronDelete, CronList
│   ├── models.go                  # CronJob, CronSchedule
│   └── persistence.go             # Persistencia en SQLite
│
└── tools/                         # Nuevas herramientas individuales
    ├── ask_user.go                # AskUser tool
    ├── notebook_edit.go           # NotebookEdit tool
    ├── config_tool.go             # ConfigTool
    ├── tool_search.go             # ToolSearch tool
    ├── send_message.go            # SendMessage tool
    └── file_history.go            # FileHistory tool
```

---

## 7. Decisiones Técnicas Clave

### 7.1 Dependencias Go Seleccionadas

| Dependencia | Propósito | Alternativas |
|-------------|-----------|--------------|
| `hashicorp/go-plugin` | Plugins binarios con RPC seguro | `so` (librería std), `krakend` (HTTP) |
| `go-co-op/gocron` | Scheduler de tareas periódicas | `robfig/cron`, `go-co-op/gocron/v2` |
| `fsnotify/fsnotify` | File watcher para hot-reload | Ya es dependencia de Bubble Tea |
| `yuin/goldmark` | Parseo de Markdown para plugins/memorias | Ya es dependencia de Glamour |
| `pandora101/go-tfidf` | TF-IDF para búsqueda de memorias | Implementación propia, embeddings ONNX |
| `onnxruntime-go` | Inferencia local de embeddings (opcional) | `go-tfidf` como alternativa ligera |

**Criterios de selección:**
- Solo dependencias puras en Go (sin CGO) para mantener compilación estática
- Licencia compatible (MIT, Apache-2.0, BSD)
- Tamaño mínimo del binario resultante
- Actividad de la comunidad y madurez del proyecto

### 7.2 Estrategia de Testing

El testing sigue los patrones existentes de Crush, con adiciones para los nuevos módulos:

| Tipo de Test | Herramienta | Aplica a |
|--------------|------------|----------|
| Unitarios | `testing` + `testify/require` | Todos los módulos |
| Integración | SQLite en memoria | db, plugins, tasks, cron |
| VCR | Grabación/reproducción LLM | agent con nuevas herramientas |
| Golden-file | Archivos `.golden` | Componentes de UI |
| Contrato | Interface verification | Plugin system |
| Regresión | Simulación de eventos | Hook system |
| Benchmarks | `testing.B` | Compact (reducción de tokens, latencia) |
| Concurrencia | `go test -race` | Coordinator mode |

### 7.3 Compatibilidad con Actualizaciones de Upstream

Para mantener la compatibilidad con las actualizaciones de Crush (charmbracelet/crush):

1. **Aislamiento total**: Todos los cambios personales en `internal/personal/` sin modificar archivos del core
2. **Interfaces públicas**: Extensiones solo a través de interfaces públicas y puntos de extensión existentes (pubsub, tool registration, config)
3. **Tests de integración**: Prueban que el fork funciona correctamente con el core de Crush sin modificaciones
4. **Documentación de puntos de extensión**: Cada punto de extensión utilizado se documenta para facilitar resolución de conflictos
5. **Revisión de PRs de upstream**: Anticipar cambios que puedan afectar las extensiones personales

En caso de conflicto durante rebase, se prioriza mantener la funcionalidad del fork adaptando los módulos personales al nuevo API de Crush.

---

## 8. Cronograma Estimado

| Fase | Descripción | Duración | Semanas |
|------|-------------|----------|---------|
| **Fase 1** | Sistema de Memoria Persistente | 3-4 semanas | 1-4 |
| **Fase 2** | Sistema de Hooks | 2-3 semanas | 3-5 |
| **Fase 3** | Sistema de Plugins | 3-4 semanas | 5-8 |
| **Fase 4** | Compresión Inteligente de Contexto | 2-3 semanas | 7-9 |
| **Fase 5** | Modo Plan + Tareas v2 | 2-3 semanas | 9-11 |
| **Fase 6** | Extensiones Avanzadas | 4-6 semanas | 11-16 |
| **Testing + Docs** | Testing integral y documentación | 2 semanas | 14-16 |

> **Nota:** Las fases se solapan parcialmente cuando hay dependencias parciales, permitiendo avanzar en paralelo. Estimación asumiendo dedicación parcial de ~15-20 horas semanales.

**Primer entregable funcional:** Fase 1 (Sistema de Memoria) en ~3-4 semanas. Proporciona valor inmediato al usuario y valida la estrategia de aislamiento.

---

## 9. Riesgos y Mitigaciones

| Riesgo | Impacto | Probabilidad | Mitigación |
|--------|---------|-------------|------------|
| Conflictos con upstream | Alto | Media | Aislamiento en `internal/personal/`, tests de integración, revisión de PRs |
| Performance de compact | Medio | Baja | Micro-compact sin LLM, async compact en goroutine separada |
| Seguridad de plugins | Alto | Media | Sandboxing, validación de esquemas, permisos granulares |
| Complejidad del coordinator | Medio | Alta | Fase 6 (baja prioridad), implementación incremental, tests de concurrencia |
| Dependencias adicionales | Bajo | Baja | Solo deps puras Go, evaluación de tamaño de binario |
| Cambios breaking en Crush | Alto | Baja | Suscripción a releases, CI que prueba fork contra upstream |

---

## 10. Conclusión y Próximos Pasos

Este plan establece una ruta clara para transformar Crush de un excelente asistente de codificación AI de terminal a una **plataforma extensible y personalizable** que incorpora las mejores ideas de Claude Code. La estrategia de fork con módulos aislados garantiza que las mejoras personales convivan armónicamente con las actualizaciones de Crush, mientras que la implementación en Go idiomático asegura rendimiento, seguridad y facilidad de distribución como binario estático.

### Próximos Pasos Inmediatos

1. **Configurar el fork**: Agregar remote upstream y crear la estructura `internal/personal/`
2. **Implementar Fase 1** (Sistema de Memoria) como primer entregable funcional
3. **Establecer CI**: Pipeline que compile el fork y ejecute tests contra el último commit de upstream

El resultado final será un asistente de codificación AI que combina lo mejor de ambos mundos: la solidez multi-proveedor y el rendimiento de Go de Crush, con la sofisticación en gestión de contexto, extensionibilidad y orquestación de Claude Code.
