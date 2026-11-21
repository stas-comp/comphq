// Articles and COSMOS places on a job (SPEC B14.2, gates 8.10-8.19), layered
// over plain forms.
//
// Everything in web/templates/tasks/references.html is an ordinary form that
// works with no script (gate 8.19): the remove icons, the Link buttons, the
// "Add a COSMOS place" form, and the "Link an article" box, which is a GET form
// that comes back with its matches. This script only intercepts them: it posts
// the same form with fragment=1 and swaps in the section the server sends back
// (so a refusal's words arrive in it too, with what was typed still in the
// boxes), and it makes the article box search as you type, through
// /tasks/articles.json. Picking an article links it at once.
//
// The section is one partial, drawn by the task window and by the job's own
// page alike, so this one script serves both. (Copy is app/copy.js.)
(function () {
  const section = () => document.querySelector(".task-refs");
  let timer = null;
  let seq = 0;

  function parse(html) {
    const fresh = new DOMParser()
      .parseFromString(html, "text/html")
      .querySelector(".task-refs");
    return fresh ? document.importNode(fresh, true) : null;
  }

  function swap(fresh) {
    const old = section();
    if (old) old.replaceWith(fresh);
  }

  async function send(url, data) {
    data.set("fragment", "1");
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: data.toString(),
      redirect: "manual",
    });
    if (res.status !== 200) throw new Error("unexpected " + res.status);
    const fresh = parse(await res.text());
    if (!fresh) throw new Error("no section in the response");
    return fresh;
  }

  // In the capture phase, and stopping the event: the task window has a save
  // handler of its own for its forms, and these are never part of "Save".
  document.addEventListener(
    "submit",
    async (event) => {
      const form = event.target;
      if (!(form instanceof HTMLFormElement) || !form.closest(".task-refs"))
        return;
      event.stopPropagation();
      event.preventDefault();
      if (form.classList.contains("ref-search")) {
        clearTimeout(timer);
        search(form.closest(".task-refs"));
        return;
      }
      const data = new URLSearchParams(new FormData(form, event.submitter));
      try {
        const fresh = await send(form.action, data);
        swap(fresh);
        // Keep the keyboard somewhere sensible: the path box after a place, the
        // article box after an article.
        const focusOn = form.classList.contains("place-form")
          ? ".place-form input[name=path]"
          : ".ref-search input";
        const next = fresh.querySelector(focusOn);
        if (next) next.focus();
      } catch (err) {
        form.submit(); // the plain form does the same job
      }
    },
    true,
  );

  // The box: results as you type; Enter searches now instead of leaving the page.

  function results(root) {
    return root.querySelector(".ref-results");
  }

  function clear(root) {
    const box = results(root);
    while (box.firstChild) box.removeChild(box.firstChild);
    return box;
  }

  function line(text, className) {
    const p = document.createElement("p");
    p.className = className;
    p.textContent = text;
    return p;
  }

  async function link(root, articleId) {
    const data = new URLSearchParams();
    data.set("article_id", String(articleId));
    try {
      const fresh = await send("/tasks/" + root.dataset.taskId + "/articles", data);
      swap(fresh);
      const box = fresh.querySelector(".ref-search input");
      if (box) box.focus();
    } catch (err) {
      // Fall back to the plain page: the no-script picker does the same job.
      const input = root.querySelector(".ref-search input");
      window.location.href =
        "/tasks/" + root.dataset.taskId + "?article_q=" + encodeURIComponent(input ? input.value : "") + "#refs";
    }
  }

  async function search(root) {
    const input = root.querySelector(".ref-search input");
    const query = input.value;
    const mine = ++seq;
    if (query.trim().length < 2) {
      clear(root);
      return;
    }
    let data;
    try {
      const res = await fetch(
        "/tasks/articles.json?q=" +
          encodeURIComponent(query) +
          "&task=" +
          encodeURIComponent(root.dataset.taskId),
      );
      if (!res.ok) return;
      data = await res.json();
    } catch (err) {
      return;
    }
    if (mine !== seq) return; // a newer keystroke has moved on
    const box = clear(root);
    if (data.results.length === 0) {
      box.appendChild(
        line(
          "No published article matches “" + query + "”, or it is already linked.",
          "refs-empty",
        ),
      );
      return;
    }
    const ul = document.createElement("ul");
    ul.className = "ref-matches";
    for (const article of data.results) {
      const li = document.createElement("li");
      li.className = "ref-match";
      const button = document.createElement("button");
      button.type = "button";
      button.className = "ref-match-pick";
      const title = document.createElement("span");
      title.className = "ref-match-title";
      title.textContent = article.title;
      const category = document.createElement("span");
      category.className = "ref-match-category";
      category.textContent = article.category;
      button.appendChild(title);
      button.appendChild(document.createTextNode(" "));
      button.appendChild(category);
      button.addEventListener("click", () => link(root, article.id));
      li.appendChild(button);
      ul.appendChild(li);
    }
    box.appendChild(ul);
  }

  document.addEventListener("input", (event) => {
    const input = event.target;
    if (!(input instanceof HTMLInputElement) || !input.closest(".ref-search"))
      return;
    const root = input.closest(".task-refs");
    clearTimeout(timer);
    timer = setTimeout(() => search(root), 250);
  });
})();
