package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

// Niveles de prioridad de una tarea. Un valor mayor indica más prioridad.
const (
	PriorityNone   = 0
	PriorityLow    = 1
	PriorityMedium = 2
	PriorityHigh   = 3
)

// PriorityRequiredMessage es el error mostrado cuando no se elige una prioridad válida.
const PriorityRequiredMessage = "Elige una prioridad."

// PriorityOption describe una opción del selector de prioridad.
type PriorityOption struct {
	Value int
	Label string
}

// ValueString devuelve el valor de la opción tal como se envía en el formulario.
func (o PriorityOption) ValueString() string {
	return strconv.Itoa(o.Value)
}

// PriorityOptions lista las prioridades en el orden en que se ofrecen.
var PriorityOptions = []PriorityOption{
	{PriorityHigh, "Alta"},
	{PriorityMedium, "Media"},
	{PriorityLow, "Baja"},
	{PriorityNone, "Sin prioridad"},
}

// ParsePriority convierte el valor del formulario en una prioridad válida.
// Una cadena vacía significa que no se ha elegido prioridad.
func ParsePriority(s string) (int, bool) {
	p, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || p < PriorityNone || p > PriorityHigh {
		return 0, false
	}
	return p, true
}

// Task representa una tarea de la lista.
type Task struct {
	ID        uuid.UUID `json:"id" db:"id" form:"-"`
	Title     string    `json:"title" db:"title" form:"title"`
	Completed bool      `json:"completed" db:"completed" form:"-"`
	Priority  int       `json:"priority" db:"priority" form:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// Tasks es una colección de Task.
type Tasks []Task

// PriorityLabel devuelve el nombre visible de la prioridad, o "" si no tiene.
func (t Task) PriorityLabel() string {
	switch t.Priority {
	case PriorityHigh:
		return "Alta"
	case PriorityMedium:
		return "Media"
	case PriorityLow:
		return "Baja"
	}
	return ""
}

// PriorityClass devuelve el sufijo de clase CSS de la prioridad, o "" si no tiene.
func (t Task) PriorityClass() string {
	switch t.Priority {
	case PriorityHigh:
		return "high"
	case PriorityMedium:
		return "medium"
	case PriorityLow:
		return "low"
	}
	return ""
}

// shortMonths son las abreviaturas en español de los meses, empezando en enero.
var shortMonths = [...]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}

// RelativeDate describe en español cuánto hace que ocurrió t respecto a now,
// contando días naturales en la zona horaria de now.
func RelativeDate(t, now time.Time) string {
	t = t.In(now.Location())
	ty, tm, td := t.Date()
	ny, nm, nd := now.Date()
	days := int(time.Date(ny, nm, nd, 0, 0, 0, 0, time.UTC).Sub(time.Date(ty, tm, td, 0, 0, 0, 0, time.UTC)).Hours() / 24)

	switch {
	case days <= 0:
		return "hoy"
	case days == 1:
		return "ayer"
	case days < 7:
		return fmt.Sprintf("hace %d días", days)
	}
	return fmt.Sprintf("%d %s %d", td, shortMonths[tm-1], ty)
}

// CreatedAtISO devuelve la fecha de creación en formato ISO 8601 (RFC 3339, UTC).
func (t Task) CreatedAtISO() string {
	return t.CreatedAt.UTC().Format(time.RFC3339)
}

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (t *Task) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	title := strings.TrimSpace(t.Title)
	if title == "" {
		verrs.Add("title", "El título es obligatorio.")
	} else if len([]rune(title)) > TitleMaxLength {
		verrs.Add("title", fmt.Sprintf("El título no puede superar los %d caracteres.", TitleMaxLength))
	}

	if t.Priority < PriorityNone || t.Priority > PriorityHigh {
		verrs.Add("priority", PriorityRequiredMessage)
	}

	return verrs, nil
}
