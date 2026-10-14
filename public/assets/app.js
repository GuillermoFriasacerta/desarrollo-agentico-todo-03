// Pide confirmación antes de enviar formularios marcados con data-confirm.
document.addEventListener("submit", function (event) {
  var form = event.target;
  if (form.dataset.confirm && !window.confirm(form.dataset.confirm)) {
    event.preventDefault();
  }

  if (!form.matches("[data-task-toggle]")) {
    return;
  }

  event.preventDefault();

  var task = form.closest(".task");
  var originalList = task.parentNode;
  var originalNext = task.nextSibling;
  var completed = !task.classList.contains("task-completed");
  applyState(task, form, completed);
  moveTask(task, completed);

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
    originalList.insertBefore(task, originalNext);
    refreshSections();
  });
});

function applyState(task, form, completed) {
  task.classList.toggle("task-completed", completed);
  var button = form.querySelector("button");
  var status = task.querySelector(".task-status");

  var label = completed ? "Reabrir" : "Completar";
  button.querySelector(".toggle-label").textContent = label;
  button.setAttribute("aria-pressed", String(completed));
  button.setAttribute("aria-label", label + " tarea: " + task.querySelector(".task-title").textContent);
  button.setAttribute("title", label + " tarea");

  if (completed && !status) {
    status = document.createElement("span");
    status.className = "task-status";
    status.textContent = "Completada";
    task.insertBefore(status, task.querySelector(".task-actions"));
  } else if (!completed && status) {
    status.remove();
  }
}

function moveTask(task, completed) {
  var list = document.querySelector(completed ? "[data-completed-list]" : "[data-pending-list]");
  if (!list) {
    return;
  }
  // Mismo orden que el servidor: prioridad descendente y, a igualdad, antigüedad.
  var priority = Number(task.dataset.priority);
  var created = Number(task.dataset.created);
  var next = Array.prototype.find.call(list.children, function (item) {
    if (item === task) {
      return false;
    }
    var itemPriority = Number(item.dataset.priority);
    return itemPriority < priority || (itemPriority === priority && Number(item.dataset.created) > created);
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
  var pendingCount = document.querySelector("[data-pending-count]");
  var completedCount = document.querySelector("[data-completed-count]");
  if (pendingCount && pendingList) {
    pendingCount.textContent = String(pendingList.children.length);
  }
  if (completedCount && completedList) {
    completedCount.textContent = String(completedList.children.length);
  }
}
