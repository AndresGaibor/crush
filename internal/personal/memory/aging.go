package memory

import (
	"log/slog"
	"os"
	"time"
)

// Ager gestiona el envejecimiento y consolidación de memorias antiguas.
type Ager struct {
	manager     *MemoryManager
	maxAge      time.Duration // Edad máxima antes de marcar para revisión
	consolidate bool          // Si debe consolidar memorias pequeñas
}

// NewAger crea un nuevo gestor de envejecimiento.
func NewAger(manager *MemoryManager, maxAge time.Duration, consolidate bool) *Ager {
	return &Ager{
		manager:     manager,
		maxAge:      maxAge,
		consolidate: consolidate,
	}
}

// Stale retorna memorias que no han sido actualizadas en más de maxAge.
func (a *Ager) Stale() ([]Memory, error) {
	memories, err := a.manager.All()
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-a.maxAge)
	var stale []Memory
	for _, mem := range memories {
		if mem.UpdatedAt.Before(cutoff) {
			stale = append(stale, mem)
		}
	}
	return stale, nil
}

// Touch actualiza la fecha de modificación de una memoria (marca como "vigente").
func (a *Ager) Touch(id string) error {
	mem, err := a.manager.Load(id)
	if err != nil {
		return err
	}
	// Toquear el archivo actualiza mtime
	now := time.Now()
	if err := os.Chtimes(mem.Path, now, now); err != nil {
		return err
	}
	slog.Info("Memory touched", "id", id)
	return nil
}

// Clean elimina memorias que superan el doble de maxAge.
func (a *Ager) Clean() (int, error) {
	stale, err := a.Stale()
	if err != nil {
		return 0, err
	}

	threshold := time.Now().Add(-a.maxAge * 2)
	var cleaned int
	for _, mem := range stale {
		if mem.UpdatedAt.Before(threshold) {
			if err := a.manager.Delete(mem.ID); err != nil {
				slog.Warn("Failed to delete stale memory", "id", mem.ID, "error", err)
				continue
			}
			cleaned++
			slog.Info("Cleaned stale memory", "id", mem.ID, "age", time.Since(mem.UpdatedAt))
		}
	}
	return cleaned, nil
}

// Stats retorna estadísticas de las memorias.
func (a *Ager) Stats() MemoryStats {
	memories, err := a.manager.All()
	if err != nil {
		return MemoryStats{}
	}

	var stats MemoryStats
	stats.Total = len(memories)
	now := time.Now()
	cutoff := now.Add(-a.maxAge)

	for _, mem := range memories {
		switch mem.Scope {
		case ScopeProject:
			stats.Project++
		case ScopeGlobal:
			stats.Global++
		}
		if mem.UpdatedAt.Before(cutoff) {
			stats.Stale++
		}
		stats.TotalSize += mem.Size
	}
	return stats
}

// MemoryStats contiene estadísticas del sistema de memorias.
type MemoryStats struct {
	Total     int
	Project   int
	Global    int
	Stale     int
	TotalSize int64
}
