package actions

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"todo/models"

	"github.com/gofrs/uuid"
)

func (as *ActionSuite) Test_TasksIndex_EmptyState() {
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "No hay tareas todavía")
}

func (as *ActionSuite) Test_TasksIndex_ListsTasks() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "Comprar material de oficina")
	as.Contains(body, "Llamar al gestor")
	as.Contains(body, "Publicar la oferta de empleo")
	as.Contains(body, "Completada")
	as.Contains(body, "Pendientes")
	as.Contains(body, "Tareas completadas")
	as.NotContains(body, "data-completed-section hidden")

	heading := strings.Index(body, `id="completed-heading"`)
	as.True(heading > 0)
	as.Less(strings.Index(body, "Comprar material de oficina"), heading)
	as.Less(strings.Index(body, "Llamar al gestor"), heading)
	as.Greater(strings.Index(body, "Publicar la oferta de empleo"), heading)
}

func (as *ActionSuite) Test_TasksIndex_HidesCompletedSectionWhenNone() {
	task := &models.Task{Title: "Solo pendiente"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "data-completed-section hidden")
	as.Contains(body, "data-pending-empty hidden")
}

func (as *ActionSuite) Test_TasksIndex_AllCompleted() {
	task := &models.Task{Title: "Ya terminada", Completed: true}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "data-pending-empty >No tienes tareas pendientes.")
	as.NotContains(body, "data-completed-section hidden")
	as.Greater(strings.Index(body, "Ya terminada"), strings.Index(body, `id="completed-heading"`))
}

func (as *ActionSuite) Test_TasksCreate_Valid() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"  Nueva tarea de prueba  "}, "priority": {"2"}})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("Nueva tarea de prueba", task.Title)
	as.Equal(models.PriorityMedium, task.Priority)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksCreate_EmptyTitle_Invalid() {
	res := as.HTML("/tasks").Post(url.Values{"title": {""}, "priority": {"2"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_OnlySpaces_Invalid() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"     "}, "priority": {"2"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksToggle_CompletesAndReopens() {
	task := &models.Task{Title: "Pendiente de completar"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)

	res = as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksToggle_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s/toggle", id).Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksToggle_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/toggle").Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksDestroy() {
	task := &models.Task{Title: "Tarea que se elimina"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s", task.ID).Delete()
	as.Equal(303, res.Code)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksDestroy_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s", id).Delete()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) taskForEditing(completed bool) *models.Task {
	task := &models.Task{Title: "Título original", Completed: completed}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	as.NoError(as.DB.Reload(task))
	return task
}

func (as *ActionSuite) Test_TasksIndex_EditLink() {
	task := as.taskForEditing(false)
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "/tasks/"+task.ID.String()+"/edit")
	as.Contains(res.Body.String(), "Editar tarea: Título original")
}

func (as *ActionSuite) Test_TasksIndex_ShowsTaskAge() {
	task := &models.Task{Title: "Tarea reciente"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	as.NoError(as.DB.Find(task, task.ID))

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, `class="task-age"`)
	as.Contains(body, `datetime="`+task.CreatedAtISO()+`"`)
	as.Contains(body, ">hoy</time>")
}

func (as *ActionSuite) Test_TasksEdit_PrefillsTitle() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `value="Título original"`)
	as.Contains(res.Body.String(), `value="PUT"`)
	as.Contains(res.Body.String(), "Guardar cambios")
	as.Contains(res.Body.String(), "Cancelar")
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksEdit_NotFound() {
	res := as.HTML("/tasks/%s/edit", uuid.Must(uuid.NewV4())).Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/edit").Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_EscapesTitle() {
	task := &models.Task{Title: `<img src=x onerror="alert(1)">`}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), `<img src=x`)
	as.Contains(res.Body.String(), "&lt;img")
}

func (as *ActionSuite) Test_TasksUpdate_PersistsTitleAndPriority() {
	task := as.taskForEditing(true)
	id, createdAt, updatedAt := task.ID, task.CreatedAt, task.UpdatedAt
	res := as.HTML("/tasks/%s", id).Put(url.Values{
		"title": {"  Título actualizado  "}, "priority": {"3"}, "completed": {"false"},
		"id": {uuid.Must(uuid.NewV4()).String()}, "created_at": {"2000-01-01T00:00:00Z"},
	})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))
	as.NoError(as.DB.Reload(task))
	as.Equal("Título actualizado", task.Title)
	as.Equal(models.PriorityHigh, task.Priority)
	as.Equal(id, task.ID)
	as.True(task.Completed)
	as.True(task.CreatedAt.Equal(createdAt))
	as.False(task.UpdatedAt.Before(updatedAt))
	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(1, count)
}

func (as *ActionSuite) Test_TasksUpdate_PendingTaskRemainsPending() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo título"}, "priority": {"0"}, "completed": {"true"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Nuevo título", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_EmptyTitle() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {""}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.Contains(res.Body.String(), `aria-describedby="title-error"`)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_OnlySpaces() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"   "}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_TitleTooLong() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength+1)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título no puede superar los 200 caracteres.")
	as.Contains(res.Body.String(), title)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_TitleAtLimit() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}, "priority": {"0"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal(title, task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_NotFound() {
	res := as.HTML("/tasks/%s", uuid.Must(uuid.NewV4())).Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid").Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksCreate_WithEachPriority() {
	for _, p := range []int{models.PriorityNone, models.PriorityLow, models.PriorityMedium, models.PriorityHigh} {
		res := as.HTML("/tasks").Post(url.Values{"title": {fmt.Sprintf("Tarea %d", p)}, "priority": {strconv.Itoa(p)}})
		as.Equal(303, res.Code)

		task := &models.Task{}
		as.NoError(as.DB.Where("title = ?", fmt.Sprintf("Tarea %d", p)).First(task))
		as.Equal(p, task.Priority)
	}
}

func (as *ActionSuite) Test_TasksCreate_MissingPriority() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Tarea sin elegir"}, "priority": {""}})
	as.Equal(422, res.Code)
	body := res.Body.String()
	as.Contains(body, models.PriorityRequiredMessage)
	as.Contains(body, `value="Tarea sin elegir"`)
	as.Contains(body, `aria-describedby="priority-error"`)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_InvalidPriority() {
	for _, p := range []string{"7", "-1", "alta"} {
		res := as.HTML("/tasks").Post(url.Values{"title": {"Tarea"}, "priority": {p}})
		as.Equal(422, res.Code)
		as.Contains(res.Body.String(), models.PriorityRequiredMessage)
	}

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_TitleErrorKeepsPriority() {
	res := as.HTML("/tasks").Post(url.Values{"title": {""}, "priority": {"3"}})
	as.Equal(422, res.Code)
	body := res.Body.String()
	as.Contains(body, "El título es obligatorio.")
	as.Contains(body, `<option value="3" selected>Alta</option>`)
	as.NotContains(body, models.PriorityRequiredMessage)
}

func (as *ActionSuite) Test_TasksIndex_OrdersByPriority() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	high := strings.Index(body, "Llamar al gestor")
	low := strings.Index(body, "Comprar material de oficina")
	none := strings.Index(body, "Publicar la oferta de empleo")
	as.True(high >= 0 && low >= 0 && none >= 0)
	as.True(high < low, "Alta debe aparecer antes que Baja")
	as.True(low < none, "Baja debe aparecer antes que Sin prioridad")
}

func (as *ActionSuite) Test_TasksIndex_ShowsPriorityLabels() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, `task-priority-high"><span class="sr-only">Prioridad </span>Alta`)
	as.Contains(body, `task-priority-low"><span class="sr-only">Prioridad </span>Baja`)
	as.Equal(2, strings.Count(body, `class="task-priority `))
}

func (as *ActionSuite) Test_TasksIndex_PrioritySelectHasPlaceholder() {
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, `<option value="" selected>Prioridad…</option>`)
	as.Contains(body, `<option value="3">Alta</option>`)
	as.Contains(body, `<option value="2">Media</option>`)
	as.Contains(body, `<option value="1">Baja</option>`)
	as.Contains(body, `<option value="0">Sin prioridad</option>`)
}

func (as *ActionSuite) Test_TasksEdit_PreselectsPriority() {
	task := &models.Task{Title: "Con prioridad media", Priority: models.PriorityMedium}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `<option value="2" selected>Media</option>`)
	as.NotContains(res.Body.String(), "Prioridad…")
}

func (as *ActionSuite) Test_TasksUpdate_ChangesPriority() {
	task := as.taskForEditing(true)
	as.Equal(models.PriorityNone, task.Priority)

	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Título original"}, "priority": {"3"}})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.Equal(models.PriorityHigh, task.Priority)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidPriority() {
	task := &models.Task{Title: "Título original", Priority: models.PriorityLow}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	for _, p := range []string{"", "9"} {
		res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Otro título"}, "priority": {p}})
		as.Equal(422, res.Code)
		as.Contains(res.Body.String(), models.PriorityRequiredMessage)

		as.NoError(as.DB.Reload(task))
		as.Equal("Título original", task.Title)
		as.Equal(models.PriorityLow, task.Priority)
	}
}

func (as *ActionSuite) Test_TasksToggle_KeepsPriority() {
	task := &models.Task{Title: "Tarea urgente", Priority: models.PriorityHigh}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
	as.Equal(models.PriorityHigh, task.Priority)
}

func (as *ActionSuite) Test_TasksIndex_SectionsOrderByPriority() {
	for _, t := range []*models.Task{
		{Title: "Pendiente baja", Priority: models.PriorityLow},
		{Title: "Completada media", Priority: models.PriorityMedium, Completed: true},
		{Title: "Completada alta", Priority: models.PriorityHigh, Completed: true},
	} {
		verrs, err := as.DB.ValidateAndCreate(t)
		as.NoError(err)
		as.False(verrs.HasAny())
	}

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	pendingLow := strings.Index(body, "Pendiente baja")
	heading := strings.Index(body, `id="completed-heading"`)
	completedHigh := strings.Index(body, "Completada alta")
	completedMedium := strings.Index(body, "Completada media")
	as.True(pendingLow >= 0 && heading >= 0 && completedHigh >= 0 && completedMedium >= 0)
	as.True(pendingLow < heading, "las pendientes van antes del apartado de completadas")
	as.True(heading < completedHigh, "una completada de prioridad alta va en completadas")
	as.True(completedHigh < completedMedium, "dentro de completadas se ordena por prioridad")
}
