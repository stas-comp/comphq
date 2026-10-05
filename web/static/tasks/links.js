// Linked jobs (SPEC B13.6, gates 7.40-7.47), layered over plain forms.
//
// Everything in web/templates/tasks/links.html is an ordinary form that works
// with no script (gate 7.47): the remove icons, the three kind buttons, and the
// "Link a job" box, which is a GET form that comes back with its matches. This
// script only intercepts them: it posts the same form with fragment=1 and swaps
// in the section the server sends back (so a refusal's words arrive in it too),
// and it makes the box search as you type, through /tasks/search.json.
//
// The section is one partial, drawn by the task window and by the job's own
// page alike, so this one script serves both.
(function () {
  const section = () => document.querySelector(".task-links");
  let timer = null;
  let seq = 0;

  function parse(html) {
    const fresh = new DOMParser()
      .parseFromString(html, "text/html")
      .querySelector(".task-links");
    return fresh ? document.importNode(fresh, true) : null;
  }

  function swap(fresh) {
    const old = section();
    if (!old) return;
    old.replaceWith(fresh);
    // The card behind the window (its WAITING stamp) may now read differently.
    document.dispatchEvent(new CustomEvent("links:changed"));
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

  // Remove icons and the three kind buttons: post as a fragment, swap, and keep
  // the keyboard somewhere sensible (the box, so the next link can be typed).
  // In the capture phase, and stopping the event: the task window has a save
  // handler of its own for its forms, and these are never part of "Save".
  document.addEventListener(
    "submit",
    async (event) => {
      const form = event.target;
      if (!(form instanceof HTMLFormElement) || !form.closest(".task-links"))
        return;
      event.stopPropagation();
      event.preventDefault();
      if (form.classList.contains("link-search")) {
        clearTimeout(timer);
        search(form.closest(".task-links"));
        return;
      }
      const data = new URLSearchParams(new FormData(form, event.submitter));
      try {
        const fresh = await send(form.action, data);
        swap(fresh);
        const box = fresh.querySelector(".link-search input");
        if (box && form.classList.contains("link-kinds")) box.focus();
      } catch (err) {
        form.submit(); // the plain form does the same job
      }
    },
    true,
  );

  // The box: results as you type; Enter searches now instead of leaving the page.

  function results(root) {
    return root.querySelector(".link-results");
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

  // Picking a job reveals the three kinds (gate 7.40).
  function pick(root, job) {
    const box = clear(root);
    const taskId = root.dataset.taskId;
    const form = document.createElement("form");
    form.method = "post";
    form.action = "/tasks/" + taskId + "/links";
    form.className = "link-kinds link-kinds-picked";
    const heading = document.createElement("p");
    heading.className = "link-picked";
    heading.textContent = "How does “" + job.title + "” go with this job?";
    form.appendChild(heading);
    const other = document.createElement("input");
    other.type = "hidden";
    other.name = "other_id";
    other.value = String(job.id);
    form.appendChild(other);
    for (const [value, label] of [
      ["first", "That one first"],
      ["then", "This one first"],
      ["related", "Related"],
    ]) {
      const b = document.createElement("button");
      b.type = "submit";
      b.className = "mini";
      b.name = "kind";
      b.value = value;
      b.textContent = label;
      form.appendChild(b);
    }
    box.appendChild(form);
    form.querySelector("button").focus();
  }

  async function search(root) {
    const input = root.querySelector(".link-search input");
    const query = input.value;
    const mine = ++seq;
    if (query.trim().length < 2) {
      clear(root);
      return;
    }
    let data;
    try {
      const res = await fetch(
        "/tasks/search.json?q=" +
          encodeURIComponent(query) +
          "&exclude=" +
          encodeURIComponent(root.dataset.taskId),
      );
      if (!res.ok) return;
      data = await res.json();
    } catch (err) {
      return;
    }
    if (mine !== seq) return; // a newer keystroke has moved on
    const box = clear(root);
    const linked = new Set(
      Array.from(root.querySelectorAll(".link-line")).map((li) =>
        li.querySelector(".link-line-title").getAttribute("href"),
      ),
    );
    const jobs = data.results.filter((r) => !linked.has(r.url));
    if (jobs.length === 0) {
      box.appendChild(
        line("No other job matches “" + query + "”.", "links-empty"),
      );
      return;
    }
    const ul = document.createElement("ul");
    ul.className = "link-matches";
    for (const job of jobs) {
      const li = document.createElement("li");
      li.className = "link-match";
      const button = document.createElement("button");
      button.type = "button";
      button.className = "link-match-pick";
      const title = document.createElement("span");
      title.className = "link-match-title";
      // The server sends the title escaped, with the words marked: read it back as text.
      title.textContent = new DOMParser().parseFromString(
        job.title,
        "text/html",
      ).body.textContent;
      const column = document.createElement("span");
      column.className = "link-match-column";
      column.textContent = job.column + (job.finished ? " · Finished" : "");
      button.appendChild(title);
      button.appendChild(document.createTextNode(" "));
      button.appendChild(column);
      button.addEventListener("click", () =>
        pick(root, { id: job.id, title: title.textContent }),
      );
      li.appendChild(button);
      ul.appendChild(li);
    }
    box.appendChild(ul);
  }

  document.addEventListener("input", (event) => {
    const input = event.target;
    if (!(input instanceof HTMLInputElement) || !input.closest(".link-search"))
      return;
    const root = input.closest(".task-links");
    clearTimeout(timer);
    timer = setTimeout(() => search(root), 250);
  });
})();
