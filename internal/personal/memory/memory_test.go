package memory

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnv crea un entorno temporal con directorios de memoria.
func setupTestEnv(t *testing.T) (*MemoryManager, string, func()) {
	t.Helper()
	tmpDir := t.TempDir()

	// Crear estructura de directorios esperada
	projectMemDir := filepath.Join(tmpDir, ".crush", "memory")
	globalMemDir := filepath.Join(tmpDir, ".config", "crush", "memory")
	require.NoError(t, os.MkdirAll(projectMemDir, 0o755))
	require.NoError(t, os.MkdirAll(globalMemDir, 0o755))

	mgr := &MemoryManager{
		projectDir:    tmpDir,
		projectMemDir: projectMemDir,
		globalMemDir:  globalMemDir,
	}

	// NO resetear singletons en tests paralelos - causa race conditions
	cleanup := func() {
		// Cleanup se usa para limpiar recursos del test si es necesario
	}

	return mgr, tmpDir, cleanup
}

// --- Tests de MemoryManager ---

func TestNewMemoryManager(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mgr, err := NewMemoryManager(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, mgr)
	assert.Equal(t, tmpDir, mgr.projectDir)
	assert.DirExists(t, mgr.projectMemDir)
	assert.DirExists(t, mgr.globalMemDir)
}

func TestSaveAndLoadMemory(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// Save
	mem, err := mgr.Save("test-memory", "# Test\nHello world", ScopeProject, []string{"test"})
	require.NoError(t, err)
	assert.Equal(t, "test-memory", mem.ID)
	assert.Equal(t, ScopeProject, mem.Scope)
	assert.Equal(t, []string{"test"}, mem.Tags)
	assert.NotEmpty(t, mem.Path)

	// Verificar archivo existe en disco
	content, err := os.ReadFile(mem.Path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Test")
	assert.Contains(t, string(content), "Hello world")

	// Load
	loaded, err := mgr.Load("test-memory")
	require.NoError(t, err)
	assert.Equal(t, mem.ID, loaded.ID)
	assert.Equal(t, mem.Content, loaded.Content)
}

func TestSaveGlobalMemory(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	mem, err := mgr.Save("global-pref", "Siempre usa tabs", ScopeGlobal, []string{"preference"})
	require.NoError(t, err)
	assert.Equal(t, ScopeGlobal, mem.Scope)
	assert.Contains(t, mem.Path, "memory") // debería estar en globalMemDir
}

func TestLoadNonExistentMemory(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, err := mgr.Load("no-existe")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteMemory(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// Crear
	_, err := mgr.Save("borrame", "contenido temporal", ScopeProject, nil)
	require.NoError(t, err)

	// Verificar que existe
	_, err = mgr.Load("borrame")
	require.NoError(t, err)

	// Eliminar
	err = mgr.Delete("borrame")
	require.NoError(t, err)

	// Verificar que ya no existe
	_, err = mgr.Load("borrame")
	assert.Error(t, err)
}

func TestDeleteNonExistentMemory(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	err := mgr.Delete("no-existe")
	assert.Error(t, err)
}

func TestAllMemories(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// Crear memorias en distintos scopes
	_, err := mgr.Save("proj-1", "Proyecto 1", ScopeProject, nil)
	require.NoError(t, err)
	_, err = mgr.Save("proj-2", "Proyecto 2", ScopeProject, nil)
	require.NoError(t, err)
	_, err = mgr.Save("glob-1", "Global 1", ScopeGlobal, nil)
	require.NoError(t, err)

	all, err := mgr.All()
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestProjectOnlyMemories(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _ = mgr.Save("proj-a", "Solo proyecto", ScopeProject, nil)
	_, _ = mgr.Save("glob-b", "Solo global", ScopeGlobal, nil)

	proj, err := mgr.Project()
	require.NoError(t, err)
	assert.Len(t, proj, 1)
	assert.Equal(t, "proj-a", proj[0].ID)
}

func TestGlobalOnlyMemories(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _ = mgr.Save("proj-a", "Solo proyecto", ScopeProject, nil)
	_, _ = mgr.Save("glob-b", "Solo global", ScopeGlobal, nil)

	glob, err := mgr.Global()
	require.NoError(t, err)
	assert.Len(t, glob, 1)
	assert.Equal(t, "glob-b", glob[0].ID)
}

func TestSaveEmptyID(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, err := mgr.Save("", "contenido", ScopeProject, nil)
	assert.Error(t, err)
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"con espacios", "con-espacios"},
		{"MAYUSCULAS", "mayusculas"},
		{"especial!@#", "especial---"},
		{"mi-memory-123", "mi-memory-123"},
		{"a b c", "a-b-c"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, sanitizeFilename(tt.input))
	}
}

func TestExtractTags(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Archivo CON frontmatter
	path1 := filepath.Join(tmpDir, "with-tags.md")
	err := os.WriteFile(path1, []byte("---\ntags: [test, example]\n---\n\n# Content\n"), 0o644)
	require.NoError(t, err)
	tags := extractTags(path1)
	assert.Equal(t, []string{"test", "example"}, tags)

	// Archivo SIN frontmatter
	path2 := filepath.Join(tmpDir, "no-tags.md")
	err = os.WriteFile(path2, []byte("# Content without frontmatter\n"), 0o644)
	require.NoError(t, err)
	tags = extractTags(path2)
	assert.Nil(t, tags)
}

// --- Tests de Scanner ---

func TestFindRelevant(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _ = mgr.Save("go-conventions", "## Go\nUsa gofmt, no uses tabs", ScopeProject, []string{"go", "conventions"})
	_, _ = mgr.Save("python-tips", "## Python\nUsa black formatter", ScopeProject, []string{"python"})
	_, _ = mgr.Save("general-notes", "Algunas notas generales sobre coding style", ScopeGlobal, []string{"general"})

	scanner := NewScanner(mgr)
	results, err := scanner.FindRelevant("go conventions gofmt", 3)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	// El primero debería ser go-conventions (tags match + content match)
	assert.Equal(t, "go-conventions", results[0].ID)
}

func TestFindRelevantEmptyQuery(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _ = mgr.Save("mem1", "contenido 1", ScopeProject, nil)

	scanner := NewScanner(mgr)
	results, err := scanner.FindRelevant("", 0)
	require.NoError(t, err)
	assert.Len(t, results, 1) // retorna todas si query vacío
}

func TestFindByTag(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _ = mgr.Save("tagged-1", "contenido", ScopeProject, []string{"go", "backend"})
	_, _ = mgr.Save("tagged-2", "contenido", ScopeGlobal, []string{"go", "frontend"})
	_, _ = mgr.Save("untagged", "contenido", ScopeProject, nil)

	scanner := NewScanner(mgr)
	results, err := scanner.FindByTag("go")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

// --- Tests de Ager ---

func TestStaleDetection(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// Crear una memoria y hacerla "vieja" artificialmente
	mem, err := mgr.Save("old-memory", "antiguo", ScopeProject, nil)
	require.NoError(t, err)

	// Poner la fecha de modificación a hace 100 días
	oldTime := time.Now().Add(-100 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(mem.Path, oldTime, oldTime))

	ager := NewAger(mgr, 30*24*time.Hour, false)
	stale, err := ager.Stale()
	require.NoError(t, err)
	require.NotEmpty(t, stale)
	assert.Equal(t, "old-memory", stale[0].ID)
}

func TestStats(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _ = mgr.Save("p1", "project", ScopeProject, nil)
	_, _ = mgr.Save("p2", "project", ScopeProject, nil)
	_, _ = mgr.Save("g1", "global", ScopeGlobal, nil)

	ager := NewAger(mgr, 30*24*time.Hour, false)
	stats := ager.Stats()
	assert.Equal(t, 3, stats.Total)
	assert.Equal(t, 2, stats.Project)
	assert.Equal(t, 1, stats.Global)
}

// --- Tests de Tool Execute ---

func TestToolSaveAndList(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	detector := NewPatternDetector(3)

	// Save
	out, err := Execute(mgr, detector, ToolInput{
		Action:  "save",
		ID:      "my-preferences",
		Content: "# Preferences\n- Usar tabs\n- Español",
		Scope:   "project",
		Tags:    []string{"preferences"},
	})
	require.NoError(t, err)
	assert.Contains(t, out.Result, "Memory saved")
	assert.Contains(t, out.Result, "my-preferences")

	// List
	out, err = Execute(mgr, detector, ToolInput{Action: "list"})
	require.NoError(t, err)
	assert.Contains(t, out.Result, "my-preferences")
}

func TestToolLoad(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _ = mgr.Save("load-test", "Contenido de prueba", ScopeProject, nil)

	out, err := Execute(mgr, nil, ToolInput{Action: "load", ID: "load-test"})
	require.NoError(t, err)
	assert.Contains(t, out.Result, "Contenido de prueba")
}

func TestToolSearch(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _ = mgr.Save("search-test", "## Testing\nComo hacer tests en Go", ScopeProject, []string{"testing", "go"})

	out, err := Execute(mgr, nil, ToolInput{Action: "search", Query: "testing go"})
	require.NoError(t, err)
	assert.Contains(t, out.Result, "search-test")
}

func TestToolDelete(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _ = mgr.Save("delete-me", "borrar", ScopeProject, nil)

	out, err := Execute(mgr, nil, ToolInput{Action: "delete", ID: "delete-me"})
	require.NoError(t, err)
	assert.Contains(t, out.Result, "deleted")
}

func TestToolStats(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _ = mgr.Save("s1", "stat1", ScopeProject, nil)

	out, err := Execute(mgr, nil, ToolInput{Action: "stats"})
	require.NoError(t, err)
	assert.Contains(t, out.Result, "Total: 1")
}

func TestToolUnknownAction(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	out, err := Execute(mgr, nil, ToolInput{Action: "inventado"})
	require.NoError(t, err) // No error, pero resultado con mensaje de error
	assert.Contains(t, out.Result, "Unknown action")
}

func TestToolSaveMissingID(t *testing.T) {
	t.Parallel()
	mgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	out, err := Execute(mgr, nil, ToolInput{
		Action:  "save",
		Content: "sin ID",
	})
	require.NoError(t, err)
	assert.Contains(t, out.Result, "Error")
}

// --- Tests de PatternDetector ---

func TestPatternDetection(t *testing.T) {
	t.Parallel()
	detector := NewPatternDetector(3)

	// Observar 2 veces (no debe sugerir aún)
	detector.Observe("correction", "usa gin en vez de echo")
	detector.Observe("correction", "usa gin en vez de echo")
	suggestions := detector.CheckSuggestions()
	assert.Empty(t, suggestions)

	// Tercera vez (debe sugerir)
	detector.Observe("correction", "usa gin en vez de echo")
	suggestions = detector.CheckSuggestions()
	assert.Len(t, suggestions, 1)
	assert.Equal(t, 3, suggestions[0].Count)
}

func TestExtractCorrectionsFromDiff(t *testing.T) {
	text := `El usuario dijo: "usa tabs en vez de espacios" y luego "evita usar echo, usa gin"`
	corrections := ExtractCorrectionsFromDiff(text)
	assert.NotEmpty(t, corrections)
	// Debería detectar al menos "tabs en vez de espacios"
}

// --- Tests de Init singleton ---

func TestInitSingleton(t *testing.T) {
	// NO usar t.Parallel() porque manipula estado global (singleton)
	t.Setenv("CRUSH_TESTING", "1")
	tmpDir := t.TempDir()

	// Guardar estado original
	origInstance := instance
	origOnce := instanceOnce
	origScanner := scanner
	origDetector := detector

	// Defer restaurar estado original
	defer func() {
		instance = origInstance
		instanceOnce = origOnce
		scanner = origScanner
		detector = origDetector
	}()

	// Reset singleton para el test
	instance = nil
	instanceOnce = sync.Once{}
	scanner = nil
	detector = nil

	mgr1, err := Init(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, mgr1)

	mgr2 := GetManager()
	assert.Equal(t, mgr1, mgr2, "GetManager should return same instance as Init")

	// Init again should not error and should return same instance
	mgr3, err := Init("/otro/path")
	require.NoError(t, err)
	assert.Equal(t, mgr1, mgr3, "Second Init should return same instance")
}
