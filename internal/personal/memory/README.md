# Sistema de Memoria Persistente - Fase 1

Sistema de memoria persistente para Crush que permite guardar y recuperar información entre sesiones.

## Estructura Implementada

```
internal/personal/memory/
├── memory.go          # MemoryManager: save, load, delete, list
├── scanner.go         # Búsqueda por relevancia y filtrado
├── autogen.go         # Detección de patrones (preparado para fase 2)
├── aging.go           # Envejecimiento y limpieza de memorias antiguas
├── tool.go            # Helper conceptual; la tool real vive en internal/agent/tools/memory_tool.go
├── tool.md            # Documentación de la herramienta
└── init.go            # Inicialización y singletons

internal/agent/tools/
├── memory_tool.go     # Integración con el sistema de herramientas de Crush
└── memory.md          # Descripción de la herramienta (copia de tool.md)
```

## Integraciones con el Core de Crush

### 1. Registro de herramienta (internal/agent/coordinator.go:459)
```go
tools.NewMemoryTool(c.cfg.WorkingDir()),
```

### 2. Inyección automática en system prompt (internal/agent/prompt/prompt.go:168-175)
```go
// Inject .crush/memory/ directory if it exists
memoryPath := filepath.Join(store.WorkingDir(), ".crush", "memory")
if info, err := os.Stat(memoryPath); err == nil && info.IsDir() {
    content := processContextPath(memoryPath, store)
    files[memoryPathKey] = content
}
```

### 3. Inicialización al arranque (internal/app/app.go:111-117)
```go
// Initialize memory system
go func() {
    if _, err := memory.Init(store.WorkingDir()); err != nil {
        slog.Warn("Failed to initialize memory system", "error", err)
    }
}()
```

## Uso

### Directorios de Memoria

- **Proyecto**: `.crush/memory/` (memorias específicas del proyecto)
- **Global**: `~/.config/crush/memory/` (memorias compartidas entre proyectos)

### Comandos de la Herramienta

#### Guardar una memoria
```
action: save
id: project-conventions
content: |
  ## Project Conventions
  - Use tabs for indentation
  - Follow REST naming conventions
scope: project
tags: [conventions, style]
```

#### Listar todas las memorias
```
action: list
```

#### Buscar memorias por query
```
action: search
query: conventions coding style
```

#### Cargar una memoria específica
```
action: load
id: project-conventions
```

#### Ver estadísticas
```
action: stats
```

#### Sugerencias automáticas (preparado para fase 2)
```
action: suggest
```

### Eliminar una memoria
```
action: delete
id: project-conventions
```

## Características Implementadas

### ✅ Fase 1 Completa

1. **Sistema de almacenamiento jerárquico**
   - Memorias de proyecto (`.crush/memory/`)
   - Memorias globales (`~/.config/crush/memory/`)
   
2. **Herramienta `memory` para el agente**
   - Guardar, cargar, eliminar memorias
   - Listar y buscar por relevancia
   - Estadísticas del sistema

3. **Inyección automática en system prompt**
   - Las memorias en `.crush/memory/` se cargan automáticamente
   - Sin necesidad de configurar `context_paths`

4. **Búsqueda por relevancia**
   - Por tags (peso alto)
   - Por nombre/ID (peso medio)
   - Por contenido (peso bajo)
   - Prioridad: proyecto > global, reciente > antiguo

5. **Envejecimiento y limpieza**
   - Detecta memorias antiguas (>90 días)
   - Limpia automáticamente memorias muy antiguas (>180 días)

6. **Detección de patrones (preparado)**
   - Infraestructura lista para fase 2
   - Observación de correcciones del usuario
   - Generación automática de sugerencias

7. **Visibilidad en la UI**
   - Resumen de memorias activas en el sidebar
   - Memorias recientes con scope y tags

## Pruebas

### Crear memorias de prueba

```bash
# Memoria de proyecto
cat > .crush/memory/project-style.md << 'EOF'
---
tags: [style, conventions, go]
---
# Project Style Guide

- Use tabs for indentation
- Follow Go conventions
- Use gofumpt for formatting
EOF

# Memoria global
cat > ~/.config/crush/memory/global-preferences.md << 'EOF'
---
tags: [preferences, global]
---
# Global Preferences

- Always respond in Spanish
- Prefer Go for new services
- Use structured logging (slog)
EOF
```

### Probar en Crush

```bash
./crush

# Dentro de Crush:
# "Lista mis memorias"
# "Busca memorias sobre estilo"
# "Guarda una memoria: este proyecto usa PostgreSQL 16"
# "Muestra estadísticas de memorias"
```

## Formato de Memorias

Las memorias son archivos Markdown con frontmatter YAML opcional:

```markdown
---
tags: [category1, category2]
---
# Título

Contenido de la memoria en Markdown.

## Secciones

- Item 1
- Item 2
```

## Próximos Pasos (Fase 2)

- [ ] UI en el sidebar para mostrar memorias activas
- [ ] Auto-detección activa de patrones en ediciones
- [ ] Sugerencias inteligentes basadas en patrones
- [ ] Consolidación automática de memorias redundantes
- [ ] Embeddings para búsqueda semántica

## Notas de Implementación

- **No modifica el core de Crush**: todas las funcionalidades están en `internal/personal/`
- **Integraciones mínimas**: solo 3 cambios pequeños en archivos core
- **Sin dependencias externas**: usa solo la stdlib de Go
- **Thread-safe**: usa mutexes para operaciones concurrentes
- **Lazy initialization**: el sistema se inicializa solo cuando se usa

## Licencia

Este código sigue la licencia del proyecto Crush principal.
