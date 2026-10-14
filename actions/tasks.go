package actions

import (
	"net/http"
	"strconv"
	"strings"

	"todo/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
)

// tasksOrder ordena por prioridad descendente y, dentro de cada nivel, por antigüedad.
const tasksOrder = "priority desc, created_at asc, id asc"

// TasksIndex muestra la lista de tareas y el formulario de creación.
func TasksIndex(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	tasks := models.Tasks{}
	if err := tx.Order(tasksOrder).All(&tasks); err != nil {
		return err
	}

	setTaskLists(c, tasks)
	c.Set("task", &models.Task{})
	c.Set("selectedPriority", "")
	c.Set("errors", validate.NewErrors())
	return c.Render(http.StatusOK, r.HTML("tasks/index.plush.html"))
}

// TasksCreate crea una tarea nueva a partir del formulario.
// setTaskLists expone la colección completa y sus apartados de pendientes y completadas.
func setTaskLists(c buffalo.Context, tasks models.Tasks) {
	pending := models.Tasks{}
	completed := models.Tasks{}
	for _, t := range tasks {
		if t.Completed {
			completed = append(completed, t)
		} else {
			pending = append(pending, t)
		}
	}

	c.Set("tasks", tasks)
	c.Set("pendingTasks", pending)
	c.Set("completedTasks", completed)
}

func TasksCreate(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	input := struct {
		Title    string `form:"title" json:"title"`
		Priority string `form:"priority" json:"priority"`
	}{}
	if err := c.Bind(&input); err != nil {
		return err
	}
	task := &models.Task{Title: strings.TrimSpace(input.Title)}

	var verrs *validate.Errors
	var err error
	if priority, ok := models.ParsePriority(input.Priority); ok {
		task.Priority = priority
		verrs, err = tx.ValidateAndCreate(task)
	} else {
		verrs, err = task.Validate(tx)
		verrs.Add("priority", models.PriorityRequiredMessage)
	}
	if err != nil {
		return err
	}

	if verrs.HasAny() {
		tasks := models.Tasks{}
		if err := tx.Order(tasksOrder).All(&tasks); err != nil {
			return err
		}

		setTaskLists(c, tasks)
		c.Set("task", task)
		c.Set("selectedPriority", input.Priority)
		c.Set("errors", verrs)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("tasks/index.plush.html"))
	}

	c.Flash().Add("success", "Tarea creada.")
	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksEdit muestra el formulario de edición de una tarea existente.
func TasksEdit(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	c.Set("task", task)
	c.Set("selectedPriority", strconv.Itoa(task.Priority))
	c.Set("errors", validate.NewErrors())
	return c.Render(http.StatusOK, r.HTML("tasks/edit.plush.html"))
}

// TasksUpdate actualiza el título y la prioridad, conservando los demás datos de la tarea.
func TasksUpdate(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	input := struct {
		Title    string `form:"title" json:"title"`
		Priority string `form:"priority" json:"priority"`
	}{}
	if err := c.Bind(&input); err != nil {
		return err
	}
	task.Title = strings.TrimSpace(input.Title)

	var verrs *validate.Errors
	var err error
	if priority, ok := models.ParsePriority(input.Priority); ok {
		task.Priority = priority
		verrs, err = tx.ValidateAndUpdate(task)
	} else {
		verrs, err = task.Validate(tx)
		verrs.Add("priority", models.PriorityRequiredMessage)
	}
	if err != nil {
		return err
	}
	if verrs.HasAny() {
		c.Set("task", task)
		c.Set("selectedPriority", input.Priority)
		c.Set("errors", verrs)
		return c.Render(http.StatusUnprocessableEntity, r.HTML("tasks/edit.plush.html"))
	}

	c.Flash().Add("success", "Tarea actualizada.")
	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksToggle alterna una tarea entre completada y pendiente.
func TasksToggle(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	task.Completed = !task.Completed
	if err := tx.Update(task); err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, "/")
}

// TasksDestroy elimina una tarea.
func TasksDestroy(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	if err := tx.Destroy(task); err != nil {
		return err
	}

	c.Flash().Add("success", "Tarea eliminada.")
	return c.Redirect(http.StatusSeeOther, "/")
}
