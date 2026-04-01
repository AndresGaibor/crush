# Sistema de Memoria: Funcionamiento Automático

## ¿Cómo Funciona?

El sistema de memoria en Crush **es completamente automático** en dos niveles:

### 1. **Inyección Automática en System Prompt** ✅

Cuando Crush inicia una sesión:

```
┌─────────────────────────────────────────┐
│ Usuario abre Crush en directorio        │
└──────────────────┬──────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│ Memory system inicializa                │
│ - .crush/memory/ escanea                │
│ - ~/.config/crush/memory/ escanea       │
└──────────────────┬──────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│ System prompt se genera:                │
│ - Archivos en <memory> inyectados       │
│ - Instrucciones de uso agregadas        │
│ - LLM recibe toda la información        │
└──────────────────┬──────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│ LLM ve:                                  │
│ ✓ Todas las memorias guardadas          │
│ ✓ Instrucciones para usar memoria tool  │
│ ✓ Contexto del proyecto                 │
└─────────────────────────────────────────┘
```

**Código relevante:**
- `internal/agent/prompt/prompt.go:168-176` — Inyecta `.crush/memory/`
- `internal/agent/templates/coder.md.tpl:385-404` — Template con instrucciones

### 2. **LLM Decide Cuándo Usar la Herramienta** ✅

Una vez que Crush inicia, el LLM ve:

1. **Las memorias cargadas** en la sección `<memory>`
   ```markdown
   <memory>
   <file path=".crush/memory/project-style.md">
   ---
   tags: [style, conventions]
   ---
   # Project Style
   - Use tabs for indentation
   - Follow Go conventions
   </file>
   </memory>
   ```

2. **Las instrucciones de uso**
   ```markdown
   <memory_instructions>
   You have access to a **memory** tool...
   Use it to:
   1. Save project patterns
   2. Search previous decisions
   3. Load context
   4. Check what's saved
   </memory_instructions>
   ```

3. **La herramienta disponible** en `AllowedTools`
   - La herramienta "memory" está registrada y disponible

**El flujo automático es:**

```
Sesión 1: Usuario enseña patrón
┌──────────────────────────────────────────┐
│ Usuario: "Usa siempre tabs en los tests" │
└─────────────────┬──────────────────────┘
                  ↓
         LLM VE las instrucciones de memoria
         y decide guardar automáticamente:
┌──────────────────────────────────────────┐
│ memory.save(id: "test-indentation",      │
│            content: "...",               │
│            tags: ["convention", "style"])│
└─────────────────┬──────────────────────┘
                  ↓
         Memory guardada en disco
         .crush/memory/test-indentation.md

─────────────────────────────────────────

Sesión 2: LLM accede a la memoria
┌──────────────────────────────────────────┐
│ Sistema prompt inyecta automáticamente:  │
│ <memory>                                 │
│ <file path=".crush/memory/...">          │
│ Test indentation: use tabs...            │
│ </file>                                  │
│ </memory>                                │
└─────────────────┬──────────────────────┘
                  ↓
         LLM respeta automáticamente
         la convención guardada
```

## Tres Niveles de Automatización

### Nivel 1: Carga Automática ✅
- `.crush/memory/` se escanea automáticamente
- Archivos se inyectan en system prompt
- **Sin configuración necesaria**

### Nivel 2: Herramienta Disponible ✅
- `memory` está en `AllowedTools`
- LLM puede invocarla cuando lo crea conveniente
- **Sin necesidad de ask user permission**

### Nivel 3: Instrucciones al LLM ✅
- Template explica cuándo usar `memory`
- LLM recibe contexto sobre:
  - Qué memorias ya existen
  - Cómo guardar nuevas memorias
  - Cómo buscar contexto previo
- **LLM decide proactivamente**

## Ejemplo Completo de Flujo

```
SESIÓN 1 - Hora 10:00
─────────────────────

Usuario: "Cuando hagas tests en Go, siempre usa t.Parallel()"

┌─ LLM procesa:
│  - Ve las instrucciones de memoria
│  - Reconoce una preferencia importante
│  - Decide guardarla automáticamente
│
└─ Invoca: memory.save(
     id: "go-parallel-tests",
     content: "Always use t.Parallel() for Go tests",
     tags: ["go", "testing", "convention"],
     scope: "project"
   )

   → Archivo creado: .crush/memory/go-parallel-tests.md

SESIÓN 2 - Hora 14:00 (mismo día o después)
──────────────────────────────────────────

Usuario abre Crush en mismo directorio

┌─ Sistema carga automáticamente:
│  - Escanea .crush/memory/
│  - Encuentra go-parallel-tests.md
│  - Lo inyecta en system prompt
│
└─ LLM ve en el prompt:
   <memory>
   <file path=".crush/memory/go-parallel-tests.md">
   Always use t.Parallel() for Go tests...
   </file>
   </memory>

Usuario: "Escribe tests para la función GetUser()"

┌─ LLM piensa:
│  - "Necesito escribir tests en Go"
│  - "El sistema prompt menciona go-parallel-tests.md"
│  - "Debo aplicar t.Parallel() automáticamente"
│
└─ LLM escribe tests CON t.Parallel()
   Sin que el usuario lo tenga que pedir de nuevo
```

## Casos de Uso Automáticos

### 1. **Aprender de Correcciones**
```
Sesión N: Usuario corrige el código
Usuario: "Usa gin, no echo"
  ↓
LLM: "Debo guardar esto"
  ↓
memory.save(id: "frameworks", content: "Use gin for routing")

Sesión N+1: LLM sabe usar gin automáticamente
```

### 2. **Buscar Contexto Previo**
```
Usuario: "Cómo manejamos los errores en este proyecto?"

LLM piensa:
- Voy a buscar en memoria sobre manejo de errores
- memory.search(query: "error handling")

Retorna memorias relevantes
LLM responde con contexto histórico
```

### 3. **Aplicar Preferencias Consistentemente**
```
Memoria guardada: "Always use slog for logging"

Cada vez que LLM escribe código:
- Ve la memoria inyectada
- Aplica slog automáticamente
- Sin que el usuario lo pida
```

## Verificación: ¿Funciona sin Manual?

**SÍ.** El sistema es completamente automático:

✅ **Sin CLI** - No necesitas comandos para activar
✅ **Sin Configuración** - .crush/memory/ se crea automáticamente
✅ **Sin Pedir Permiso** - LLM usa la herramienta cuando lo cree necesario
✅ **Sin User Intervention** - Crush carga memorias sin preguntar
✅ **Sin Prompt Especial** - Memorias se inyectan en todo prompt

## Flujo Visual: De Inicio a Fin

```
START: Crush abre en proyecto
  │
  ├─→ [memory.Init()] carga memoria system
  │
  ├─→ [prompt.promptData()] inyecta:
  │   ├─ CLAUDE.md (si existe)
  │   ├─ .crush/memory/* (automático)
  │   └─ other context_paths
  │
  ├─→ [coder.md.tpl] genera system prompt con:
  │   ├─ <memory> section (archivos inyectados)
  │   └─ <memory_instructions> (guía de uso)
  │
  ├─→ LLM recibe prompt completo
  │   ├─ Ve todas las memorias
  │   ├─ Lee instrucciones de uso
  │   ├─ Tiene "memory" en AllowedTools
  │   └─ Decide automáticamente cuándo usarla
  │
  ├─→ Usuario pide tarea
  │   └─ LLM aplica memorias + herramienta
  │
  └─→ Memorias persistentes entre sesiones ✓
```

## Comandos Para Verificar

```bash
# Ver memorias cargadas
ls -la .crush/memory/
ls -la ~/.config/crush/memory/

# Ver contenido
cat .crush/memory/project-style.md

# Verificar que se inyectan
# (Solo visible internamente en Crush)
# Crush carga las memorias en el system prompt

# Usar la herramienta manualmente (si necesario)
# En Crush: "Usa memory.search(query: 'conventions')"
```

## Resumen

**La memoria es automática porque:**

1. ✅ Se carga automáticamente en el system prompt
2. ✅ Se inyecta en la sección `<memory>`
3. ✅ Se agregan instrucciones explícitas `<memory_instructions>`
4. ✅ La herramienta está disponible en `AllowedTools`
5. ✅ El LLM decide cuándo usarla basado en el contexto

**No necesitas hacer nada especial. El sistema de memoria es completamente transparente y automático.**
