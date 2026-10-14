# Feature: Prioridades en las tareas

## Encargo y criterios de aceptación

Encargo: permitir asignar una prioridad a cada tarea, verla en la lista y usarla para ordenar.

Decisiones confirmadas con el usuario:

- Niveles: **Alta**, **Media**, **Baja** y **Sin prioridad**.
- Al crear, la prioridad es **obligatoria**: el selector arranca sin valor y hay que elegir explícitamente (elegir "Sin prioridad" es válido).
- Las tareas existentes se migran a **Sin prioridad**.
- La lista se ordena por prioridad (Alta → Media → Baja → Sin prioridad) y, dentro de cada nivel, por fecha de creación como hasta ahora.
- La prioridad se fija al crear y se cambia desde la página de edición.
- En la lista se muestra como etiqueta con texto y color; "Sin prioridad" no muestra etiqueta.

Criterios de aceptación:

1. El formulario de alta muestra un selector "Prioridad" con un placeholder sin valor ("Prioridad…") y las opciones Alta, Media, Baja y Sin prioridad.
2. Enviar el alta sin elegir prioridad responde 422, no crea la tarea, muestra "Elige una prioridad." y conserva el título escrito.
3. Enviar un valor de prioridad no reconocido responde 422 con el mismo mensaje y no crea la tarea.
4. Crear con un título válido y cualquiera de las cuatro prioridades responde 303 a `/` y persiste esa prioridad.
5. Si el alta falla por el título, el selector conserva la prioridad elegida; si falla por la prioridad, conserva el título.
6. La página de edición muestra el selector con la prioridad actual preseleccionada; guardar persiste título y prioridad y conserva `completed`, `id` y `created_at`.
7. En edición, un valor de prioridad vacío o no válido responde 422, no modifica la tarea y muestra "Elige una prioridad.".
8. La lista se ordena por `priority desc, created_at asc, id asc`.
9. Cada tarea Alta/Media/Baja muestra una etiqueta visible con el texto del nivel; las tareas Sin prioridad no muestran etiqueta.
10. Tras la migración, las tareas existentes tienen prioridad "Sin prioridad" y la base de datos rechaza valores fuera de rango.

Inferencias (no pedidas explícitamente, derivadas de las decisiones):

- Completar o reabrir una tarea no cambia su prioridad ni su posición por prioridad.
- El texto de ayuda de edición deja de decir que "los demás datos no cambian" y pasa a mencionar que el estado de completada se conserva.

## Comportamiento actual y propuesta

Actual:

- `tasks` tiene `id`, `title`, `completed`, `created_at` y `updated_at` (`migrations/20260901000000_create_tasks.up.fizz`).
- `TasksIndex` y la rama de error de `TasksCreate` ordenan por `created_at asc, id asc` (`actions/tasks.go`).
- `TasksCreate` hace `c.Bind(task)` sobre `models.Task`; `TasksUpdate` enlaza un struct de entrada solo con `title`.
- Las plantillas `templates/tasks/index.plush.html` y `templates/tasks/edit.plush.html` solo tienen el campo de título.

Propuesta:

- Nueva columna `priority integer NOT NULL DEFAULT 0` con una restricción `CHECK (priority BETWEEN 0 AND 3)`. Valores: `0` Sin prioridad, `1` Baja, `2` Media y `3` Alta. Un entero permite ordenar con `priority desc` sin `CASE` en SQL.
- Para distinguir "no elegida" de "Sin prioridad" (`0`), el formulario envía la prioridad como **cadena**: `""` significa no elegida y `"0"`–`"3"` son valores válidos. Las acciones la parsean desde un struct de entrada, el mismo patrón que ya usa `TasksUpdate`.
- `models.Task.Priority` lleva `form:"-"` para que `c.Bind` nunca la rellene implícitamente.

## Alcance

Incluido:

- Migración, modelo con constantes, etiquetas y validación de rango.
- Alta y edición con selector de prioridad y su validación.
- Orden de la lista por prioridad.
- Etiqueta visual en la lista y CSS asociado.
- Actualizar el fixture `tres tareas` y los tests de acciones y modelo.

Excluido:

- Cambio rápido de prioridad desde la lista (sin rutas nuevas).
- Filtros por prioridad u orden seleccionable por el usuario.
- Cambios en `public/assets/app.js`.
- Arreglar el `fetch(form.action)` del toggle en `app.js`, que hace un GET a una ruta solo-PUT (ver Riesgos).
- Cambiar la migración de seed existente (`20260901000001_seed_tasks.up.sql`).

## Diseño por capas

### Persistencia

Nueva migración `migrations/20261014000000_add_priority_to_tasks.up.fizz`:

```
add_column("tasks", "priority", "integer", {"default": 0})
sql("ALTER TABLE tasks ADD CONSTRAINT tasks_priority_range CHECK (priority BETWEEN 0 AND 3)")
```

En fizz las columnas son `NOT NULL` salvo `{"null": true}`; el `DEFAULT 0` asigna "Sin prioridad" a las filas existentes.

`migrations/20261014000000_add_priority_to_tasks.down.fizz`:

```
sql("ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_priority_range")
drop_column("tasks", "priority")
```

### Modelo (`models/task.go`)

```go
const (
	PriorityNone   = 0
	PriorityLow    = 1
	PriorityMedium = 2
	PriorityHigh   = 3
)

// PriorityOption describe una opción del selector de prioridad.
type PriorityOption struct {
	Value int
	Label string
}

// PriorityOptions lista las prioridades en el orden en que se ofrecen.
var PriorityOptions = []PriorityOption{
	{PriorityHigh, "Alta"}, {PriorityMedium, "Media"}, {PriorityLow, "Baja"}, {PriorityNone, "Sin prioridad"},
}
```

- Campo `Priority int `json:"priority" db:"priority" form:"-"``.
- `ParsePriority(s string) (int, bool)`: aplica `strings.TrimSpace` y `strconv.Atoi` y acepta solo 0–3. Devuelve `false` para `""` o cualquier otro valor.
- `(t Task) PriorityLabel() string`: devuelve "Alta", "Media", "Baja" o "".
- `(t Task) PriorityClass() string`: devuelve `"high"`, `"medium"`, `"low"` o `""` para la clase CSS.
- `Validate`: si `Priority` está fuera de 0–3, añade `verrs.Add("priority", PriorityRequiredMessage)` con `const PriorityRequiredMessage = "Elige una prioridad."`.

### Acciones (`actions/tasks.go`)

- Constante `tasksOrder = "priority desc, created_at asc, id asc"`, usada en `TasksIndex` y en la rama de error de `TasksCreate`.
- `TasksIndex`: hace `c.Set("selectedPriority", "")`.
- `TasksCreate`:
  - Enlaza un struct de entrada `{Title string `form:"title"`; Priority string `form:"priority"`}` en lugar de `c.Bind(task)`.
  - Construye la tarea con el título recortado.
  - Si `ParsePriority` falla, ejecuta igualmente `task.Validate(tx)` para recoger posibles errores del título, añade `verrs.Add("priority", models.PriorityRequiredMessage)` y renderiza 422 sin crear nada.
  - Si la prioridad es válida, la asigna y sigue el flujo actual de `tx.ValidateAndCreate`.
  - En cualquier 422 hace `c.Set("selectedPriority", input.Priority)` para conservar la elección.
- `TasksEdit`: hace `c.Set("selectedPriority", strconv.Itoa(task.Priority))`.
- `TasksUpdate`:
  - Añade `Priority string `form:"priority"`` al struct de entrada.
  - Si `ParsePriority` falla, añade el error de prioridad, junto con los errores del título si los hay, y renderiza 422 **sin** llamar a `ValidateAndUpdate`.
  - Si es válida, asigna título y prioridad y continúa como ahora. `completed`, `id` y `created_at` siguen sin enlazarse.
  - En el 422 hace `c.Set("selectedPriority", input.Priority)`.
- Helper en `actions/render.go`: `"priorityOptions": func() []models.PriorityOption { return models.PriorityOptions }`, para no repetir `c.Set` en cada acción.

### Renderizado

`templates/tasks/index.plush.html`:

- En la fila del alta, entre el input de título y el botón "Añadir", añadir:
  ```html
  <label for="task-priority" class="sr-only">Prioridad</label>
  <select id="task-priority" name="priority" class="priority-select" required <%= if (len(errors.Get("priority")) > 0) { %>aria-invalid="true" aria-describedby="priority-error"<% } %>>
    <option value="" <%= if (selectedPriority == "") { %>selected<% } %>>Prioridad…</option>
    <%= for (opt) in priorityOptions() { %>
      <option value="<%= opt.Value %>" <%= if (selectedPriority == fmt.Sprint(opt.Value)) { %>selected<% } %>><%= opt.Label %></option>
    <% } %>
  </select>
  ```
  Si Plush no expone `fmt.Sprint`, el helper devuelve el valor ya como cadena: añadir `ValueString()` a `PriorityOption` o un campo `Value string`. Elegir la opción que compile, sin crear otro mecanismo.
- `required` aporta la validación nativa del navegador, y el servidor sigue validando igual.
- El bloque de error actual solo pinta `errors.Get("title")` con `id="title-error"`; añadir un párrafo hermano `id="priority-error"` para `errors.Get("priority")`. Ajustar las condiciones `aria-invalid` y `aria-describedby` del input de título para que dependan de `errors.Get("title")` y no de `errors.HasAny()`, de modo que cada campo apunte a su propio error.
- En cada `<li>`, tras `.task-title`:
  ```html
  <%= if (t.PriorityLabel() != "") { %>
    <span class="task-priority task-priority-<%= t.PriorityClass() %>"><span class="sr-only">Prioridad </span><%= t.PriorityLabel() %></span>
  <% } %>
  ```

`templates/tasks/edit.plush.html`:

- Debajo del bloque del título, añadir `<label for="task-priority">Prioridad</label>` y el mismo `<select>` sin la opción placeholder, ya que siempre hay valor actual, con `selectedPriority` preseleccionado y su error `priority-error`.
- Cambiar el texto de `title-help` a "Hasta 200 caracteres." y añadir una ayuda breve que diga que el estado de completada no cambia.
- Ajustar `aria-invalid` y `aria-describedby` del título al igual que en el alta.

### CSS (`public/assets/app.css`)

- `.priority-select`: misma altura, borde (`--color-border`), radio y foco (`--color-focus`) que el input de título. Ancho automático y sin estirarse en `.new-task-row`. En pantallas estrechas debe caber en la fila o bajar sin provocar scroll horizontal.
- `.task-priority`: píldora pequeña (`font-size` menor, `padding` 0.1em 0.5em, `border-radius` 999px, `white-space: nowrap`), en la misma línea que `.task-status`.
- Variantes:
  - `-high`: texto `--color-primary` sobre un fondo rojo muy claro.
  - `-medium`: texto ámbar oscuro (p. ej. `#8a5a00`) sobre ámbar muy claro.
  - `-low`: texto `--color-muted` sobre `--color-sidebar`.
  - Verificar contraste ≥ 4.5:1 en cada combinación.
- Si `app.css` define tema oscuro, añadir equivalentes para las tres variantes.
- Subir la versión de caché del CSS en `templates/application.plush.html` (`?v=N` → `?v=N+1`) si el enlace la lleva, igual que hace `app.js?v=2`.

### Fixtures (`fixtures/tasks.toml`)

Añadir `priority` a las filas de `tres tareas`: "Comprar material de oficina" = 1 (Baja), "Llamar al gestor" = 3 (Alta), "Publicar la oferta de empleo" = 0. Así el orden por prioridad es observable en los tests.

## Archivos relevantes

| Ruta | Responsabilidad |
| --- | --- |
| `migrations/20261014000000_add_priority_to_tasks.up.fizz` | Nueva columna y restricción (nuevo) |
| `migrations/20261014000000_add_priority_to_tasks.down.fizz` | Reversión (nuevo) |
| `models/task.go` | Campo, constantes, opciones, parseo, etiquetas y validación |
| `models/task_test.go` | Tests de validación, parseo y etiquetas |
| `actions/tasks.go` | Orden, enlace y validación de prioridad en alta y edición |
| `actions/render.go` | Helper `priorityOptions` |
| `actions/tasks_test.go` | Tests de alta, edición, orden y etiquetas |
| `templates/tasks/index.plush.html` | Selector en el alta, etiqueta en la lista y errores por campo |
| `templates/tasks/edit.plush.html` | Selector preseleccionado y errores por campo |
| `public/assets/app.css` | Estilos del selector y de las etiquetas |
| `templates/application.plush.html` | Versión de caché del CSS, si aplica |
| `fixtures/tasks.toml` | Prioridades en el escenario `tres tareas` |

## Implementación paso a paso

1. Crear las dos migraciones fizz descritas.
2. Modelo: añadir constantes, `PriorityOption`, `PriorityOptions`, `PriorityRequiredMessage`, el campo `Priority`, `ParsePriority`, `PriorityLabel`, `PriorityClass` y la validación de rango en `Validate`.
3. Añadir el helper `priorityOptions` en `actions/render.go`.
4. Acciones:
   - Añadir la constante `tasksOrder` y usarla en las dos consultas.
   - Reescribir el enlace de `TasksCreate` con un struct de entrada y el manejo de prioridad.
   - Ampliar `TasksUpdate`.
   - Añadir `selectedPriority` en `TasksIndex`, `TasksEdit` y en las ramas 422.
5. Plantillas: selector y errores por campo en alta y edición, y etiqueta en la lista.
6. CSS: selector, píldoras y versión de caché.
7. Fixture: prioridades en `tres tareas`.
8. Tests: actualizar los existentes y añadir los nuevos (ver abajo).
9. Ejecutar las validaciones y revisar la UI en el navegador (alta, error sin prioridad, edición y orden).

## Tests y validación

Tests existentes que hay que adaptar:

- `Test_TasksCreate_Valid`, `Test_TasksCreate_EmptyTitle_Invalid` y `Test_TasksCreate_OnlySpaces_Invalid` usan `Post(&models.Task{...})`. Con `Priority` en `form:"-"` no se envía prioridad, así que hay que cambiarlos a `url.Values{"title": {...}, "priority": {"2"}}` para seguir probando solo el título.
- Los tests de `TasksUpdate` que esperan 303 (`PersistsTitleOnly`, `PendingTaskRemainsPending`, `TitleAtLimit`) deben enviar `"priority"`. Renombrar `Test_TasksUpdate_PersistsTitleOnly` a `Test_TasksUpdate_PersistsTitleAndPriority` y comprobar que `completed`, `id` y `created_at` no cambian.
- `Test_TasksUpdate_EmptyTitle` comprueba `aria-describedby="title-error"`, que debe seguir presente.

Nuevos tests de modelo (`models/task_test.go`):

- `Validate` acepta 0–3 y rechaza -1 y 4 con "Elige una prioridad.".
- `ParsePriority` acepta "0"–"3" y " 2 ", y rechaza "", "4", "-1", "alta" y "1.5".
- `PriorityLabel` y `PriorityClass` devuelven los valores esperados para cada nivel, y vacío para 0.
- Una tarea creada sin asignar `Priority` se persiste con 0 (`DEFAULT`).

Nuevos tests de acciones (`actions/tasks_test.go`):

- `Test_TasksCreate_WithEachPriority`: crea con "0" a "3" y persiste cada valor.
- `Test_TasksCreate_MissingPriority`: título válido y prioridad vacía dan 422, "Elige una prioridad.", cero tareas y título conservado en el `value`.
- `Test_TasksCreate_InvalidPriority`: "7" y "alta" dan 422 y cero tareas.
- `Test_TasksCreate_TitleErrorKeepsPriority`: título vacío y prioridad "3" dan 422 con la opción Alta `selected`.
- `Test_TasksIndex_OrdersByPriority`: con el fixture, "Llamar al gestor" aparece antes que "Comprar material de oficina", y esta antes que "Publicar la oferta de empleo" (comparar `strings.Index`).
- `Test_TasksIndex_ShowsPriorityLabels`: aparecen las etiquetas "Alta" y "Baja", y la tarea Sin prioridad no tiene `task-priority`.
- `Test_TasksIndex_PrioritySelectHasPlaceholder`: el alta contiene `<option value="" selected>` (o equivalente) y las cuatro opciones.
- `Test_TasksEdit_PreselectsPriority`: una tarea con prioridad 2 muestra la opción Media `selected`.
- `Test_TasksUpdate_ChangesPriority`: pasar de 0 a 3 persiste y conserva `completed`.
- `Test_TasksUpdate_InvalidPriority`: "" o "9" dan 422 y la tarea no cambia (título y prioridad originales).
- `Test_TasksToggle_KeepsPriority`: alternar no cambia `priority`.

Comandos (requieren PostgreSQL en `localhost:5432` y la base de test migrada):

```bash
docker compose up -d db
gofmt -l .
go vet ./...
go test -p 1 ./...
go build ./...
```

`go test` con `suite/v4` aplica las migraciones a `todo_test`. Si la base de test ya existía, comprobar que la nueva migración se aplica (la columna `priority` existe) antes de interpretar los fallos.

Validación manual: `docker compose up --build -d` y en http://localhost:3000 comprobar:

- Las tres tareas de seed aparecen sin etiqueta.
- Enviar el alta sin elegir prioridad lo bloquea el navegador; quitando `required` en DevTools, el servidor muestra el error.
- Crear una tarea Alta la coloca arriba.
- Editar y cambiar a Baja la recoloca.
- La vista es correcta a 375 px de ancho.

## Riesgos y recuperación

- **Tests existentes rotos por el campo obligatorio**: es esperado. Se adaptan en el paso 8; no relajar la obligatoriedad para que pasen.
- **`0` frente a "no elegida"**: el parseo debe hacerse desde la cadena del formulario, nunca desde un `int` enlazado, porque `""` se enlazaría como 0 y "Sin prioridad" pasaría por elección válida. Los tests `MissingPriority` lo cubren.
- **Comparación en Plush de cadena con entero**: si `selectedPriority == opt.Value` no compila o no coincide, exponer el valor como cadena en `PriorityOption` en lugar de convertir en la plantilla.
- **Migración en bases existentes**: `ADD COLUMN ... DEFAULT 0 NOT NULL` es instantáneo en PostgreSQL 11+ (aquí 17.6). Para revertir: `soda migrate down` (o `buffalo pop migrate down`) ejecuta el `.down.fizz`, que elimina restricción y columna.
- **Contenedor `app` con imagen antigua**: tras implementar, ejecutar `docker compose up --build -d` para que la imagen incluya la migración y las plantillas nuevas.
- **Bug preexistente fuera de alcance**: `public/assets/app.js` intercepta el toggle y llama a `fetch(form.action)` (GET), pero solo existe `PUT /tasks/{id}/toggle`. Con JS activo, el cambio de estado no se persiste. Esta feature no lo toca; conviene tratarlo con `/bug` por separado.
