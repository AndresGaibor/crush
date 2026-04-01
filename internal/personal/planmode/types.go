package planmode

import "time"

// PlanStepStatus representa el estado de un paso del plan.
type PlanStepStatus string

const (
	StepPending    PlanStepStatus = "pending"
	StepInProgress PlanStepStatus = "in_progress"
	StepCompleted  PlanStepStatus = "completed"
	StepFailed     PlanStepStatus = "failed"
	StepSkipped    PlanStepStatus = "skipped"
)

// IsValid indica si el estado pertenece al conjunto soportado.
func (s PlanStepStatus) IsValid() bool {
	switch s {
	case StepPending, StepInProgress, StepCompleted, StepFailed, StepSkipped:
		return true
	default:
		return false
	}
}

// PlanStep representa un paso del plan.
type PlanStep struct {
	Number       int            `json:"number"`
	Description  string         `json:"description"`
	Status       PlanStepStatus `json:"status"`
	Dependencies []int          `json:"dependencies,omitempty"`
	Result       string         `json:"result,omitempty"`
	Error        string         `json:"error,omitempty"`
}

// Plan representa un plan estructurado generado por el agente.
type Plan struct {
	ID             string     `json:"id"`
	Goal           string     `json:"goal"`
	Steps          []PlanStep `json:"steps"`
	Considerations []string   `json:"considerations,omitempty"`
	AffectedFiles  []string   `json:"affected_files,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Normalize completa los campos derivados del plan.
func (p *Plan) Normalize() {
	for i := range p.Steps {
		if p.Steps[i].Number == 0 {
			p.Steps[i].Number = i + 1
		}
		if p.Steps[i].Status == "" {
			p.Steps[i].Status = StepPending
		}
	}
}

// CurrentStep retorna el paso actualmente en progreso.
func (p *Plan) CurrentStep() *PlanStep {
	for i := range p.Steps {
		if p.Steps[i].Status == StepInProgress {
			return &p.Steps[i]
		}
	}
	return nil
}

// NextPendingStep retorna el siguiente paso pendiente cuyas dependencias ya se cumplieron.
func (p *Plan) NextPendingStep() *PlanStep {
	completed := make(map[int]struct{}, len(p.Steps))
	for i := range p.Steps {
		switch p.Steps[i].Status {
		case StepCompleted, StepSkipped:
			completed[p.Steps[i].Number] = struct{}{}
		}
	}

	for i := range p.Steps {
		step := &p.Steps[i]
		if step.Status != StepPending {
			continue
		}
		ok := true
		for _, dep := range step.Dependencies {
			if _, exists := completed[dep]; !exists {
				ok = false
				break
			}
		}
		if ok {
			return step
		}
	}
	return nil
}

// Progress retorna el porcentaje de pasos completados.
func (p *Plan) Progress() float64 {
	if len(p.Steps) == 0 {
		return 0
	}
	done := 0
	for _, step := range p.Steps {
		if step.Status == StepCompleted || step.Status == StepSkipped {
			done++
		}
	}
	return float64(done) / float64(len(p.Steps))
}

// PlanInput es la estructura que puede recibir la tool para cargar un plan.
type PlanInput struct {
	Goal           string          `json:"goal" description:"Objetivo general del plan"`
	Steps          []PlanStepInput `json:"steps" description:"Pasos del plan"`
	Considerations []string        `json:"considerations,omitempty" description:"Riesgos o notas importantes"`
	AffectedFiles  []string        `json:"affected_files,omitempty" description:"Archivos que podrían verse afectados"`
}

// PlanStepInput representa un paso recibido desde la tool.
type PlanStepInput struct {
	Number       int    `json:"number" description:"Número del paso; si se omite, se asigna automáticamente"`
	Description  string `json:"description" description:"Descripción del paso"`
	Dependencies []int  `json:"dependencies,omitempty" description:"Números de pasos de los que depende"`
}
