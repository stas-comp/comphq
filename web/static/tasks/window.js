// The task window (SPEC B9.7, gates 4.22-4.29). One <dialog> over the Board
// with three states: new (the form for a task that doesn't exist yet),
// reading (a task laid out as text, with its History behind an expander) and
// editing (the same form, filled in). + Add task and a card's title are
// ordinary links, to the new-task page and to the task's own page; this
// script only intercepts a plain click on them, fetches what they point to
// as a fragment and shows it here. If script doesn't run, or anything fails,
// the link simply goes to the page, which does the same job.
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
    if (dirty && !window.confirm('Leave without saving?')) return false
    dialog.close()
    return true
  }

  // Draws a fragment in the window. The fragment names its own heading
  // (data-window-title); a form fragment has none, so it keeps "Add task".
  function show(state, html, fallbackTitle, trigger) {
    if (trigger) opener = trigger
    body.innerHTML = html
    body.dataset.state = state
    const named = body.querySelector('[data-window-title]')
    heading.textContent = named ? named.dataset.windowTitle : fallbackTitle
    dirty = false
    if (!dialog.open) dialog.showModal()
    // The Steps list has fields of its own; the keyboard lands on the window's
    // own first field, or its heading, never on a step.
    const first = Array.from(body.querySelectorAll('[autofocus], input:not([type="hidden"]), select, textarea')).find(
      (el) => !el.closest('.task-steps'),
    )
    if (first) first.focus()
    else heading.focus()
  }

  async function fetchFragment(url) {
    const res = await fetch(url)
    if (!res.ok) throw new Error('could not load ' + url)
    return res.text()
  }

  async function openNewTask(link) {
    try {
      show('new', await fetchFragment('/tasks/new?fragment=1'), 'Add task', link)
    } catch (err) {
      window.location.href = link.href // the page does the same job
    }
  }

  async function openTask(id, link) {
    try {
      show('reading', await fetchFragment('/tasks/' + id + '?fragment=1'), 'Task', link)
    } catch (err) {
      window.location.href = link.href // the task's own page does the same job
    }
  }

  async function readTask(id) {
    show('reading', await fetchFragment('/tasks/' + id + '?fragment=1'), 'Task')
  }

  async function editTask(id) {
    show('editing', await fetchFragment('/tasks/' + id + '?fragment=1&mode=edit'), 'Edit task')
  }

  // Refreshes the piece of the page the screen marks as refreshable (the
  // Board, My jobs or Team), so a saved change shows without a reload.
  async function refreshPage() {
    const marker = document.querySelector('[data-refresh-fragment-selector]')
    if (!marker) throw new Error('nothing to refresh')
    const selector = marker.dataset.refreshFragmentSelector
    const res = await fetch(location.pathname + location.search)
    const fresh = new DOMParser().parseFromString(await res.text(), 'text/html').querySelector(selector)
    const old = document.querySelector(selector)
    if (!fresh || !old) throw new Error('nothing to refresh')
    old.replaceWith(fresh)
    document.dispatchEvent(new CustomEvent('refresh:applied')) // the screen's own drag setup listens
  }

  const plainClick = (event) => event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey

  // + Add task, and a card's title: a plain click opens the window; anything
  // else (a new tab, a modified click) is left to the link.
  document.addEventListener('click', (event) => {
    if (!(event.target instanceof Element) || !plainClick(event)) return
    const add = event.target.closest('a[href="/tasks/new"]')
    if (add) {
      event.preventDefault()
      openNewTask(add)
      return
    }
    const title = event.target.closest('.task-card-title a')
    if (title) {
      const match = /^\/tasks\/(\d+)$/.exec(new URL(title.href).pathname)
      if (match) {
        event.preventDefault()
        openTask(match[1], title)
      }
    }
  })

  // Anything typed makes the window "unsaved" — except in the Steps list,
  // whose steps are saved as they are used (steps.js), not by Save.
  body.addEventListener('input', (event) => {
    if (event.target instanceof Element && event.target.closest('.task-steps')) return
    dirty = true
  })

  // A step changed in the window changes the card behind it (its 3/7), so the
  // Board, My jobs or Team refreshes in place, the same as after a save.
  document.addEventListener('steps:changed', () => {
    refreshPage().catch(() => {})
  })

  // A drag that starts inside the window (selecting text) and ends on the
  // backdrop must not close it, so the press has to start on the backdrop too.
  let pressedOnBackdrop = false
  dialog.addEventListener('mousedown', (event) => {
    pressedOnBackdrop = event.target === dialog
  })

  const taskIdOf = () => {
    const holder = body.querySelector('[data-task-id]')
    return holder ? holder.dataset.taskId : null
  }

  dialog.addEventListener('click', async (event) => {
    if (event.target === dialog) {
      if (pressedOnBackdrop) requestClose()
      return
    }
    const hook = event.target instanceof Element ? event.target.closest('[data-hook]') : null
    if (!hook) return
    switch (hook.dataset.hook) {
      case 'window-close':
        requestClose()
        break
      case 'window-edit': // reading -> editing
        await editTask(taskIdOf())
        break
      case 'window-cancel-edit': // editing -> reading, asking first if something was typed
        if (dirty && !window.confirm('Leave without saving?')) break
        await readTask(taskIdOf())
        break
    }
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

  // Saving: the form posts as a fragment. A saved task is 204: a new task
  // closes the window; an edited one shows the task again, reading, with its
  // History updated. Either way the board is refreshed in place. A refused
  // save is 422 with the form again, message and typed values included, and
  // the window stays.
  body.addEventListener('submit', async (event) => {
    const form = event.target
    if (!(form instanceof HTMLFormElement)) return
    event.preventDefault()
    const state = body.dataset.state
    const id = taskIdOf()
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
      try {
        if (state === 'editing' && id) await readTask(id)
        else dialog.close()
        await refreshPage()
      } catch (err) {
        window.location.reload() // saved; just show the fresh page
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
