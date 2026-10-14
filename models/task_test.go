package models

import "strings"

func (ms *ModelSuite) Test_Task_Create_Valid() {
	task := &Task{Title: "Comprar café"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NotZero(task.ID)
	ms.False(task.Completed)

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(1, count)
}

func (ms *ModelSuite) Test_Task_Title_Required() {
	task := &Task{Title: ""}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Title_OnlySpaces_Invalid() {
	task := &Task{Title: "   \t  "}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")
}

func (ms *ModelSuite) Test_Task_Title_MaxLength() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
}

func (ms *ModelSuite) Test_Task_Title_TooLong_Invalid() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength+1)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.NotEmpty(verrs.Get("title"))
}

func (ms *ModelSuite) Test_Task_Toggle_Persists() {
	task := &Task{Title: "Revisar el correo"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	task.Completed = true
	ms.NoError(ms.DB.Update(task))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.True(reloaded.Completed)
}

func (ms *ModelSuite) Test_Task_Priority_DefaultsToNone() {
	task := &Task{Title: "Sin prioridad asignada"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.Equal(PriorityNone, reloaded.Priority)
}

func (ms *ModelSuite) Test_Task_Priority_Range() {
	for _, p := range []int{PriorityNone, PriorityLow, PriorityMedium, PriorityHigh} {
		verrs, err := (&Task{Title: "Tarea", Priority: p}).Validate(ms.DB)
		ms.NoError(err)
		ms.False(verrs.HasAny())
	}

	for _, p := range []int{-1, 4} {
		verrs, err := (&Task{Title: "Tarea", Priority: p}).Validate(ms.DB)
		ms.NoError(err)
		ms.Contains(verrs.Get("priority"), PriorityRequiredMessage)
	}
}

func (ms *ModelSuite) Test_ParsePriority() {
	for in, want := range map[string]int{"0": 0, "1": 1, "2": 2, "3": 3, " 2 ": 2} {
		got, ok := ParsePriority(in)
		ms.True(ok, in)
		ms.Equal(want, got, in)
	}

	for _, in := range []string{"", "4", "-1", "alta", "1.5"} {
		_, ok := ParsePriority(in)
		ms.False(ok, in)
	}
}

func (ms *ModelSuite) Test_Task_PriorityLabelAndClass() {
	cases := []struct {
		priority     int
		label, class string
	}{
		{PriorityHigh, "Alta", "high"},
		{PriorityMedium, "Media", "medium"},
		{PriorityLow, "Baja", "low"},
		{PriorityNone, "", ""},
	}
	for _, c := range cases {
		task := Task{Priority: c.priority}
		ms.Equal(c.label, task.PriorityLabel())
		ms.Equal(c.class, task.PriorityClass())
	}
}
