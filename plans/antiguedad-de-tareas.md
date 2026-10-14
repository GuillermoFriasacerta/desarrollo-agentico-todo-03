# Feature: Mostrar la antigüedad de cada tarea

Issue de origen: https://github.com/GuillermoFriasacerta/desarrollo-agentico-todo-03/issues/10

## Encargo y criterios de aceptación

Encargo: mostrar en cada tarea, en texto discreto, cuándo se creó en formato relativo ("hoy", "ayer", "hace 3 días"…).

Requisitos explícitos (de la issue):

1. Cada tarea muestra su antigüedad a partir de `created_at`, en español:
   - mismo día: `hoy`;
   - un día antes: `ayer`;
   - de 2 a 6 días: `hace N días`;
   - 7 días o más: la fecha con formato `2 ene 2026` (día sin cero inicial, mes abreviado en minúsculas y año).
2. El texto va dentro de `<time datetime="...">` con la fecha ISO 8601.
3. El formato se calcula en el servidor con una función Go pura y testeable que recibe la fecha y el "ahora". Sin JavaScript.
4. Tests unitarios de la función para cada regla y sus límites (0, 1, 2, 6 y 7 días).
5. Estilo discreto (color atenuado, tamaño pequeño), coherente con el CSS existente y sin romper el diseño en móvil.
6. Si cambia `app.css`, se sube su `?v=N` en `templates/application.plush.html`.

Inferencias:

- "Días" son **días naturales** (cambio de fecha en el calendario), no bloques de 24 h: una tarea creada ayer a las 23:50 es "ayer" a las 00:10 de hoy.
- Ambas fechas se comparan en la zona horaria de `now` (`created.In(now.Location())`). La acción pasa `time.Now()`, es decir, la zona local del servidor.
- Una fecha futura (reloj desajustado) se muestra como `hoy`.
- Se muestra en todas las tareas, pendientes y completadas, también en la respuesta 422 de `TasksCreate`, que reutiliza la misma plantilla.
- Meses abreviados: `ene feb mar abr may jun jul ago sep oct nov dic`.
- El atributo `datetime` usa RFC 3339 en UTC (`2026-01-02T10:04:05Z`), que es ISO 8601 válido.

## Comportamiento actual y propuesta

Actual: `templates/tasks/_task.plush.html` pinta en cada `<li class="task">` el toggle, el título (`span.task-title`, columna 2 del grid), la prioridad (columna 3), la etiqueta oculta "Completada" y las acciones (columna 4). `created_at` solo aparece como `data-created` para que `app.js` ordene al mover tareas. No se muestra ninguna fecha.

Propuesta:

- `models/task.go` añade la función pura `RelativeDate(t, now time.Time) string` y el método `Task.CreatedAtISO() string`.
- `actions/render.go` registra el helper Plush `taskAge` que llama a `models.RelativeDate(t, time.Now())`. La fuente de "ahora" queda en la capa de presentación y la función sigue siendo pura.
- El parcial envuelve el título y un nuevo `<time class="task-age">` en un contenedor `div.task-body` que ocupa la columna 2, así la antigüedad queda bajo el título sin tocar las otras columnas.
- `app.css` añade `.task-body` y `.task-age`, mueve el centrado y el relleno vertical de `.task-title` a `.task-body`, y se sube la versión del CSS a `?v=6`.

Sin cambios en el modelo persistido ni en migraciones.

## Alcance

Incluido:

- Función `models.RelativeDate` y método `Task.CreatedAtISO`.
- Tests unitarios de la función (sin base de datos).
- Helper `taskAge` en `actions/render.go`.
- Marcado en `_task.plush.html` y estilos en `app.css`, con la versión del CSS subida.
- Un test de acción que comprueba el `<time>` renderizado.

Excluido:

- Fecha de completado o de última edición.
- Cambios en el modelo, migraciones o fixtures.
- JavaScript (no se toca `app.js`; el `<li>` se mueve entero entre apartados y conserva su `<time>`).
- Configurar una zona horaria concreta (p. ej. `Europe/Madrid`) o añadir `time/tzdata`.

## Diseño por capas

### Modelo (`models/task.go`)

```go
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
```

Construir las medianoches en UTC después de extraer la fecha local evita errores por cambios de horario de verano (días de 23 o 25 h). `fmt` y `time` ya están importados.

### Presentación (`actions/render.go`)

Añadir a `Helpers`, junto a `priorityOptions`:

```go
"taskAge": func(t time.Time) string { return models.RelativeDate(t, time.Now()) },
```

Importar `time`.

### Plantilla (`templates/tasks/_task.plush.html`)

Sustituir `<span class="task-title"><%= t.Title %></span>` por:

```html
<div class="task-body">
  <span class="task-title"><%= t.Title %></span>
  <time class="task-age" datetime="<%= t.CreatedAtISO() %>"><%= taskAge(t.CreatedAt) %></time>
</div>
```

`app.js` busca `.task-title` con `querySelector` dentro del `<li>` (línea 45) y sigue funcionando. La etiqueta `.task-status` se sigue insertando antes de `.task-actions` como hijo directo del `<li>`, igual que ahora.

### CSS (`public/assets/app.css`)

- Cambiar `.task-title` a solo `min-width: 0; overflow-wrap: anywhere; line-height: 1.55;` (quitar `align-self: center` y `padding: 0.65rem 0`).
- Añadir, justo después:

```css
.task-body {
  grid-column: 2;
  align-self: center;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 0.65rem 0;
}

.task-age {
  color: var(--color-muted);
  font-size: 0.75rem;
  line-height: 1.4;
}
```

- `.task-completed .task-title` sigue aplicando el tachado solo al título; la antigüedad no se tacha.
- No hace falta regla nueva en `@media (max-width: 760px)`: la columna 2 es `minmax(0, 1fr)` y el texto es corto. La regla móvil de `.task-title { font-size: 0.9rem; }` se mantiene.

En `templates/application.plush.html`, cambiar `/assets/app.css?v=5` por `/assets/app.css?v=6`. `app.js` no cambia, así que su versión no se toca.

## Archivos relevantes

- `models/task.go`: modelo `Task`; aquí se añaden `RelativeDate`, `shortMonths` y `CreatedAtISO`.
- `models/relative_date_test.go` (nuevo): tests unitarios `testing` estándar de `RelativeDate` y `CreatedAtISO`, sin `ModelSuite`.
- `actions/render.go`: motor de render y helpers Plush; aquí se añade `taskAge`.
- `templates/tasks/_task.plush.html`: parcial de cada tarea; añade `div.task-body` y `<time>`.
- `public/assets/app.css`: estilos de `.task-title`, `.task-body` y `.task-age`.
- `templates/application.plush.html`: layout; versión del CSS.
- `actions/tasks_test.go`: test de acción del renderizado.
- `public/assets/app.js`: solo lectura; confirma que los selectores siguen siendo válidos.

## Implementación paso a paso

1. En `models/task.go`, añadir `shortMonths`, `RelativeDate` y `CreatedAtISO` tal como se describe arriba.
2. Crear `models/relative_date_test.go` (paquete `models`, `import "testing"` y `time`) con un test de tabla `TestRelativeDate`. Usar `now := time.Date(2026, 1, 9, 10, 0, 0, 0, time.UTC)` y casos:
   - 0 días: `now` → `hoy`; mismo día a las 00:00 → `hoy`;
   - 1 día: 2026-01-08 23:59 → `ayer`; 2026-01-08 00:00 → `ayer`;
   - 2 días: 2026-01-07 → `hace 2 días`;
   - 6 días: 2026-01-03 → `hace 6 días`;
   - 7 días: 2026-01-02 → `2 ene 2026`;
   - más antiguo: 2025-12-15 → `15 dic 2025`;
   - futuro: `now.Add(48 * time.Hour)` → `hoy`;
   - zona horaria: con `loc := time.FixedZone("CET", 3600)`, `now = 2026-01-09 00:30 CET` y `t = 2026-01-08 23:10 UTC` (que es 09/01 00:10 CET) → `hoy`.
   Añadir `TestTaskCreatedAtISO`: una tarea con `CreatedAt` en `FixedZone` devuelve la cadena RFC 3339 en UTC terminada en `Z`.
3. En `actions/render.go`, importar `time` y registrar el helper `taskAge`.
4. En `templates/tasks/_task.plush.html`, envolver el título y añadir `<time>` como arriba.
5. En `public/assets/app.css`, ajustar `.task-title` y añadir `.task-body` y `.task-age`.
6. En `templates/application.plush.html`, subir `app.css?v=5` a `?v=6`.
7. En `actions/tasks_test.go`, añadir `Test_TasksIndex_ShowsTaskAge`: crear una tarea con `ValidateAndCreate`, recargarla con `as.DB.Find`, hacer `GET /` y comprobar que el cuerpo contiene `class="task-age"`, `datetime="` + `task.CreatedAtISO()` + `"` y `>hoy</time>`.
8. Ejecutar `gofmt -w` sobre los archivos Go tocados.

## Tests y validación

Con PostgreSQL de test disponible y migraciones aplicadas:

```bash
gofmt -l .
go vet ./...
GO_ENV=test go test -p 1 ./...
go build ./...
```

- `gofmt -l .` no debe listar archivos.
- Deben pasar los tests nuevos (`TestRelativeDate`, `TestTaskCreatedAtISO`, `Test_TasksIndex_ShowsTaskAge`) y todos los existentes. Sin `GO_ENV=test`, los tests de formularios fallan con 403 por CSRF.
- Comprobación de focalización: `go test ./models -run 'TestRelativeDate|TestTaskCreatedAtISO'` no necesita base de datos.

Comprobación manual en navegador (con `/env:start` y `/playwright-cli`):

- La lista muestra "hoy" bajo el título de cada tarea, en gris y más pequeño, con `<time datetime>` en el DOM.
- A 375 px de ancho la fila no se desborda y las acciones siguen visibles.
- Completar y reabrir una tarea la mueve de apartado conservando la antigüedad; solo el título se tacha.
- La hoja servida es `app.css?v=6`.

## Riesgos y recuperación

- **Zona horaria**: la imagen Alpine no incluye `tzdata` ni define `TZ`, así que el servidor calcula los días en UTC. Para un usuario en España, entre las 00:00 y la 01:00/02:00 locales una tarea de "ayer" puede verse como "hoy". Es aceptable para este alcance; si molesta, el arreglo es pasar `time.Now().In(loc)` en el helper con `time/tzdata` importado, sin tocar `RelativeDate`.
- **Diseño de la fila**: el contenedor nuevo aumenta algo la altura de las filas (una línea de 0.75rem). Si se rompe la alineación con la prioridad o las acciones, revisar `align-self` de `.task-body`; la prioridad y las acciones mantienen sus `grid-column` explícitos.
- **Caché de assets**: si no se sube `?v=6`, los navegadores pueden servir el CSS antiguo, donde `.task-title` aún tiene relleno. Comprobar la versión en el layout.
- **Tests de acción con el reloj**: el test de acción depende de `time.Now()`. Crear la tarea y pedir la página casi a la vez hace que "hoy" sea estable, salvo que la ejecución cruce la medianoche UTC, lo cual es improbable y basta con repetirlo.
- Recuperación: el cambio no toca datos; revertir el commit restaura el estado anterior.
