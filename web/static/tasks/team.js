// Team view drag interactions (SPEC gates 2.28, 2.29): dragging a job
// between lanes assigns it (kept separate from the Board's own drag,
// which reorders — the same job title appearing in several lanes here
// is one task, and dragging it here changes who's on it, not its
// column or priority); dragging within one lane's Up next reorders
// that lane's slice of the shared To do order via the existing move
// endpoint (the same one the Board's own buttons already use). An idea
// (any lane's Ideas group, Unassigned's included) can be dragged into an Up
// next list (gates 7.04, 7.05): it moves to To do where it is dropped and,
// if that is another lane, is handed over as well — move first, then assign.
// The Move to To do button on every idea does the same, with no drag.
document.addEventListener('DOMContentLoaded', initTeamSortable)
document.addEventListener('refresh:applied', initTeamSortable)

function initTeamSortable() {
  document.querySelectorAll('.team-lane').forEach((lane) => {
    const upNext = lane.querySelector('.team-up-next-list')
    if (upNext) {
      comphqSortable(upNext, {
        group: 'team-assign',
        filter: 'button, select',
        preventOnFilter: false,
        onStart: () => { document.body.dataset.refreshBusy = '1' },
        onEnd: (evt) => handleTeamDrop(evt),
      })
    }
    const ideas = lane.querySelector('.team-ideas-list')
    if (ideas) {
      comphqSortable(ideas, {
        // Ideas only ever leave, into an Up next list.
        group: { name: 'team-assign', put: false },
        sort: false,
        filter: 'button, select',
        preventOnFilter: false,
        onStart: () => { document.body.dataset.refreshBusy = '1' },
        onEnd: (evt) => handleTeamDrop(evt),
      })
    }
    const working = lane.querySelector('.team-working-list')
    if (working) {
      comphqSortable(working, {
        group: 'team-assign',
        // Working on now isn't reorderable (SPEC B4 only describes
        // reordering Up next) — this only ever leaves this list to
        // another lane (an assign), never reshuffles within it.
        sort: false,
        filter: 'button, select',
        preventOnFilter: false,
        onStart: () => { document.body.dataset.refreshBusy = '1' },
        onEnd: (evt) => handleTeamDrop(evt),
      })
    }
  })
}

// Mirrors board.js's own computeMoveParams, scoped to the dropped
// item's new siblings within this same list — null (no reorder to do)
// rather than "to_bottom" when there's no sibling to land next to,
// since "bottom" here would otherwise wrongly mean the bottom of the
// whole shared To do order, not just this lane's slice of it.
function computeLaneMoveParams(item) {
  const next = item.nextElementSibling
  const prev = item.previousElementSibling
  if (next) return { before_id: next.dataset.taskId }
  if (prev) return { after_id: prev.dataset.taskId }
  return null
}

async function postMove(taskId, params) {
  const body = new URLSearchParams({ stage: 'todo', ...params })
  const res = await fetch(`/tasks/${taskId}/move`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: body.toString(),
  })
  return res.ok
}

async function postAssign(taskId, fromPersonId, toPersonId) {
  const body = new URLSearchParams({ from_person: fromPersonId, to_person: toPersonId })
  const res = await fetch(`/tasks/${taskId}/assign`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: body.toString(),
  })
  return res.ok
}

// A drop that couldn't be carried out: the screen is refreshed to how things
// really are, and says so in plain words (the text is in the page).
function showDragMessage(show) {
  const el = document.getElementById('team-drag-message')
  if (el) el.hidden = !show
}

async function handleTeamDrop(evt) {
  delete document.body.dataset.refreshBusy

  const item = evt.item
  const taskId = item.dataset.taskId
  const fromLane = evt.from.closest('.team-lane')
  const toLane = evt.to.closest('.team-lane')

  let ok = true
  if (evt.from.classList.contains('team-ideas-list')) {
    // An idea only means something dropped into an Up next list.
    if (evt.to.classList.contains('team-up-next-list')) {
      ok = await postMove(taskId, computeLaneMoveParams(item) || { to_bottom: '1' })
      if (ok && fromLane !== toLane) {
        ok = await postAssign(taskId, fromLane.dataset.personId, toLane.dataset.personId)
      }
    }
  } else if (fromLane === toLane) {
    if (evt.from === evt.to && evt.from.classList.contains('team-up-next-list')) {
      const params = computeLaneMoveParams(item)
      if (params) await postMove(taskId, params)
    }
    // Any other same-lane drop (e.g. Up next onto Working on now) has
    // no defined meaning here — left alone, and the refresh below
    // restores whatever the server still says is true.
  } else {
    ok = await postAssign(taskId, fromLane.dataset.personId, toLane.dataset.personId)
  }

  await refreshTeamLanes()
  showDragMessage(!ok)
}

// Re-fetches the Team page and swaps in just the lanes, the same
// "server response is the truth" pattern board.js uses after its own
// drop — an assign or a lane-scoped move can change which lane a task
// belongs to or how it's numbered, and only the server knows the
// result.
async function refreshTeamLanes() {
  const res = await fetch(location.pathname + location.search)
  if (!res.ok) return
  const html = await res.text()
  const newLanes = new DOMParser().parseFromString(html, 'text/html').querySelector('.team-lanes')
  const oldLanes = document.querySelector('.team-lanes')
  if (newLanes && oldLanes) {
    oldLanes.replaceWith(newLanes)
    initTeamSortable()
  }
}
