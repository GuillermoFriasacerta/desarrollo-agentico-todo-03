# Bug: completar una tarea desde la lista no se guarda

## Encargo y resultado esperado

Informe del usuario: al marcar una tarea como hecha desde la lista, la tarea "no se elimina"; al recargar la página sigue apareciendo.

Resultado esperado tras el arreglo:

1. Al pulsar el botón de completar, la tarea se marca como completada en la base de datos, no solo en pantalla.
2. Al recargar `/`, la tarea sigue mostrándose como completada (título tachado, etiqueta "Completada" y botón "Reabrir").
3. Pulsar de nuevo el botón la reabre y el cambio también persiste al recargar.
4. Si la petición falla, la interfaz vuelve al estado anterior en lugar de mostrar un estado que no se ha guardado.

## Evidencia y causa raíz

Hechos observados en el repositorio (rama base `main`, commit `ff94db6`):

- La ruta para alternar el estado es `PUT /tasks/{task_id}/toggle` (`actions/app.go:55`). No existe ninguna ruta `GET` para ese path.
- El formulario de cada tarea (`templates/tasks/index.plush.html`) hace `POST` a `/tasks/<id>/toggle` con `authenticity_token` y `_method=PUT` ocultos y el atributo `data-task-toggle`. Enviado de forma nativa, Buffalo lo convierte en `PUT` mediante `MethodOverride` (que solo actúa sobre peticiones `POST` y lee `_method` del cuerpo del formulario), y `TasksToggle` (`actions/tasks.go`) invierte `Completed` y redirige con 303.
- Pero `public/assets/app.js` intercepta el envío de cualquier formulario `[data-task-toggle]`: llama a `event.preventDefault()`, cambia la clase `task-completed`, las etiquetas y el estado "Completada" **solo en el DOM**, y después ejecuta `fetch(form.action)`.
- `fetch(form.action)` sin opciones hace un **`GET`** a `/tasks/<id>/toggle`, sin cuerpo, sin `_method=PUT` y sin `authenticity_token`. Ese `GET` no coincide con ninguna ruta, así que el servidor no ejecuta `TasksToggle` y responde con un error. El código no comprueba la respuesta.

Causa raíz: la petición que lanza `app.js` usa el método equivocado y no envía los datos del formulario. El cambio de estado nunca llega a la base de datos; la interfaz lo muestra de forma optimista y, al recargar, el servidor pinta la tarea como pendiente, igual que antes.

Los tests de acciones (`Test_TasksToggle_CompletesAndReopens` en `actions/tasks_test.go`) pasan porque llaman directamente a `PUT /tasks/<id>/toggle` y no ejercitan `app.js`. Por eso el fallo no se detectó.

Reproducción mínima:

1. Arrancar la aplicación (skill `env:start`) y abrir `/`.
2. Pulsar el botón de completar de una tarea pendiente: aparece tachada con la etiqueta "Completada".
3. En la pestaña de red se ve una petición `GET /tasks/<id>/toggle` con respuesta de error (no `PUT`/`POST`).
4. Recargar la página: la tarea vuelve a aparecer pendiente.

Incertidumbre: el usuario describe lo esperado como que la tarea "se elimine" o deje de aparecer. El diseño actual del repositorio muestra a propósito las tareas completadas (tachadas, con la etiqueta "Completada" y el botón "Reabrir", según la plantilla, el CSS `.task-completed` y `Test_TasksIndex_ListsTasks`). Por eso, lo que el usuario ve al recargar es casi con seguridad la tarea **otra vez pendiente**, porque el cambio no se guardó. Este plan corrige ese fallo y conserva el diseño actual. Ocultar o borrar las tareas completadas sería un cambio de producto y queda fuera de este plan (ver Alcance).

## Alcance

Incluido:

- Corregir en `public/assets/app.js` la petición del toggle para que envíe el formulario como `POST` con sus campos (`authenticity_token` y `_method=PUT`), igual que el envío nativo.
- Revertir el cambio optimista del DOM si la respuesta no es correcta o la petición falla.
- Actualizar el parámetro de versión del script en `templates/application.plush.html` (`app.js?v=2` → `app.js?v=3`) para que los navegadores no sigan usando la versión en caché.

Excluido:

- Ocultar, filtrar o borrar las tareas completadas de la lista: no cambian `TasksIndex`, la plantilla ni el modelo.
- Cambios en rutas, acciones Go, modelo, migraciones o CSS.
- Añadir infraestructura de tests de JavaScript o navegador al repositorio.

## Archivos relevantes

- `public/assets/app.js`: intercepta el envío del formulario de toggle, actualiza el DOM y hace la petición. Es donde está el fallo.
- `templates/application.plush.html`: carga `/assets/app.js?v=2`. Hay que subir la versión.
- `templates/tasks/index.plush.html`: formulario `data-task-toggle` con `authenticity_token` y `_method=PUT` (no cambia, solo es referencia).
- `actions/app.go`: ruta `PUT /tasks/{task_id}/toggle` (no cambia).
- `actions/tasks.go`: `TasksToggle` (no cambia).
- `actions/tasks_test.go`: tests de toggle existentes que deben seguir en verde.

## Implementación paso a paso

1. En `public/assets/app.js`, dentro del manejador de `submit` para `[data-task-toggle]`, extrae la actualización del DOM (clase `task-completed`, `.toggle-label`, `aria-pressed`, `aria-label`, `title` y la etiqueta `.task-status`) a una función local `applyState(task, form, completed)`, para poder reutilizarla al revertir. Mantén exactamente el mismo comportamiento visual que hoy.
2. Sustituye `fetch(form.action);` por una petición que reproduzca el envío nativo del formulario:

   ```js
   fetch(form.action, {
     method: "POST",
     body: new URLSearchParams(new FormData(form)),
     credentials: "same-origin",
     headers: { "Accept": "text/html" }
   }).then(function (response) {
     if (!response.ok) {
       throw new Error("toggle failed: " + response.status);
     }
   }).catch(function () {
     applyState(task, form, !completed);
   });
   ```

   `URLSearchParams` envía el cuerpo como `application/x-www-form-urlencoded`, que `MethodOverride` y el middleware CSRF leen con `FormValue`. `fetch` sigue la redirección 303 hacia `GET /` y `response.ok` será `true` cuando el guardado tenga éxito.
3. Conserva el resto de `app.js` sin cambios: la confirmación `data-confirm` debe seguir ejecutándose antes y, si el usuario cancela, no se envía nada.
4. En `templates/application.plush.html`, cambia `/assets/app.js?v=2` por `/assets/app.js?v=3`.

## Tests y validación

Este repositorio no tiene tests de JavaScript; los tests Go cubren el contrato del servidor, que ya es correcto. Validación:

1. Comprobaciones Go (no debe haber regresiones):
   - `gofmt -l .` sin salida.
   - `go vet ./...`
   - `go test -p 1 ./...` (requiere PostgreSQL y las migraciones de test, según `database.yml` y `.github/workflows/ci.yaml`).
   - `go build ./...`
2. Reproducción en navegador antes y después del arreglo (skill `env:start` para arrancar la aplicación; navegador integrado o skill `playwright-cli`):
   - Antes: tras pulsar completar y recargar, la tarea vuelve a estar pendiente, y la red muestra `GET /tasks/<id>/toggle` con error.
   - Después: la red muestra `POST /tasks/<id>/toggle` con 303 seguido de `GET /` 200. Al recargar, la tarea sigue completada. Pulsar otra vez y recargar la deja pendiente.
   - La consola del navegador no muestra errores.
   - La página carga `app.js?v=3`.

## Riesgos y recuperación

- **CSRF**: si `authenticity_token` no viaja en el cuerpo, el servidor responde con error y la interfaz revierte el cambio. Se detecta en la validación en navegador. Recuperación: comprobar que el cuerpo se construye con `new FormData(form)` y que el formulario incluye el campo oculto.
- **Caché del navegador**: sin subir la versión del script, un navegador podría seguir ejecutando el `app.js` antiguo. Lo cubre el paso 4.
- **Expectativa del usuario**: si lo que se quería era que las tareas completadas desaparezcan de la lista, este arreglo no basta y habrá que planificar una feature aparte (`/feature`) para ocultarlas o borrarlas.
- **Reversión**: el cambio se limita a dos archivos estáticos; deshacer el commit restaura el comportamiento anterior sin migraciones ni datos afectados.
