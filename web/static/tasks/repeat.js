// The Repeats choice (SPEC B14.4, gate 8.40): the weeks box ("Show it in To do __
// weeks before it's due") belongs to Every year only, so it is shown only then.
// Without script it is always there, with its hint, and the server ignores it for
// any other choice.
//
// The form is drawn on its own pages and fetched into the task window as a
// fragment, so this is set up on load and again whenever the window draws
// something (window.js sends "window:shown").
(function () {
  function sync(select) {
    const form = select.closest("form");
    if (!form) return;
    const box = form.querySelector("[data-yearly-only]");
    if (box) box.hidden = select.value !== "yearly";
  }

  function init() {
    document.querySelectorAll("[data-repeat-select]").forEach((select) => {
      sync(select);
    });
  }

  document.addEventListener("change", (event) => {
    const select = event.target;
    if (select instanceof HTMLSelectElement && select.hasAttribute("data-repeat-select")) sync(select);
  });

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("window:shown", init);
})();
