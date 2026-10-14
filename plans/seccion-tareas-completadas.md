# Feature: Sección de tareas completadas

## Encargo y criterios de aceptación

Encargo del usuario: la lista debe tener un apartado de **tareas completadas**. Cuando el usuario marque una tarea como hecha, la tarea se tacha y pasa a ese apartado.

Base: este plan parte del arreglo de persistencia del toggle (commit `92d9284`, rama `fix/completar-tarea-no-persiste`). Con ese arreglo, `public/assets/app.js` ya envía el toggle como `POST` con `_method=PUT` y revierte la interfaz si falla.

Requisitos explícitos:

- Existe un apartado visible titulado **"Tareas completadas"**.
- Al marcar una tarea como hecha, se muestra tachada y se mueve a ese apartado.

Inferencias (derivadas del encargo y del diseño actual):

- Las tareas no completadas se muestran en un apartado **"Pendientes"**, encima del de completadas.
- El movimiento ocurre al instante en la interfaz, sin recargar, igual que hoy cambia el estado visual. Tras recargar, el servidor pinta cada tarea en su apartado.
- Reabrir una tarea completada la devuelve a "Pendientes" sin tachar.
- Dentro de cada apartado se conserva el orden actual: `created_at asc, id asc`. Al moverse en la interfaz, la tarea se inserta en su posición según ese orden.
- El contador de la cabecera ("N tareas en tu lista") sigue contando todas las tareas.
- El estado vacío global ("No hay tareas todavía…") se mantiene cuando no hay ninguna tarea.

Criterios de aceptación:

1. Con tareas pendientes y completadas, `GET /` muestra las pendientes dentro del apartado "Pendientes" y las completadas dentro del apartado "Tareas completadas", que va después.
2. Las tareas completadas mantienen el tachado (`task-completed`), la etiqueta "Completada" y el botón "Reabrir".
3. Si no hay ninguna tarea completada, el apartado "Tareas completadas" se renderiza con el atributo `hidden`.
4. Si todas las tareas están completadas, el apartado "Pendientes" muestra "No tienes tareas pendientes.".
5. Sin ninguna tarea se muestra solo el estado vacío actual.
6. En el navegador, al completar una tarea, esta se tacha y se mueve al apartado de completadas. Si estaba oculto, el apartado aparece. Tras recargar sigue ahí.
7. Al reabrirla vuelve a "Pendientes" en su posición por fecha. Si el apartado de completadas queda vacío, se oculta. Tras recargar sigue ahí.
8. Si la petición del toggle falla, la tarea vuelve a su estado y apartado anteriores.
9. El alta con título inválido (422) sigue renderizando la lista con ambos apartados.

## Comportamiento actual y propuesta

Actual:

- `TasksIndex` y la rama de error de `TasksCreate` (`actions/tasks.go`) cargan todas las tareas ordenadas por `created_at asc, id asc` y las exponen como `tasks`.
- `templates/tasks/index.plush.html` las pinta en una única `<ul class="task-list">`. Las completadas solo se distinguen por la clase `task-completed` (tachado) y la etiqueta "Completada".
- `public/assets/app.js` cambia el estado visual de la tarea en su sitio y la persiste con `fetch`.

Propuesta:

- Las acciones separan la colección en `pendingTasks` y `completedTasks`, conservando el orden. Se mantiene `tasks` para el contador y el estado vacío. Se hace con un helper compartido para no duplicar código entre `TasksIndex` y `TasksCreate`.
- La plantilla pinta dos apartados con un parcial común para cada tarea (`templates/tasks/_task.plush.html`), siguiendo el patrón de `partial("flash.plush.html")` en el layout.
- `app.js` mueve el `<li>` entre listas e inserta la tarea en orden según el atributo `data-created`. También muestra u oculta el apartado de completadas y el mensaje de pendientes vacío.

No se necesitan migraciones ni cambios en el modelo: `completed` ya existe y es `NOT NULL`.

## Alcance

Incluido:

- Helper en acciones para separar pendientes y completadas, usado por `TasksIndex` y la rama 422 de `TasksCreate`.
- Plantilla con los apartados "Pendientes" y "Tareas completadas" y un parcial `_task.plush.html`.
- JavaScript para mover la tarea entre apartados, también al revertir.
- CSS para los encabezados de apartado y versión de los assets.
- Tests de acciones para la separación en apartados.

Excluido:

- Borrar u ocultar definitivamente las tareas completadas, plegar el apartado, "borrar completadas" o contadores por apartado.
- Cambios en rutas, `TasksToggle`, `TasksUpdate`, `TasksDestroy`, el modelo o las migraciones.
- La página de edición.
- Infraestructura de tests de JavaScript.

## Diseño por capas

### Acción

En `actions/tasks.go` añade:

```go
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
```

Sustituye `c.Set("tasks", tasks)` por `setTaskLists(c, tasks)` en `TasksIndex` y en la rama `verrs.HasAny()` de `TasksCreate`. La consulta y su orden no cambian.

### Renderizado

`templates/tasks/_task.plush.html` (nuevo): mueve aquí, sin cambios, el `<li class="task…">…</li>` completo de `index.plush.html`, con la variable `t`. Añade al `<li>` el atributo `data-created="<%= t.CreatedAt.UnixNano() %>"` para que el JavaScript pueda ordenar.

`templates/tasks/index.plush.html`: sustituye el bloque `<section aria-labelledby="list-heading">…</section>` por:

```html
<%= if (len(tasks) == 0) { %>
  <p class="empty-state">No hay tareas todavía. Añade la primera con el formulario.</p>
<% } else { %>
  <section aria-labelledby="pending-heading" class="task-section">
    <h2 id="pending-heading" class="section-heading">Pendientes</h2>
    <p class="empty-state" data-pending-empty <%= if (len(pendingTasks) > 0) { %>hidden<% } %>>No tienes tareas pendientes.</p>
    <ul class="task-list" data-pending-list>
      <%= for (t) in pendingTasks { %><%= partial("tasks/task.plush.html", {t: t}) %><% } %>
    </ul>
  </section>

  <section aria-labelledby="completed-heading" class="task-section" data-completed-section <%= if (len(completedTasks) == 0) { %>hidden<% } %>>
    <h2 id="completed-heading" class="section-heading">Tareas completadas</h2>
    <ul class="task-list" data-completed-list>
      <%= for (t) in completedTasks { %><%= partial("tasks/task.plush.html", {t: t}) %><% } %>
    </ul>
  </section>
<% } %>
```

La cabecera con `list-count` y el formulario de alta no cambian. Si el parcial no recibiera `authenticity_token` del contexto, pásalo explícitamente: `{t: t, authenticity_token: authenticity_token}`. El layout ya usa parciales de Buffalo, que heredan el contexto de la vista.

### JavaScript

En `public/assets/app.js`, dentro del manejador de `[data-task-toggle]`:

1. Antes de cambiar nada, guarda el `<li>` (`task`), la lista y el siguiente hermano originales (`task.parentNode`, `task.nextSibling`) para poder revertir.
2. Tras `applyState(task, form, completed)`, llama a `moveTask(task, completed)`.
3. En el `catch`, además de `applyState(task, form, !completed)`, devuelve el `<li>` a su posición original con `originalList.insertBefore(task, originalNext)` y llama a `refreshSections()`.

Funciones nuevas:

```js
function moveTask(task, completed) {
  var list = document.querySelector(completed ? "[data-completed-list]" : "[data-pending-list]");
  if (!list) {
    return;
  }
  var created = Number(task.dataset.created);
  var next = Array.prototype.find.call(list.children, function (item) {
    return item !== task && Number(item.dataset.created) > created;
  });
  list.insertBefore(task, next || null);
  refreshSections();
}

function refreshSections() {
  var pendingList = document.querySelector("[data-pending-list]");
  var completedList = document.querySelector("[data-completed-list]");
  var pendingEmpty = document.querySelector("[data-pending-empty]");
  var completedSection = document.querySelector("[data-completed-section]");
  if (pendingEmpty && pendingList) {
    pendingEmpty.hidden = pendingList.children.length > 0;
  }
  if (completedSection && completedList) {
    completedSection.hidden = completedList.children.length === 0;
  }
}
```

`UnixNano` puede superar `Number.MAX_SAFE_INTEGER` y perder precisión en el último dígito. Para ordenar por fecha con resolución de microsegundos es suficiente. A igualdad de valor, la tarea se inserta al final de su grupo.

### CSS

En `public/assets/app.css`, tras `.task-list`, añade:

```css
.task-section + .task-section {
  margin-top: 2rem;
}

.section-heading {
  margin-bottom: 0.25rem;
  color: var(--color-muted);
  font-size: 0.8rem;
  font-weight: 650;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

[hidden] {
  display: none !important;
}
```

La regla `[hidden]` garantiza que `display` de otras clases no anula el atributo.

En `templates/application.plush.html` sube las versiones de los assets: `app.css?v=3` → `app.css?v=4` y `app.js?v=3` → `app.js?v=4`.

## Archivos relevantes

- `actions/tasks.go`: `TasksIndex` y `TasksCreate` (rama 422); nuevo helper `setTaskLists`.
- `templates/tasks/index.plush.html`: estructura de los apartados.
- `templates/tasks/_task.plush.html` (nuevo): parcial con el `<li>` de cada tarea.
- `public/assets/app.js`: toggle con movimiento entre apartados y reversión.
- `public/assets/app.css`: estilos de apartado y `[hidden]`.
- `templates/application.plush.html`: versiones de `app.css` y `app.js`.
- `actions/tasks_test.go`: tests de la página principal.
- `fixtures/tasks.toml`: escenario `tres tareas` (dos pendientes y una completada); no cambia.

## Implementación paso a paso

1. Añade `setTaskLists` en `actions/tasks.go` y úsalo en `TasksIndex` y en la rama de error de `TasksCreate`.
2. Crea `templates/tasks/_task.plush.html` moviendo el `<li>` actual y añadiendo `data-created`.
3. Reescribe la sección de la lista en `templates/tasks/index.plush.html` como se indica en Diseño.
4. Actualiza `public/assets/app.js` con `moveTask`, `refreshSections` y la reversión de posición.
5. Añade los estilos a `public/assets/app.css` y sube las versiones en `templates/application.plush.html`.
6. Actualiza y añade los tests de la sección siguiente.

## Tests y validación

Tests en `actions/tasks_test.go`:

- Actualiza `Test_TasksIndex_ListsTasks` (fixture `tres tareas`). Además de lo que ya comprueba, verifica el orden en el HTML. Las posiciones (`strings.Index`) de "Comprar material de oficina" y "Llamar al gestor" son menores que la de `id="completed-heading"`, y esta es menor que la de "Publicar la oferta de empleo". Comprueba también que el cuerpo contiene "Tareas completadas" y "Pendientes", y que no contiene `data-completed-section hidden`.
- Nuevo `Test_TasksIndex_HidesCompletedSectionWhenNone`: crea solo una tarea pendiente. Comprueba que el cuerpo contiene `data-completed-section hidden` y que el mensaje `data-pending-empty` lleva `hidden`.
- Nuevo `Test_TasksIndex_AllCompleted`: crea solo una tarea con `Completed: true`. Comprueba que "No tienes tareas pendientes." se muestra sin `hidden` y que la tarea aparece después de `id="completed-heading"`.
- `Test_TasksIndex_EmptyState` y los tests de alta inválida deben seguir pasando sin cambios.

Comandos:

- `gofmt -l .` sin salida.
- `go vet ./...`
- `go test -p 1 ./...` (PostgreSQL y migraciones de test preparadas según `database.yml` y `.github/workflows/ci.yaml`).
- `go build ./...`

Validación en navegador sobre un binario de esta rama (`go build -o <tmp>/todo-app ./cmd/app` y `PORT=3001 <tmp>/todo-app`, con la base de datos de desarrollo):

- Al completar una tarea pendiente, se tacha y se mueve a "Tareas completadas". Tras recargar sigue ahí.
- Al reabrirla, vuelve a "Pendientes" en su posición. Tras recargar sigue ahí.
- Al completar la única pendiente aparece "No tienes tareas pendientes.". Al reabrir la única completada se oculta el apartado.
- Sin errores en la consola. Deja los datos de desarrollo en el estado inicial al terminar.

## Riesgos y recuperación

- **Contexto del parcial**: si `authenticity_token` no llegara al parcial, los formularios fallarían por CSRF. Lo detectan los tests (render) y la validación en navegador. Recuperación: pasarlo explícitamente en el mapa del parcial.
- **Nombre del parcial**: Buffalo resuelve `partial("tasks/task.plush.html")` a `templates/tasks/_task.plush.html`. Si falla, el render da error y los tests de índice fallan.
- **Atributo `hidden` y CSS**: sin la regla `[hidden]`, una clase con `display` podría mostrar el apartado vacío. La cubre el paso 5.
- **Caché de assets**: lo cubre la subida de versiones.
- **Dependencia del PR del arreglo**: esta rama parte de `fix/completar-tarea-no-persiste`. Si ese PR cambia antes de fusionarse, habrá que rebasar esta rama.
- **Reversión**: no hay migraciones; deshacer el commit restaura el comportamiento anterior.
