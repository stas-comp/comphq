// Steps inside a job (SPEC B10, gates 5.01-5.13), layered over plain forms.
//
// Every step control in web/templates/tasks/steps.html is an ordinary form
// (add, tick, rename, remove, undo, reorder) that works with no script at
// all (gate 5.08). This script only intercepts them: it posts the same form
// with fragment=1 and swaps in the list the server sends back, so nothing
// reloads. If anything fails, the plain form does the same job.
//
// The list is one partial, drawn by the task window and by the task's own
// page alike, so this one script serves both.
;(function () {
  const UNDO_MS = 10000
  const POLL_MS = 60000

  let inFlight = 0
  let knownVersion = null
  let undoTimer = null

  const section = () => document.querySelector('.task-steps')

  // ---- Swapping the list in ----------------------------------------------

  function parse(html) {
    const fresh = new DOMParser().parseFromString(html, 'text/html').querySelector('.task-steps')
    return fresh ? document.importNode(fresh, true) : null
  }

  // Puts a fresh list where the current one is and hands the keyboard on to
  // wherever `focusFor` says; a swap that lost the user's place would make
  // every tick a small disaster for anyone not using a mouse. Whatever has
  // been typed in the Add a step box since the request left stays there, so
  // a list can be typed straight through without waiting on each step.
  function swap(fresh, focusFor, selectText) {
    const old = section()
    if (!old) return
    const typed = addBox(old)
    const kept = typed && !fresh.querySelector('.steps-notice') ? typed.value : null
    old.replaceWith(fresh)
    enhance(fresh)
    const box = addBox(fresh)
    if (box && kept) box.value = kept
    const target = focusFor && focusFor(fresh)
    if (target) {
      target.focus()
      if (target.matches('input[type="text"]')) {
        if (selectText) target.select()
        else target.setSelectionRange(target.value.length, target.value.length)
      }
    }
    document.dispatchEvent(new CustomEvent('steps:changed'))
  }

  // Posts to a step route as a fragment request and returns the fresh list.
  async function send(url, data) {
    data.set('fragment', '1')
    const res = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: data.toString(),
      redirect: 'manual',
    })
    // 200 worked and 422 was refused with the words to show; both carry the list.
    if (res.status !== 200 && res.status !== 422) throw new Error('unexpected ' + res.status)
    const fresh = parse(await res.text())
    if (!fresh) throw new Error('no list in the response')
    return fresh
  }

  async function load(taskId, query) {
    const res = await fetch('/tasks/' + taskId + '/steps?fragment=1' + (query || ''))
    if (!res.ok) throw new Error('could not load the steps')
    const fresh = parse(await res.text())
    if (!fresh) throw new Error('no list in the response')
    return fresh
  }

  // ---- Where the keyboard goes afterwards --------------------------------

  const byId = (root, id) => root.querySelector('#' + id)
  const tickOf = (root, stepId) => byId(root, 'step-check-' + stepId)
  const addBox = (root) => root.querySelector('.step-add input[type="text"]')

  function intentOf(form) {
    const action = new URL(form.action, location.href).pathname
    const stepId = (/\/steps\/(\d+)/.exec(action) || [])[1]
    if (form.classList.contains('step-add')) return { kind: 'add' }
    if (form.classList.contains('step-tick')) return { kind: 'tick', stepId }
    if (form.classList.contains('step-rename')) return { kind: 'rename', stepId }
    if (form.classList.contains('steps-undo')) return { kind: 'restore', stepId }
    if (/\/remove$/.test(action)) {
      const row = form.closest('.step')
      const rows = Array.from(form.closest('.steps-list').children)
      return { kind: 'remove', stepId, index: rows.indexOf(row) }
    }
    if (/\/move$/.test(action)) {
      const dir = form.querySelector('[name="direction"]')
      return { kind: 'move', stepId, direction: dir ? dir.value : '' }
    }
    return { kind: 'other', stepId }
  }

  function focusAfter(intent, refused) {
    return (root) => {
      switch (intent.kind) {
        case 'add':
          return addBox(root)
        case 'tick':
        case 'restore':
          return tickOf(root, intent.stepId)
        case 'rename': {
          if (refused) return root.querySelector('.step-rename input[type="text"]') || tickOf(root, intent.stepId)
          const link = root.querySelector('#step-' + intent.stepId + ' .step-rename-link')
          return link || tickOf(root, intent.stepId)
        }
        case 'remove': {
          const rows = root.querySelectorAll('.step')
          const next = rows[Math.min(intent.index, rows.length - 1)]
          return next ? next.querySelector('.check') : addBox(root)
        }
        case 'move': {
          const forms = root.querySelectorAll('#step-' + intent.stepId + ' .step-actions form')
          const enabled = Array.from(forms).filter((f) => f.querySelector('button:not(:disabled)'))
          const same = enabled.find((f) => f.querySelector('[name="direction"]').value === intent.direction)
          const pick = same || enabled[0]
          return pick ? pick.querySelector('button') : tickOf(root, intent.stepId)
        }
      }
      return null
    }
  }

  // ---- Forms: add, tick, rename, remove, undo, move ----------------------

  // One request at a time, in the order the person made them: a step typed
  // and entered while the last one is still on its way waits its turn.
  let queue = Promise.resolve()
  function enqueue(job) {
    inFlight++
    queue = queue.then(job).finally(() => inFlight--)
  }

  document.addEventListener(
    'submit',
    (event) => {
      const form = event.target
      if (!(form instanceof HTMLFormElement) || !form.closest('.task-steps')) return
      // The task window has a save handler of its own for its forms; steps are
      // saved as they are used, never as part of "Save", so it must not see these.
      event.stopPropagation()
      event.preventDefault()
      const intent = intentOf(form)
      const data = new URLSearchParams(new FormData(form, event.submitter))
      const url = form.action
      // The words are on their way: the box is ready for the next step at once.
      if (intent.kind === 'add') addBox(form.closest('.task-steps')).value = ''
      enqueue(async () => {
        try {
          const fresh = await send(url, data)
          const refused = !!fresh.querySelector('.steps-notice')
          swap(fresh, focusAfter(intent, refused))
        } catch (err) {
          // Nothing usable came back: the plain form does the same job, as a page.
          const marker = form.querySelector('[name="fragment"]')
          if (marker) marker.remove()
          form.submit()
        }
      })
    },
    true,
  )

  // A box ticked or unticked is saved at once; without script the same form
  // has an Update button instead.
  document.addEventListener('change', (event) => {
    const box = event.target
    if (!(box instanceof HTMLInputElement) || !box.matches('.step-tick input[type="checkbox"]')) return
    box.form.requestSubmit()
  })

  // ---- Renaming in place -------------------------------------------------

  async function openRename(link) {
    const sec = link.closest('.task-steps')
    const stepId = link.closest('.step').dataset.stepId
    try {
      const fresh = await load(sec.dataset.taskId, '&rename_step=' + stepId)
      swap(fresh, (root) => root.querySelector('#step-text-' + stepId), true)
    } catch (err) {
      window.location.href = link.href // the page's own rename form
    }
  }

  async function closeRename(sec, stepId) {
    try {
      const fresh = await load(sec.dataset.taskId)
      swap(fresh, (root) => root.querySelector('#step-' + stepId + ' .step-rename-link') || tickOf(root, stepId))
    } catch (err) {
      window.location.reload()
    }
  }

  let justDragged = false

  document.addEventListener('click', (event) => {
    if (!(event.target instanceof Element)) return
    const plain = event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey
    const rename = event.target.closest('[data-hook="step-rename"]')
    if (rename && plain) {
      event.preventDefault()
      openRename(rename)
      return
    }
    const cancel = event.target.closest('[data-hook="step-cancel-rename"]')
    if (cancel && plain) {
      event.preventDefault()
      closeRename(cancel.closest('.task-steps'), cancel.closest('.step').dataset.stepId)
      return
    }
    // The words themselves are the way in, too: click a step to change it.
    const words = event.target.closest('.step-words')
    if (words && plain && !justDragged && !window.getSelection().toString()) {
      const link = words.closest('.step').querySelector('[data-hook="step-rename"]')
      if (link) openRename(link)
    }
  })

  // Escape leaves a rename without saving it. It must not also close the task
  // window, which is what Escape means everywhere else in it.
  document.addEventListener(
    'keydown',
    (event) => {
      if (event.key !== 'Escape' || !(event.target instanceof Element)) return
      const box = event.target.closest('.step-rename input')
      if (!box) return
      event.preventDefault()
      event.stopPropagation()
      closeRename(box.closest('.task-steps'), box.closest('.step').dataset.stepId)
    },
    true,
  )

  // ---- The Undo goes after about ten seconds, or at the next action ------

  function armUndo(sec) {
    clearTimeout(undoTimer)
    if (!sec.querySelector('.steps-undo')) return
    undoTimer = setTimeout(() => {
      const undo = document.querySelector('.task-steps .steps-undo')
      if (undo) undo.remove()
    }, UNDO_MS)
  }

  // ---- Dragging ----------------------------------------------------------

  function armDrag(sec) {
    const list = sec.querySelector('.steps-list')
    if (!list || typeof Sortable === 'undefined') return
    Sortable.create(list, {
      animation: 150,
      handle: '.step-body',
      // Clicking a control must never be taken for the start of a drag.
      filter: 'button, input, a, select',
      preventOnFilter: false,
      onStart: () => {
        document.body.dataset.refreshBusy = '1'
      },
      onEnd: async (evt) => {
        delete document.body.dataset.refreshBusy
        justDragged = true
        setTimeout(() => (justDragged = false), 50)
        if (evt.oldIndex === evt.newIndex) return
        const item = evt.item
        const url = '/tasks/' + sec.dataset.taskId + '/steps/' + item.dataset.stepId + '/move'
        const position = String(evt.newIndex + 1)
        enqueue(async () => {
          try {
            const fresh = await send(url, new URLSearchParams({ position }))
            swap(fresh, (root) => tickOf(root, item.dataset.stepId))
          } catch (err) {
            window.location.reload() // saved or not, the page shows what the server has
          }
        })
      },
    })
  }

  // ---- Another computer's change ----------------------------------------

  // Left alone while somebody is in the middle of something in the list: a
  // typed step, a rename, a drag or a request on its way. The next check
  // tries again, so nothing is lost, only put off.
  function isBusy(sec) {
    if (inFlight > 0 || document.body.dataset.refreshBusy === '1') return true
    if (sec.querySelector('.step-rename')) return true
    const box = addBox(sec)
    return !!(box && box.value.trim() !== '')
  }

  async function check() {
    const sec = section()
    if (!sec || knownVersion === null) return
    let version
    try {
      const res = await fetch('/tasks/version')
      if (!res.ok) return
      version = (await res.json()).version
    } catch (err) {
      return
    }
    if (version <= knownVersion || isBusy(sec)) return
    try {
      const fresh = await load(sec.dataset.taskId)
      const current = section()
      if (!current || isBusy(current)) return
      current.replaceWith(fresh)
      enhance(fresh)
    } catch (err) {
      /* the next check tries again */
    }
  }

  setInterval(() => {
    if (document.visibilityState === 'visible') check()
  }, POLL_MS)
  window.addEventListener('focus', check)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') check()
  })

  // ---- Setting up each list as it appears --------------------------------

  function enhance(sec) {
    if (sec.dataset.enhanced) return
    sec.dataset.enhanced = '1'
    const version = Number(sec.dataset.version)
    if (!Number.isNaN(version)) knownVersion = version
    armUndo(sec)
    armDrag(sec)
  }

  function scan() {
    document.querySelectorAll('.task-steps:not([data-enhanced])').forEach(enhance)
  }

  // The task window draws its list after the page has loaded, and draws it
  // again each time a task is opened.
  new MutationObserver(scan).observe(document.documentElement, { childList: true, subtree: true })
  scan()
})()
