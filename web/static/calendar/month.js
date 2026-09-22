// A blank day starts an event (SPEC B12.4, D-84, gates 6.21-6.23). Nothing
// new on the server: the day number is already a link to
// /calendar/new?date=, this just makes the whole cell (not only the small
// day number) follow it, on a click anywhere that isn't already a link
// (an event or job chip).
document.addEventListener('click', (e) => {
  const cell = e.target.closest('td.calendar-day')
  if (!cell) return
  if (e.target.closest('a')) return // a chip, or the day number itself
  const link = cell.querySelector('.calendar-day-number')
  if (link) link.click()
})
