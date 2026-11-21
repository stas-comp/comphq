// Steps, articles and places in the Add task form (SPEC B14.3, D-106, gates
// 8.30-8.34).
//
// Before the job exists nothing can be saved, so everything added here lives in
// the form: each step, article and place is a line with hidden fields (step,
// article_id, place_name and place_path), and Add task posts them all with the
// job, in one go. The lines use the same classes as a saved job's lists, so they
// look the same; the controls work on the form in the browser and post nothing.
// The server checks every limit again when it saves.
//
// Without script, the form shows only the one-per-line box (steps_text) and a
// note about articles and places: this script is what reveals the rest.
//
// The form is drawn on the new-task page and fetched into the task window as a
// fragment, so it is set up on load and again whenever the window draws
// something (window.js sends "window:shown").
(function () {
  const MAX_STEPS = 50;
  const MAX_STEP_CHARS = 200;
  const MAX_ARTICLES = 20;
  const MAX_PLACES = 20;
  const MAX_PATH = 400;
  const MAX_NAME = 80;
  const PATH_WORDS =
    "That doesn't look like a folder path. It should start with \\\\ (like \\\\NAS\\COSMOS) or a drive letter (like S:\\COSMOS).";

  function el(tag, className, text) {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined) node.textContent = text;
    return node;
  }

  function hidden(name, value) {
    const input = document.createElement("input");
    input.type = "hidden";
    input.name = name;
    input.value = value;
    return input;
  }

  // The icon-only buttons, drawn as the server draws them: a plain button with
  // an icon and the same title and spoken name.
  function iconButton(icon, label) {
    const b = el("button", "icon-btn");
    b.type = "button";
    b.title = label;
    b.setAttribute("aria-label", label);
    const span = el("span", "icon icon-" + icon);
    span.setAttribute("aria-hidden", "true");
    b.appendChild(span);
    return b;
  }

  function notice(section, text) {
    let p = section.querySelector(".new-notice");
    if (!text) {
      if (p) p.remove();
      return;
    }
    if (!p) {
      p = el("p", "message new-notice");
      p.setAttribute("role", "alert");
      const first = section.querySelector(".new-js");
      section.insertBefore(p, first);
    }
    p.textContent = text;
  }

  // Steps ------------------------------------------------------------------

  function stepLine(text) {
    const li = el("li", "step");
    li.setAttribute("data-new-step", "");
    li.appendChild(hidden("step", text));
    const tick = el("span", "step-tick");
    const box = el("input", "check");
    box.type = "checkbox";
    box.disabled = true;
    box.tabIndex = -1;
    box.setAttribute("aria-hidden", "true");
    tick.appendChild(box);
    li.appendChild(tick);
    const body = el("span", "step-body");
    body.appendChild(el("span", "step-words", text));
    li.appendChild(body);
    const rename = el("button", "mini quiet step-rename-link", "Rename");
    rename.type = "button";
    rename.setAttribute("data-new", "rename-step");
    rename.appendChild(el("span", "visually-hidden", " the step “" + text + "”"));
    li.appendChild(rename);
    const actions = el("span", "step-actions");
    const label = "step “" + text + "”";
    actions.appendChild(iconButton("arrow-up", "Move " + label + " up"));
    actions.appendChild(iconButton("arrow-down", "Move " + label + " down"));
    actions.appendChild(iconButton("trash", "Remove " + label));
    li.appendChild(actions);
    return li;
  }

  function addStep(form) {
    const section = form.querySelector(".new-steps");
    const list = section.querySelector("[data-new-steps]");
    const input = section.querySelector("#new-step-text");
    const text = input.value.replace(/[\r\n\t]+/g, " ").trim();
    if (text === "") {
      notice(section, "Type what the step is first.");
      return;
    }
    if (Array.from(text).length > MAX_STEP_CHARS) {
      notice(section, "A step can be up to 200 characters.");
      return;
    }
    if (list.children.length >= MAX_STEPS) {
      notice(section, "A job can have up to 50 steps.");
      return;
    }
    notice(section, "");
    list.appendChild(stepLine(text));
    input.value = "";
    input.focus();
  }

  function renameStep(li) {
    const body = li.querySelector(".step-body");
    if (!body || li.querySelector(".step-rename")) return;
    const current = li.querySelector('input[name="step"]').value;
    const box = el("span", "step-rename");
    const input = el("input", "field");
    input.type = "text";
    input.value = current;
    input.setAttribute("autocomplete", "off");
    input.setAttribute("aria-label", "New words for the step “" + current + "”");
    const save = el("button", "mini go", "Save");
    save.type = "button";
    const cancel = el("button", "mini quiet", "Cancel");
    cancel.type = "button";
    box.append(input, save, cancel);
    body.hidden = true;
    li.querySelector('[data-new="rename-step"]').hidden = true;
    body.after(box);
    input.focus();
    input.select();

    const finish = () => {
      box.remove();
      body.hidden = false;
      li.querySelector('[data-new="rename-step"]').hidden = false;
    };
    const commit = () => {
      const text = input.value.replace(/[\r\n\t]+/g, " ").trim();
      const section = li.closest(".new-steps");
      if (text === "") {
        notice(section, "Type what the step is first.");
        return;
      }
      if (Array.from(text).length > MAX_STEP_CHARS) {
        notice(section, "A step can be up to 200 characters.");
        return;
      }
      notice(section, "");
      li.replaceWith(stepLine(text));
    };
    save.addEventListener("click", commit);
    cancel.addEventListener("click", finish);
    input.addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        commit();
      } else if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        finish();
      }
    });
  }

  // Articles ---------------------------------------------------------------

  function articleLine(id, title, category) {
    const li = el("li", "ref-line");
    li.setAttribute("data-new-article", "");
    li.appendChild(hidden("article_id", String(id)));
    li.appendChild(el("span", "ref-line-title", title));
    if (category) li.appendChild(el("span", "ref-line-meta", category));
    li.appendChild(iconButton("trash", "Remove article " + title));
    return li;
  }

  let seq = 0;
  let timer = null;

  async function searchArticles(form) {
    const section = form.querySelector(".new-refs");
    const box = section.querySelector(".ref-results");
    const query = section.querySelector("#new-article-q").value;
    const mine = ++seq;
    const clear = () => {
      while (box.firstChild) box.removeChild(box.firstChild);
    };
    if (query.trim().length < 2) {
      clear();
      return;
    }
    let data;
    try {
      const res = await fetch("/tasks/articles.json?q=" + encodeURIComponent(query));
      if (!res.ok) return;
      data = await res.json();
    } catch (err) {
      return;
    }
    if (mine !== seq) return; // a newer keystroke has moved on
    const taken = new Set(
      Array.from(section.querySelectorAll('input[name="article_id"]')).map((i) => i.value),
    );
    const found = data.results.filter((a) => !taken.has(String(a.id)));
    clear();
    if (found.length === 0) {
      box.appendChild(
        el("p", "refs-empty", "No published article matches “" + query + "”, or it is already linked."),
      );
      return;
    }
    const ul = el("ul", "ref-matches");
    for (const article of found) {
      const li = el("li", "ref-match");
      const button = el("button", "ref-match-pick");
      button.type = "button";
      button.appendChild(el("span", "ref-match-title", article.title));
      button.appendChild(document.createTextNode(" "));
      button.appendChild(el("span", "ref-match-category", article.category));
      button.addEventListener("click", () => {
        const list = section.querySelector("[data-new-articles]");
        if (list.children.length >= MAX_ARTICLES) {
          notice(section, "A job can have at most 20 articles.");
          return;
        }
        notice(section, "");
        list.appendChild(articleLine(article.id, article.title, article.category));
        section.querySelector("#new-article-q").value = "";
        clear();
        section.querySelector("#new-article-q").focus();
      });
      li.appendChild(button);
      ul.appendChild(li);
    }
    box.appendChild(ul);
  }

  // Places -----------------------------------------------------------------

  function normalisePath(raw) {
    return raw.trim().replace(/\//g, "\\").trim();
  }

  function placeLabel(name, path) {
    if (name.trim() !== "") return name.trim();
    const parts = path.split("\\").filter((p) => p !== "");
    return parts.length ? parts[parts.length - 1] : path;
  }

  function placeLine(name, path) {
    const li = el("li", "ref-line ref-place");
    li.setAttribute("data-new-place", "");
    li.appendChild(hidden("place_name", name));
    li.appendChild(hidden("place_path", path));
    const label = placeLabel(name, path);
    li.appendChild(el("span", "ref-line-title", label));
    li.appendChild(el("code", "ref-path", path));
    const copy = el("button", "mini copy-button", "Copy");
    copy.type = "button";
    copy.setAttribute("data-copy", path);
    copy.setAttribute("data-copied-label", "Copied");
    li.appendChild(copy);
    li.appendChild(iconButton("trash", "Remove place " + label));
    return li;
  }

  function addPlace(form) {
    const section = form.querySelector(".new-refs");
    const pathBox = section.querySelector("#new-place-path");
    const nameBox = section.querySelector("#new-place-name");
    const path = normalisePath(pathBox.value);
    const name = nameBox.value.trim();
    const list = section.querySelector("[data-new-places]");
    if (path === "" || Array.from(path).length > MAX_PATH || !(/^\\\\[^\\]+\\/.test(path) || /^[A-Za-z]:\\/.test(path))) {
      notice(section, PATH_WORDS);
      return;
    }
    if (Array.from(name).length > MAX_NAME) {
      notice(section, "A place's name can be at most 80 characters.");
      return;
    }
    const lower = path.toLowerCase();
    const dup = Array.from(list.querySelectorAll('input[name="place_path"]')).some((i) => i.value.toLowerCase() === lower);
    if (dup) {
      notice(section, "That folder is already on this job.");
      return;
    }
    if (list.children.length >= MAX_PLACES) {
      notice(section, "A job can have at most 20 COSMOS places.");
      return;
    }
    notice(section, "");
    list.appendChild(placeLine(name, path));
    pathBox.value = "";
    nameBox.value = "";
    pathBox.focus();
  }

  // Setting up a form --------------------------------------------------------

  function enhance(form) {
    if (form.dataset.newEnhanced) return;
    form.dataset.newEnhanced = "1";
    form.querySelectorAll(".new-js").forEach((n) => (n.hidden = false));
    form.querySelectorAll(".new-nojs").forEach((n) => (n.hidden = true));

    const list = form.querySelector("[data-new-steps]");
    if (list && typeof Sortable !== "undefined") {
      Sortable.create(list, {
        animation: 150,
        handle: ".step-body",
        // Clicking a control must never be taken for the start of a drag.
        filter: "button, input, a, select",
        preventOnFilter: false,
      });
    }

    form.addEventListener("click", (event) => {
      const target = event.target instanceof Element ? event.target : null;
      if (!target) return;
      const action = target.closest("[data-new]");
      if (action && action.dataset.new === "add-step") {
        addStep(form);
        return;
      }
      if (action && action.dataset.new === "add-place") {
        addPlace(form);
        return;
      }
      if (action && action.dataset.new === "rename-step") {
        renameStep(action.closest("[data-new-step]"));
        return;
      }
      const button = target.closest("button.icon-btn");
      if (!button || !form.contains(button)) return;
      const step = button.closest("[data-new-step]");
      const removable = button.closest("[data-new-article], [data-new-place]");
      if (step) {
        if (button.querySelector(".icon-arrow-up") && step.previousElementSibling) {
          step.parentNode.insertBefore(step, step.previousElementSibling);
          button.focus();
        } else if (button.querySelector(".icon-arrow-down") && step.nextElementSibling) {
          step.parentNode.insertBefore(step.nextElementSibling, step);
          button.focus();
        } else if (button.querySelector(".icon-trash")) {
          const next = step.nextElementSibling || step.previousElementSibling;
          step.remove();
          const section = form.querySelector(".new-steps");
          notice(section, "");
          (next ? next.querySelector("button.icon-btn") : section.querySelector("#new-step-text"))?.focus();
        }
      } else if (removable && button.querySelector(".icon-trash")) {
        const section = form.querySelector(".new-refs");
        removable.remove();
        notice(section, "");
        section.querySelector("#new-article-q").focus();
      }
    });

    // Enter in one of the little boxes adds its line; it never saves the job.
    form.addEventListener("keydown", (event) => {
      if (event.key !== "Enter" || !(event.target instanceof HTMLInputElement)) return;
      const id = event.target.id;
      if (id === "new-step-text") {
        event.preventDefault();
        addStep(form);
      } else if (id === "new-place-path" || id === "new-place-name") {
        event.preventDefault();
        addPlace(form);
      } else if (id === "new-article-q") {
        event.preventDefault();
        clearTimeout(timer);
        searchArticles(form);
      }
    });

    form.addEventListener("input", (event) => {
      if (!(event.target instanceof HTMLInputElement) || event.target.id !== "new-article-q") return;
      clearTimeout(timer);
      timer = setTimeout(() => searchArticles(form), 250);
    });
  }

  function init() {
    document.querySelectorAll(".add-task-form").forEach(enhance);
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("window:shown", init);
})();
