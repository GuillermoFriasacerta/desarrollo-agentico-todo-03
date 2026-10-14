package models

import (
	"testing"
	"time"
)

func TestRelativeDate(t *testing.T) {
	now := time.Date(2026, 1, 9, 10, 0, 0, 0, time.UTC)
	cet := time.FixedZone("CET", 3600)

	cases := []struct {
		name string
		t    time.Time
		now  time.Time
		want string
	}{
		{"0 días, ahora", now, now, "hoy"},
		{"0 días, medianoche", time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC), now, "hoy"},
		{"1 día, última hora", time.Date(2026, 1, 8, 23, 59, 0, 0, time.UTC), now, "ayer"},
		{"1 día, medianoche", time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC), now, "ayer"},
		{"2 días", time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC), now, "hace 2 días"},
		{"6 días", time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC), now, "hace 6 días"},
		{"7 días", time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC), now, "2 ene 2026"},
		{"más antiguo", time.Date(2025, 12, 15, 8, 0, 0, 0, time.UTC), now, "15 dic 2025"},
		{"futuro", now.Add(48 * time.Hour), now, "hoy"},
		{"zona horaria de now", time.Date(2026, 1, 8, 23, 10, 0, 0, time.UTC), time.Date(2026, 1, 9, 0, 30, 0, 0, cet), "hoy"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RelativeDate(c.t, c.now); got != c.want {
				t.Errorf("RelativeDate(%v, %v) = %q, want %q", c.t, c.now, got, c.want)
			}
		})
	}
}

func TestTaskCreatedAtISO(t *testing.T) {
	task := Task{CreatedAt: time.Date(2026, 1, 2, 11, 4, 5, 0, time.FixedZone("CET", 3600))}

	if got, want := task.CreatedAtISO(), "2026-01-02T10:04:05Z"; got != want {
		t.Errorf("CreatedAtISO() = %q, want %q", got, want)
	}
}
