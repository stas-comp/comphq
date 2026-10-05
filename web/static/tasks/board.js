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
    comphqSortable(list, {
      group: 'tasks',
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

// Calls the move endpoint, then re-renders the whole board from the
// server (the page we are on, filter included) rather than guessing the new state client-side — positions,
// disabled move buttons and everything else stay exactly what the
// server actually computed.
async function handleDrop(item) {
  const stage = item.closest('.task-column').dataset.stage
  const taskId = item.dataset.taskId
  const params = new URLSearchParams({ stage, ...computeMoveParams(item) })

  await fetch(`/tasks/${taskId}/move`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: params.toString(),
    redirect: 'manual',
  })
  // The move endpoint redirects to the bare Board, which would swap in the
  // unfiltered cards. Re-fetch the page we are on instead, query string and
  // all, so a filter in use stays applied (gate 7.14).
  await replaceBoardFromPage()
}


// Remove and give back each post to their own endpoint from a plain form
// (SPEC gates 4.18, 4.19), so they work with no script at all. With script,
// the board changes in place instead: post the form, then re-fetch the page
// you are on (keeping any filter) and swap in the fresh board. The people
// circles and their menu are in people-menu.js, shared with Team and My jobs.
//
// replaceBoardFromPage is also how the task window refreshes the board after
// it saves (window.js).
//
// The message at the bottom of the screen (toast.js) lives beside the board,
// not in it. Every swap replaces its contents with what the server sent, so it
// is empty after any action but a removal, which asks for it with the id of
// the job just removed (gates 7.15, 7.17).
async function replaceBoardFromPage(removedId) {
  let url = location.pathname + location.search
  if (removedId) url += (location.search ? '&' : '?') + 'removed=' + encodeURIComponent(removedId)
  const res = await fetch(url)
  const html = await res.text()
  const doc = new DOMParser().parseFromString(html, 'text/html')
  const newBoard = doc.querySelector('.task-board')
  const oldBoard = document.querySelector('.task-board')
  if (!newBoard || !oldBoard) throw new Error('no board in the response')
  oldBoard.replaceWith(newBoard)
  const newRegion = doc.getElementById('toast-region')
  const oldRegion = document.getElementById('toast-region')
  if (newRegion && oldRegion) oldRegion.innerHTML = newRegion.innerHTML
  if (window.comphqToast) window.comphqToast.arm()
  initSortable()
}

async function swapBoardAfter(form, submitter) {
  await fetch(form.action, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams(new FormData(form, submitter)).toString(),
    redirect: 'manual',
  })
  const removed = new URL(form.action).pathname.match(/^\/tasks\/(\d+)\/remove$/)
  await replaceBoardFromPage(removed ? removed[1] : undefined)
}

document.addEventListener('submit', async (event) => {
  const form = event.target
  if (!(form instanceof HTMLFormElement) || !form.classList.contains('board-action')) return
  if (!form.closest('.task-board')) return
  event.preventDefault()
  try {
    await swapBoardAfter(form, event.submitter)
  } catch (err) {
    form.submit() // the plain form does the same job
  }
})
