// The task window (SPEC B9.7, gates 4.22, 4.25, 4.26, 4.28). A single
// <dialog>, opened over the Board by the + Add task link. The link is an
// ordinary link to the new-task page and stays one: this script only
// intercepts a plain click on it, fetches the same form as a fragment and
// shows it here. If script doesn't run, or anything fails, the link simply
// goes to the page, which does the same job.
//
// The platform's own <dialog> gives Escape, the backdrop, the keyboard
// staying inside the window (everything behind it is inert) and, on close,
// the return of focus; this adds the close icon, the click on the backdrop,
// and the "leave without saving?" question.
;(function () {
  const dialog = document.getElementById('task-window')
  if (!dialog || typeof dialog.showModal !== 'function') return
  const body = dialog.querySelector('.task-window-body')
  const heading = dialog.querySelector('.task-window-title')

  let opener = null
  let dirty = false

  function requestClose() {
    if (dirty && !window.confirm('Leave without saving?')) return
    dialog.close()
  }

  function show(state, title, html, trigger) {
    if (trigger) opener = trigger
    heading.textContent = title
    body.innerHTML = html
    body.dataset.state = state
    dirty = false
    if (!dialog.open) dialog.showModal()
    const first = body.querySelector('[autofocus], input:not([type="hidden"]), select, textarea')
    if (first) first.focus()
  }

  async function openNewTask(link) {
    try {
      const res = await fetch('/tasks/new?fragment=1')
      if (!res.ok) throw new Error('could not load the form')
      show('new', 'Add task', await res.text(), link)
    } catch (err) {
      window.location.href = link.href // the page does the same job
    }
  }

  // + Add task: a plain click opens the window; anything else (a new tab,
  // a modified click) is left to the link.
  document.addEventListener('click', (event) => {
    const link = event.target instanceof Element ? event.target.closest('a[href="/tasks/new"]') : null
    if (!link || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
    event.preventDefault()
    openNewTask(link)
  })

  // Anything typed makes the window "unsaved".
  body.addEventListener('input', () => {
    dirty = true
  })

  // Close: the close icon and Cancel, the backdrop (a click on the dialog
  // element itself, since the inner box fills the rest), and Escape.
  // A drag that starts inside the window (selecting text) and ends on the
  // backdrop must not close it, so the press has to start on the backdrop too.
  let pressedOnBackdrop = false
  dialog.addEventListener('mousedown', (event) => {
    pressedOnBackdrop = event.target === dialog
  })
  dialog.addEventListener('click', (event) => {
    if (event.target === dialog) {
      if (pressedOnBackdrop) requestClose()
      return
    }
    if (event.target instanceof Element && event.target.closest('[data-hook="window-close"]')) requestClose()
  })
  dialog.addEventListener('cancel', (event) => {
    event.preventDefault() // Escape: ask first if there is something to lose
    requestClose()
  })
  dialog.addEventListener('close', () => {
    dirty = false
    body.innerHTML = ''
    if (opener && document.contains(opener)) opener.focus() // back on what opened it
  })

  // Saving: the form posts as a fragment. A saved task is 204: close the
  // window and refresh the board in place. A refused one is 422 with the
  // form again, message and typed values included, and the window stays.
  body.addEventListener('submit', async (event) => {
    const form = event.target
    if (!(form instanceof HTMLFormElement)) return
    event.preventDefault()
    let res
    try {
      res = await fetch(form.action, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams(new FormData(form, event.submitter)).toString(),
      })
    } catch (err) {
      // Nothing was sent: the plain form does the same job, as a page.
      const marker = form.querySelector('[name="fragment"]')
      if (marker) marker.remove()
      form.submit()
      return
    }
    if (res.status === 204) {
      dirty = false
      dialog.close()
      try {
        await replaceBoardFromPage()
      } catch (err) {
        window.location.reload() // saved; just show the fresh board
      }
      return
    }
    if (res.status === 422) {
      body.innerHTML = await res.text()
      dirty = true
      const alert = body.querySelector('[role="alert"]')
      if (alert) {
        alert.setAttribute('tabindex', '-1')
        alert.focus()
      }
      return
    }
    window.location.reload()
  })
})()
