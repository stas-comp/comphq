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
