// My jobs drag interactions (SPEC gates 2.34, 2.35, 7.01-7.03): dragging a
// card out of Up for grabs and into Up next takes it, placed where it was
// dropped — the Take it button (a plain form, no JS needed) does the
// same without a drop position. Dragging within Up next reorders it
// (the shared To do order, gate 7.02), and dragging an idea from Ideas I'm
// on into Up next moves it to To do where it was dropped (gate 7.03); the
// move icons and the Move to To do button are plain forms that do the same.
// Mirrors team.js's own assign-by-drag pattern, just with one target list
// instead of one per person. Also
// handles gate 2.36's take conflict for the drag path (the button path
// needs no JS at all — a 409 response still renders as the response
// body, same as any other status).
document.addEventListener('DOMContentLoaded', initMyJobsSortable)
document.addEventListener('refresh:applied', initMyJobsSortable)

function initMyJobsSortable() {
  const upNext = document.querySelector('.myjobs-upnext-list')
  if (upNext) {
    comphqSortable(upNext, {
      group: { name: 'myjobs-take', pull: false },
      draggable: '.team-task', // not the "Drop a job here" box at the foot
      filter: 'button',
      preventOnFilter: false,
      onStart: () => { document.body.dataset.refreshBusy = '1' },
      onEnd: (evt) => handleMyJobsDrop(evt),
    })
  }
  const ideas = document.querySelector('.myjobs-ideas-list')
  if (ideas) {
    comphqSortable(ideas, {
      // Ideas only ever leave: into Up next, which makes them a To do job.
      group: { name: 'myjobs-take', put: false },
      sort: false,
      filter: 'button',
      preventOnFilter: false,
      onStart: () => { document.body.dataset.refreshBusy = '1' },
      onEnd: (evt) => handleMyJobsDrop(evt),
    })
  }
  const grabs = document.querySelector('.myjobs-grabs-list')
  if (grabs) {
    comphqSortable(grabs, {
      group: { name: 'myjobs-take', put: false },
      filter: 'button',
      preventOnFilter: false,
      onStart: () => { document.body.dataset.refreshBusy = '1' },
      onEnd: (evt) => handleMyJobsDrop(evt),
    })
  }
}

// Mirrors team.js's own computeLaneMoveParams: the dropped item's final
// DOM neighbours become the drop position (SPEC gate 2.34); no sibling
// means the drop didn't imply any particular position.
function computeDropParams(item) {
  // The neighbours that count are real jobs; the "Drop a job here" box has no id.
  const jobSibling = (el, dir) => {
    while (el && !el.dataset.taskId) el = el[dir]
    return el
  }
  const next = jobSibling(item.nextElementSibling, 'nextElementSibling')
  const prev = jobSibling(item.previousElementSibling, 'previousElementSibling')
  if (next) return { before_id: next.dataset.taskId }
  if (prev) return { after_id: prev.dataset.taskId }
  return {}
}

async function postTake(taskId, params) {
  const body = new URLSearchParams(params)
  return fetch(`/tasks/${taskId}/take`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: body.toString(),
  })
}

async function postMove(taskId, params) {
  return fetch(`/tasks/${taskId}/move`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({ stage: 'todo', ...params }).toString(),
    redirect: 'manual',
  })
}

async function handleMyJobsDrop(evt) {
  delete document.body.dataset.refreshBusy

  // Within Up next: put the job next to the one it was dropped beside, in
  // the shared To do order, so every other job keeps its place (gates
  // 7.01, 7.02). From Ideas I'm on into Up next: the same move, which also
  // turns the idea into a To do job (gate 7.03); with nothing beside it,
  // an idea goes to the bottom of To do. Any other drop has no defined
  // action: the refresh below restores whatever the server still says is
  // true.
  if (evt.from === evt.to && evt.to.classList.contains('myjobs-upnext-list')) {
    const params = computeDropParams(evt.item)
    if (params.before_id || params.after_id) await postMove(evt.item.dataset.taskId, params)
  } else if (evt.from.classList.contains('myjobs-ideas-list') && evt.to.classList.contains('myjobs-upnext-list')) {
    const params = computeDropParams(evt.item)
    await postMove(evt.item.dataset.taskId, params.before_id || params.after_id ? params : { to_bottom: '1' })
  }

  // Only a drag from Up for grabs into Up next takes a job. Dragging
  // within Up next itself has no defined action in this session (only
  // taking a job is wired up here, not reordering one already owned) —
  // left alone, and the refresh below restores whatever the server
  // still says is true, the same "no defined action" pattern team.js
  // already uses for its own unhandled same-lane drops.
  if (evt.from.classList.contains('myjobs-grabs-list') && evt.to.classList.contains('myjobs-upnext-list')) {
    const res = await postTake(evt.item.dataset.taskId, computeDropParams(evt.item))
    // SPEC gate 2.36: someone beat this drag to it — the response is
    // already the current My jobs page, with the exact conflict
    // message, so swap that straight in rather than fetching again.
    if (res.status === 409) {
      swapMyJobsPage(await res.text())
      return
    }
  }

  await refreshMyJobs()
}

function swapMyJobsPage(html) {
  const newPage = new DOMParser().parseFromString(html, 'text/html').querySelector('.myjobs-page')
  const oldPage = document.querySelector('.myjobs-page')
  if (newPage && oldPage) {
    oldPage.replaceWith(newPage)
    initMyJobsSortable()
  }
}

// Re-fetches the My jobs page and swaps it in, the same "server
// response is the truth" pattern team.js uses after its own drop.
async function refreshMyJobs() {
  const res = await fetch(location.pathname + location.search)
  if (!res.ok) return
  swapMyJobsPage(await res.text())
}
