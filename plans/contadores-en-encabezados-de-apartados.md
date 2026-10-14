# Feature: Contadores en los encabezados de los apartados

## Encargo y criterios de aceptación

Origen: issue #5 (https://github.com/GuillermoFriasacerta/desarrollo-agentico-todo-03/issues/5). En la lista de tareas, los encabezados de los apartados deben mostrar cuántas tareas contiene cada uno: "Pendientes (N)" y "Tareas completadas (N)".

Requisitos explícitos:

1. Al cargar `/`, cada encabezado muestra entre paréntesis el número de tareas de su apartado.
2. Al completar o reabrir una tarea, los dos contadores se actualizan al instante, sin recargar.
3. Tras recargar, los contadores coinciden con lo que muestra el servidor.
4. El contador de la cabecera ("N tareas en tu lista") no cambia.
5. Hay tests de acciones que comprueban los números renderizados.

Fuera de alcance (explícito): cambios en el modelo, las migraciones o las rutas.

Inferencias:

- Si el toggle falla y la interfaz se revierte, los contadores vuelven también a sus valores anteriores (ya ocurre si se recalculan en `refreshSections()`, que se llama en el camino de reversión).
- Con "Pendientes" vacío el encabezado muestra "Pendientes (0)". El apartado de completadas vacío sigue oculto (`hidden`), aunque su contador valga 0.
- La rama 422 de `TasksCreate` renderiza la misma plantilla, así que también muestra los contadores sin cambios adicionales.

## Comportamiento actual y propuesta

Actual:

- `setTaskLists` (`actions/tasks.go`) expone `tasks`, `pendingTasks` y `completedTasks`.
- `templates/tasks/index.plush.html` pinta `<h2 id="pending-heading" class="section-heading">Pendientes</h2>` y `<h2 id="completed-heading" class="section-heading">Tareas completadas</h2>`, sin números.
- `public/assets/app.js` mueve la tarea entre `[data-pending-list]` y `[data-completed-list]` y llama a `refreshSections()` para ajustar el estado vacío y la visibilidad del apartado de completadas, tanto al mover como al revertir.

Propuesta:

- La plantilla añade al final de cada encabezado ` (<span data-pending-count>N</span>)` y ` (<span data-completed-count>N</span>)`, con `N = len(pendingTasks)` y `len(completedTasks)`.
- `refreshSections()` actualiza además el texto de esos `span` con `children.length` de cada lista. Así se cubren completar, reabrir y revertir con un solo punto.

## Alcance

Incluido:

- `templates/tasks/index.plush.html`: contadores en los dos `<h2>`.
- `public/assets/app.js`: actualización de contadores en `refreshSections()`.
- `actions/tasks_test.go`: aserciones sobre los números renderizados.

Excluido:

- Modelo, migraciones, rutas y acciones Go (los datos ya existen en el contexto).
- El contador de la cabecera `list-count` (no se toca).
- CSS: el número hereda el estilo de `.section-heading`; no se añade estilo nuevo.

## Diseño por capas

- **Acción**: sin cambios. `setTaskLists` ya expone `pendingTasks` y `completedTasks`.
- **Template** (`templates/tasks/index.plush.html`):
  - `<h2 id="pending-heading" class="section-heading">Pendientes (<span data-pending-count><%= len(pendingTasks) %></span>)</h2>`
  - `<h2 id="completed-heading" class="section-heading">Tareas completadas (<span data-completed-count><%= len(completedTasks) %></span>)</h2>`
  - Escribir cada `<h2>` en una sola línea, sin espacios dentro del `span`, para que los tests puedan buscar la cadena exacta.
- **JavaScript** (`public/assets/app.js`), en `refreshSections()`:
  - Obtener `[data-pending-count]` y `[data-completed-count]`.
  - Si existen y su lista también, asignar `textContent = String(list.children.length)`.

## Archivos relevantes

- `actions/tasks.go` — `TasksIndex`, `TasksCreate` (rama 422) y `setTaskLists`; solo lectura.
- `templates/tasks/index.plush.html` — encabezados de los apartados.
- `public/assets/app.js` — `moveTask`, reversión del toggle y `refreshSections`.
- `actions/tasks_test.go` — tests de acciones de la lista.
- `fixtures/tasks.toml` — fixture "tres tareas": 2 pendientes y 1 completada.

## Implementación paso a paso

1. En `templates/tasks/index.plush.html`, sustituir el contenido de los dos `<h2>` como se indica en "Diseño por capas".
2. En `public/assets/app.js`, ampliar `refreshSections()`:

   ```js
   var pendingCount = document.querySelector("[data-pending-count]");
   var completedCount = document.querySelector("[data-completed-count]");
   if (pendingCount && pendingList) {
     pendingCount.textContent = String(pendingList.children.length);
   }
   if (completedCount && completedList) {
     completedCount.textContent = String(completedList.children.length);
   }
   ```

3. En `actions/tasks_test.go`, añadir aserciones:
   - `Test_TasksIndex_ListsTasks` (fixture "tres tareas"): `Pendientes (<span data-pending-count>2</span>)` y `Tareas completadas (<span data-completed-count>1</span>)`; y que la cabecera sigue mostrando `3 tareas en tu lista`.
   - `Test_TasksIndex_HidesCompletedSectionWhenNone`: `data-pending-count>1<` y `data-completed-count>0<`.
   - `Test_TasksIndex_AllCompleted`: `data-pending-count>0<` y `data-completed-count>1<`.
   - Nuevo `Test_TasksToggle_UpdatesSectionCounts`: crear dos tareas pendientes, hacer `PUT /tasks/{id}/toggle` sobre una, luego `GET /` y comprobar `data-pending-count>1<` y `data-completed-count>1<` (criterio 3).
4. Ejecutar `gofmt -l .` y la validación.

## Tests y validación

Los tests de aplicación necesitan PostgreSQL con la base de test migrada, y `GO_ENV=test` (sin él, CSRF devuelve 403 en los tests de formularios).

```bash
gofmt -l .
go vet ./...
GO_ENV=test go test -p 1 ./...
go build ./...
```

Comprobación manual opcional (navegador, con el entorno arrancado): completar y reabrir una tarea y ver que "Pendientes (N)" y "Tareas completadas (N)" cambian al instante; recargar y comprobar que coinciden; el texto "N tareas en tu lista" no varía.

## Riesgos y recuperación

- **Espacios en blanco en la plantilla**: si los `span` o el texto se parten en varias líneas, las aserciones de cadena exacta fallarán. Mantener cada `<h2>` en una línea.
- **Tests existentes que buscan "Pendientes"/"Tareas completadas"**: siguen pasando porque el texto se conserva como prefijo.
- **Sin tests de JS**: la actualización en vivo solo se valida manualmente; el riesgo es bajo porque `refreshSections()` ya se ejecuta en todos los caminos.
- Recuperación: revertir los tres archivos; no hay datos ni esquema afectados.
