// Drag-and-drop on the Board (SPEC gate 2.04/2.05), on top of the
// button-driven moves P2-02 already built — dragging is an additional
// way to do the same thing, not a replacement, so this never needs to
// be keyboard-accessible on its own (the buttons already are).
document.addEventListener('DOMContentLoaded', initSortable)
// The board's own fragment can be replaced from underneath it by the
// shared refresh mechanism (SPEC gate 2.10) when another computer's
// change arrives — SortableJS's bindings don't survive that, so it
// needs re-creating on the fresh DOM nodes exactly like it does after a
// drag's own drop re-render.
document.addEventListener('refresh:applied', initSortable)

function initSortable() {
  document.querySelectorAll('.task-card-list').forEach((list) => {
    Sortable.create(list, {
      group: 'tasks',
      animation: 150,
      // Clicking a move button or the "Move to…" select inside a card
      // must never be mistaken for the start of a drag.
      filter: 'button, select',
      preventOnFilter: false,
      // Marks a drag in progress via the same body attribute
      // refresh.js checks before ever swapping the board's DOM out
      // from under an active gesture (SPEC gate 2.10's "never wiped
      // out").
      onStart: () => { document.body.dataset.refreshBusy = '1' },
      onEnd: (evt) => {
        delete document.body.dataset.refreshBusy
        handleDrop(evt.item)
      },
    })
  })
}

// After SortableJS moves the card's own DOM element to its dropped
// spot, its new neighbours (if any) are exactly the id the move
// endpoint needs to land it in the same place server-side (SPEC B4:
// "one of before_id, after_id or to_bottom").
function computeMoveParams(item) {
  const next = item.nextElementSibling
  const prev = item.previousElementSibling
  if (next) return { before_id: next.dataset.taskId }
  if (prev) return { after_id: prev.dataset.taskId }
  return { to_bottom: '1' }
}

// Calls the move endpoint, then re-renders the whole board from its
// response (a full server-rendered page, since the endpoint redirects
// there) rather than guessing the new state client-side — positions,
// disabled move buttons and everything else stay exactly what the
// server actually computed.
async function handleDrop(item) {
  const stage = item.closest('.task-column').dataset.stage
  const taskId = item.dataset.taskId
  const params = new URLSearchParams({ stage, ...computeMoveParams(item) })

  const res = await fetch(`/tasks/${taskId}/move`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: params.toString(),
  })
  const html = await res.text()
  const newBoard = new DOMParser().parseFromString(html, 'text/html').querySelector('.task-board')
  const oldBoard = document.querySelector('.task-board')
  if (newBoard && oldBoard) {
    oldBoard.replaceWith(newBoard)
    initSortable()
  }
}


// Remove, give back and the people menu each post to their own endpoint
// from a plain form (SPEC gates 4.18-4.20), so they work with no script at
// all. With script, the board changes in place instead: post the form, then
// re-fetch the page you are on (keeping any filter) and swap in the fresh
// board — and keep the people menu open on its card, so several names can
// be picked in a row.
async function swapBoardAfter(form, submitter, menuTaskId) {
  await fetch(form.action, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams(new FormData(form, submitter)).toString(),
    redirect: 'manual',
  })
  const res = await fetch(location.pathname + location.search)
  const html = await res.text()
  const newBoard = new DOMParser().parseFromString(html, 'text/html').querySelector('.task-board')
  const oldBoard = document.querySelector('.task-board')
  if (!newBoard || !oldBoard) throw new Error('no board in the response')
  oldBoard.replaceWith(newBoard)
  initSortable()
  if (menuTaskId) {
    const trigger = document.querySelector(`.task-card[data-task-id="${menuTaskId}"] .people-menu-trigger`)
    if (trigger) await openPeopleMenu(trigger)
  }
}

document.addEventListener('submit', async (event) => {
  const form = event.target
  if (!(form instanceof HTMLFormElement) || !form.classList.contains('board-action')) return
  if (!form.closest('.task-board')) return
  event.preventDefault()
  const card = form.closest('.task-card')
  const menuTaskId = form.closest('.people-menu') && card ? card.dataset.taskId : null
  try {
    await swapBoardAfter(form, event.submitter, menuTaskId)
  } catch (err) {
    form.submit() // the plain form does the same job
  }
})

// The people circles are a link to a page listing everyone. With script the
// same list is fetched as a fragment and shown as a menu beside the circles;
// clicking them again, pressing Escape, or clicking elsewhere closes it and
// puts the keyboard back on the circles.
function closePeopleMenus(exceptTrigger) {
  document.querySelectorAll('.task-board .people-menu-panel').forEach((panel) => {
    const holder = panel.closest('.people-menu')
    const trigger = holder && holder.querySelector('.people-menu-trigger')
    if (trigger === exceptTrigger) return
    panel.remove()
    if (trigger) trigger.setAttribute('aria-expanded', 'false')
  })
}

async function openPeopleMenu(trigger) {
  closePeopleMenus(trigger)
  const holder = trigger.closest('.people-menu')
  if (!holder) return
  const path = new URL(trigger.getAttribute('href'), location.href).pathname
  const res = await fetch(path + '?fragment=1')
  if (!res.ok) throw new Error('could not load the people menu')
  holder.querySelectorAll('.people-menu-panel').forEach((p) => p.remove())
  holder.insertAdjacentHTML('beforeend', await res.text())
  trigger.setAttribute('aria-expanded', 'true')
  const first = holder.querySelector('.people-menu-item')
  if (first) first.focus()
}

document.addEventListener('click', async (event) => {
  const target = event.target instanceof Element ? event.target : null
  if (!target) return
  const trigger = target.closest('.task-board .people-menu-trigger')
  if (trigger) {
    if (event.metaKey || event.ctrlKey || event.shiftKey) return // let "open in a new tab" through
    event.preventDefault()
    if (trigger.getAttribute('aria-expanded') === 'true') {
      closePeopleMenus(null)
    } else {
      try {
        await openPeopleMenu(trigger)
      } catch (err) {
        window.location.href = trigger.getAttribute('href') // the page does the same job
      }
    }
    return
  }
  if (!target.closest('.people-menu-panel')) closePeopleMenus(null)
})

document.addEventListener('keydown', (event) => {
  if (event.key !== 'Escape') return
  const open = document.querySelector('.task-board .people-menu-panel')
  if (!open) return
  const trigger = open.closest('.people-menu').querySelector('.people-menu-trigger')
  closePeopleMenus(null)
  if (trigger) trigger.focus()
})
