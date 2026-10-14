---
name: adw
description: Workflow autónomo que planifica (opcional), implementa los planes de plans/, hace commit, push y abre un pull request
argument-hint: "[issue_url | plans/<plan>.md] [--remote origin] [--base main]"
allowed-tools: Bash, Read, Glob, Grep, Write, Edit, Skill, AskUserQuestion
---

# AI Developer Workflow

Coordina las skills del proyecto de principio a fin sin intervención: planificar (`/feature`, `/bug`, `/chore`), implementar (`/implement`), hacer commit (`/git:commit`) y abrir el pull request.

## Argumentos

`$ARGUMENTS` puede contener, en cualquier orden:

- una URL de issue de GitHub (modo trigger, ver `adws/README.md`);
- una o varias rutas `plans/<plan>.md` concretas;
- `--remote <nombre>` (por defecto `origin`);
- `--base <rama>` (por defecto `main`).

## 1. Preparación

1. Ejecuta `git status --short --branch`. Si hay cambios sin confirmar fuera de `plans/`, detente y explícalo: no mezcles trabajo ajeno.
2. Ejecuta `git fetch <remote>` y comprueba que existe `<remote>/<base>`.
3. Obtén el repositorio de destino a partir de la URL de `<remote>` (`git remote get-url <remote>`, formato `OWNER/REPO`). Úsalo siempre con `gh ... --repo OWNER/REPO`, porque `gh` puede apuntar por defecto a otro remoto.

## 2. Decidir si hay un encargo nuevo

Antes de invocar `/feature`, `/bug` o `/chore`, recuerda la rama actual y sitúate sobre la rama de partida con `git switch --detach <remote>/<partida>`. Así el plan describe el código del que realmente partirá el cambio y no el de otra rama de trabajo. Los archivos no rastreados (los planes) viajan con el checkout.

**Rama de partida y PRs apilados.** Por defecto, la rama de partida es `<base>`. Si el encargo depende de un cambio que todavía no está en `<base>` porque vive en un PR abierto sin fusionar, la rama de partida es la rama de ese PR. Ocurre, por ejemplo, cuando la funcionalidad solo funciona con un arreglo pendiente de revisión.

- Para detectarlo, revisa `gh pr list --repo OWNER/REPO --state open --json number,title,headRefName,baseRefName` y relaciona el encargo con esos PRs.
- Solo apila si la dependencia es real: el cambio no funcionaría o no se podría validar sin ese PR. Si no lo es, parte de `<base>`.
- Si apilas, la rama de trabajo (paso 4) se crea desde `<remote>/<rama-del-PR>` y el PR nuevo (paso 7) usa `--base <rama-del-PR>`.
- Indícalo en la primera línea del cuerpo del PR: "Depende de #N. Cuando #N se fusione, hay que cambiar la base a `<base>`."

- **Con URL de issue**: no preguntes nada. Lee la issue con `gh issue view <url> --json title,body,labels` y clasifícala como `feature`, `bug` o `chore` según su contenido y etiquetas. Invoca esa skill con el título y el cuerpo de la issue como argumento para generar el plan en `plans/`. Solo se implementará ese plan.
- **Con rutas de planes**: no preguntes nada. Implementa solo esas rutas.
- **Sin argumentos**: pregunta al usuario con `AskUserQuestion` si quiere añadir algo nuevo al proyecto antes de implementar, con las opciones:
  - *Nueva funcionalidad* → pide la descripción e invoca `/feature`.
  - *Corregir un bug* → pide la descripción e invoca `/bug`.
  - *Tarea de mantenimiento* → pide la descripción e invoca `/chore`.
  - *No, solo ejecutar los planes pendientes*.

  Esa es la única pregunta del workflow. A partir de aquí continúa sin intervención.

## 3. Seleccionar los planes

`plans/README.md` es documentación del directorio, no un plan: **nunca** lo implementes, lo pases a `/implement` ni lo incluyas en un commit, aunque aparezca modificado o se pase como argumento. Descártalo siempre antes de cualquier otro filtro.

Si no se dieron rutas explícitas ni issue, los planes pendientes son los demás archivos `plans/*.md` que aparecen como no rastreados o modificados en `git status --short -- plans/`. Un plan ya confirmado en git se considera implementado.

Si no hay ningún plan pendiente, informa de ello y termina sin crear rama, commit ni PR.

## 4. Rama de trabajo

Por cada plan, crea una rama desde la rama de partida actualizada (`<base>` o la rama del PR del que depende, ver paso 2):

```bash
git switch -c <tipo>/<nombre-del-plan> <remote>/<partida> --no-track
```

`<tipo>` es `feat`, `fix` o `chore` según el encabezado del plan (`# Feature`, `# Bug`, `# Chore`). Los planes no rastreados viajan con el checkout. Si la rama ya existe, detente y explícalo.

Procesa los planes de uno en uno: cada plan obtiene su propia rama, commit y pull request.

## 5. Implementar

Invoca `/implement plans/<plan>.md`. Respeta sus reglas: no rediseñes ni amplíes el alcance.

Ejecuta los tests de Go siempre con `GO_ENV=test`, como hace la CI (`.github/workflows/ci.yaml`): `GO_ENV=test go test -p 1 ./...`. Sin esa variable, la aplicación arranca en modo desarrollo con CSRF activo y los tests de acciones que envían formularios fallan con 403. Si ves 403 inesperados, comprueba `GO_ENV` antes de dar los tests por fallidos.

Si `/implement` informa de un bloqueo, o los tests o validaciones del plan fallan, **detente**: no hagas commit ni PR de ese plan, deja la rama tal como está e informa del motivo. No continúes con los planes siguientes.

## 6. Commit

Invoca `/git:commit` indicando como alcance el plan y los archivos que ha cambiado `/implement`. El commit incluye el propio plan. El mensaje sigue Conventional Commits y termina con las líneas de atribución que indique la sesión.

## 7. Push y pull request

1. `git push -u <remote> <rama>`.
2. Crea el PR contra la rama de partida (`<base>`, o la rama del PR del que depende si está apilado):

   ```bash
   gh pr create --repo OWNER/REPO --base <partida> --head <rama> --title "<título del commit>" --body "<cuerpo>"
   ```

   El cuerpo, en español, incluye: resumen del cambio, lista de cambios por capa, cómo probarlo (comandos de validación ejecutados y su resultado) y la ruta del plan. Si se partió de una issue, añade `Closes <url>`. Termina con las líneas de atribución que indique la sesión.
3. Nunca hagas merge, no actives auto-merge ni fuerces el push.

## 8. Informe final

Para cada plan procesado indica: rama, hash del commit, URL del PR y resultado de las validaciones. Para los planes no procesados, el motivo. Vuelve a dejar el checkout en la rama donde empezaste si no quedan cambios sin confirmar.
