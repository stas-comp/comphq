// The date calendar (SPEC B12.2, D-81, gates 6.10-6.17). Every input with
// class field-date gets one, attached lazily and delegated from the
// document (not bound once at load), so it also covers ones the task
// window adds later. The typed box stays the real field (D-61, P6-02):
// this only writes into it, the same text the no-JavaScript form posts.
;(function () {
  // The calendar mark (theme.css) only shows once this has run (gate 6.17).
  document.body.classList.add('has-datepicker')

  const DAY_ABBR = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
  const DAY_FULL = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
  const MONTH_FULL = [
    'January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December',
  ]

  let openInput = null // the input currently showing a popover, or null
  let popover = null // the popover element, or null
  let viewYear = 0
  let viewMonth = 0 // 0-11
  let activeDate = null // the Date the grid's roving tabindex currently sits on

  // ---- "Today", the same as the server's (SPEC B4, P6-03) ----------------

  function serverToday() {
    const raw = document.body.dataset.today
    const d = raw && isoToDate(raw)
    return d && !isNaN(d) ? d : new Date()
  }

  function isoToDate(iso) {
    const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso)
    if (!m) return null
    return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]))
  }

  function pad2(n) {
    return String(n).padStart(2, '0')
  }

  function formatDayFirst(d) {
    return pad2(d.getDate()) + '/' + pad2(d.getMonth() + 1) + '/' + d.getFullYear()
  }

  function sameDate(a, b) {
    return !!a && !!b && a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
  }

  // Saturday a briefing dated `today` is for: today if it's a Saturday,
  // otherwise the coming one (SPEC A8, internal/briefing.SaturdayFor).
  function saturdayFor(today) {
    const days = (6 - today.getDay() + 7) % 7
    const d = new Date(today)
    d.setDate(d.getDate() + days)
    return d
  }

  // ---- Parsing/formatting, mirroring internal/app/format (D-82) ---------
  // Two places, one rule: this must read exactly what ParseDayFirstNear
  // reads, so the calendar and the server never disagree about what "25/9"
  // means.

  function realDate(year, month, day) {
    const d = new Date(year, month, day)
    if (d.getFullYear() !== year || d.getMonth() !== month || d.getDate() !== day) return null
    return d
  }

  function parseDayFirstNear(s, today) {
    s = s.trim()
    let m = /^(\d{1,2})[/.-](\d{1,2})[/.-](\d{4})$/.exec(s)
    if (m) return realDate(Number(m[3]), Number(m[2]) - 1, Number(m[1]))

    m = /^(\d{1,2})[/.-](\d{1,2})$/.exec(s)
    if (!m) return null
    const day = Number(m[1])
    const month = Number(m[2]) - 1
    let best = null
    let bestDiff = Infinity
    for (const year of [today.getFullYear() - 1, today.getFullYear(), today.getFullYear() + 1]) {
      const d = realDate(year, month, day)
      if (!d) continue
      const diff = Math.abs(d.getTime() - today.getTime())
      if (diff < bestDiff) {
        best = d
        bestDiff = diff
      }
    }
    return best
  }

  // ---- The month grid, Sunday first (SPEC B12.3, D-83) -------------------

  function sundayOnOrBefore(d) {
    const x = new Date(d)
    x.setDate(x.getDate() - x.getDay())
    return x
  }

  function saturdayOnOrAfter(d) {
    const x = new Date(d)
    x.setDate(x.getDate() + ((6 - x.getDay() + 7) % 7))
    return x
  }

  // ---- Finding and pairing fields -----------------------------------

  function fieldsOf(popoverEl) {
    return {
      head: popoverEl.querySelector('.datepicker-month'),
      grid: popoverEl.querySelector('.datepicker-grid'),
      prev: popoverEl.querySelector('.datepicker-prev'),
      next: popoverEl.querySelector('.datepicker-next'),
      // Scoped to .datepicker-foot: a grid cell for today's date reuses
      // the .datepicker-today class name (theme.css styles both the same
      // way), and an unscoped query would find whichever comes first.
      today: popoverEl.querySelector('.datepicker-foot .datepicker-today'),
      saturday: popoverEl.querySelector('.datepicker-foot .datepicker-saturday'),
    }
  }

  // A box with data-date-after="<id>" opens, when empty, on the month of
  // the box it names (gate 6.15: event end/until follow the start date).
  function pairedDate(input) {
    const afterId = input.dataset.dateAfter
    if (!afterId) return null
    const other = document.getElementById(afterId)
    if (!other) return null
    return parseDayFirstNear(other.value, serverToday())
  }

  // ---- Building the popover ------------------------------------------

  function buildPopover() {
    const el = document.createElement('div')
    el.className = 'datepicker'
    el.setAttribute('role', 'dialog')
    el.setAttribute('aria-modal', 'false')
    el.setAttribute('aria-label', 'Choose a date')
    const monthId = 'datepicker-month-' + Math.random().toString(36).slice(2, 8)
    el.innerHTML =
      '<div class="datepicker-head">' +
      '<button type="button" class="datepicker-nav datepicker-prev" aria-label="Previous month">&#8249;</button>' +
      '<div class="datepicker-month" id="' + monthId + '" aria-live="polite"></div>' +
      '<button type="button" class="datepicker-nav datepicker-next" aria-label="Next month">&#8250;</button>' +
      '</div>' +
      '<table class="datepicker-grid" role="grid" aria-labelledby="' + monthId + '">' +
      '<thead><tr>' + DAY_ABBR.map((d) => '<th scope="col" abbr="' + DAY_FULL[DAY_ABBR.indexOf(d)] + '">' + d + '</th>').join('') + '</tr></thead>' +
      '<tbody></tbody>' +
      '</table>' +
      '<div class="datepicker-foot">' +
      '<button type="button" class="datepicker-today">Today</button>' +
      '<button type="button" class="datepicker-saturday">This Saturday</button>' +
      '</div>'
    return el
  }

  function renderGrid(input) {
    const { head, grid } = fieldsOf(popover)
    head.textContent = MONTH_FULL[viewMonth] + ' ' + viewYear
    const today = serverToday()
    const selected = parseDayFirstNear(input.value, today)
    const firstOfMonth = new Date(viewYear, viewMonth, 1)
    const gridStart = sundayOnOrBefore(firstOfMonth)
    const lastOfMonth = new Date(viewYear, viewMonth + 1, 0)
    const gridEnd = saturdayOnOrAfter(lastOfMonth)

    const tbody = grid.querySelector('tbody')
    tbody.innerHTML = ''
    let cursor = new Date(gridStart)
    let activeCell = null
    while (cursor <= gridEnd) {
      const row = document.createElement('tr')
      for (let col = 0; col < 7; col++) {
        const cell = document.createElement('td')
        cell.setAttribute('role', 'gridcell')
        cell.className = 'datepicker-day'
        if (col === 6) cell.classList.add('datepicker-saturday-col')
        if (cursor.getMonth() !== viewMonth) cell.classList.add('datepicker-outside')
        if (sameDate(cursor, today)) cell.classList.add('datepicker-today')
        if (sameDate(cursor, selected)) {
          cell.classList.add('datepicker-selected')
          cell.setAttribute('aria-selected', 'true')
        }
        cell.textContent = String(cursor.getDate())
        cell.setAttribute('aria-label', DAY_FULL[cursor.getDay()] + ' ' + cursor.getDate() + ' ' + MONTH_FULL[cursor.getMonth()] + ' ' + cursor.getFullYear())
        cell.dataset.date = cursor.getFullYear() + '-' + pad2(cursor.getMonth() + 1) + '-' + pad2(cursor.getDate())
        cell.tabIndex = -1
        row.appendChild(cell)

        const isTheActiveOne = activeDate ? sameDate(cursor, activeDate) : selected ? sameDate(cursor, selected) : sameDate(cursor, today) && cursor.getMonth() === viewMonth
        if (isTheActiveOne) activeCell = cell

        cursor = new Date(cursor)
        cursor.setDate(cursor.getDate() + 1)
      }
      tbody.appendChild(row)
    }
    if (!activeCell) {
      // The active day fell outside this month (e.g. Page Up/Down): fall
      // back to the 1st, kept in bounds by the browser's own Date maths.
      activeCell = tbody.querySelector('td:not(.datepicker-outside)')
    }
    if (activeCell) activeCell.tabIndex = 0
    return activeCell
  }

  function moveMonth(delta) {
    viewMonth += delta
    while (viewMonth < 0) {
      viewMonth += 12
      viewYear--
    }
    while (viewMonth > 11) {
      viewMonth -= 12
      viewYear++
    }
  }

  // ---- Positioning (gate 6.10, 6.16: never cut off, never widens the
  // page). Native <dialog> content is in the browser's own "top layer":
  // a popover appended to document.body would render behind an open task
  // window regardless of z-index, so it's appended inside the dialog
  // instead when the field lives in one. ----------------------------------

  function place(input) {
    const dialog = input.closest('dialog[open]')
    const container = dialog || document.body
    // appendChild on a node that is already the last child still removes
    // and reinserts it — harmless on its own, but if a grid cell inside
    // currently has focus (arrow-key navigation, PLAN P6-03 gate 6.16),
    // that reinsertion silently blurs it. Only move the node when it
    // isn't already exactly where it belongs.
    if (popover.parentNode !== container) {
      container.appendChild(popover)
    }
    const box = input.getBoundingClientRect()
    const pw = popover.offsetWidth
    const ph = popover.offsetHeight
    let top = box.bottom + 4
    if (top + ph > window.innerHeight && box.top - ph - 4 >= 0) {
      top = box.top - ph - 4
    }
    let left = box.left
    if (left + pw > window.innerWidth - 8) left = Math.max(8, window.innerWidth - pw - 8)
    popover.style.position = 'fixed'
    popover.style.top = Math.max(4, top) + 'px'
    popover.style.left = left + 'px'
  }

  // ---- Opening, closing, following typing --------------------------------

  // Set right before this module moves focus itself (after a pick, after
  // Escape): the resulting focusin on the same input must not reopen the
  // popover it was just told to close (gate 6.12: picking a day "closes
  // the calendar").
  let suppressOpen = false

  function open(input) {
    if (suppressOpen) return
    if (openInput === input) return
    close()
    openInput = input
    popover = buildPopover()
    popover.style.visibility = 'hidden'
    const base = parseDayFirstNear(input.value, serverToday()) || pairedDate(input) || serverToday()
    viewYear = base.getFullYear()
    viewMonth = base.getMonth()
    activeDate = null
    ;(input.closest('dialog[open]') || document.body).appendChild(popover)
    renderGrid(input)
    wireEvents(input)
    place(input)
    popover.style.visibility = ''
  }

  function close() {
    if (!popover) return
    popover.remove()
    popover = null
    openInput = null
    activeDate = null
  }

  function refresh(input) {
    if (popover !== null && openInput === input) {
      const parsed = parseDayFirstNear(input.value, serverToday())
      if (parsed) {
        viewYear = parsed.getFullYear()
        viewMonth = parsed.getMonth()
        activeDate = parsed
      }
      renderGrid(input)
      place(input)
    }
  }

  function pick(input, date) {
    input.value = formatDayFirst(date)
    input.dispatchEvent(new Event('input', { bubbles: true }))
    input.dispatchEvent(new Event('change', { bubbles: true }))
    close()
    suppressOpen = true
    input.focus()
    suppressOpen = false
  }

  // Leaving the box rewrites a parseable value to the full dd/mm/yyyy
  // (gate 6.18), same as the server, so the chosen year is visible before
  // saving. An unparseable value is left exactly as typed.
  function commit(input) {
    const parsed = parseDayFirstNear(input.value, serverToday())
    if (parsed) input.value = formatDayFirst(parsed)
  }

  // ---- Wiring one popover's controls -------------------------------------

  function wireEvents(input) {
    const { prev, next, today, saturday, grid } = fieldsOf(popover)
    prev.addEventListener('click', () => {
      moveMonth(-1)
      activeDate = null
      const cell = renderGrid(input)
      if (cell) cell.focus()
      place(input)
    })
    next.addEventListener('click', () => {
      moveMonth(1)
      activeDate = null
      const cell = renderGrid(input)
      if (cell) cell.focus()
      place(input)
    })
    today.addEventListener('click', () => pick(input, serverToday()))
    saturday.addEventListener('click', () => pick(input, saturdayFor(serverToday())))

    grid.addEventListener('click', (e) => {
      const cell = e.target.closest('.datepicker-day')
      if (!cell) return
      pick(input, isoToDate(cell.dataset.date))
    })

    grid.addEventListener('keydown', (e) => onGridKeydown(e, input))

    // Mousedown, not click: firing before the input's own blur keeps the
    // popover from closing itself out from under a click on it.
    document.addEventListener('mousedown', onOutsideMouseDown, true)
  }

  function onOutsideMouseDown(e) {
    if (!popover || !openInput) return
    if (popover.contains(e.target) || e.target === openInput) return
    document.removeEventListener('mousedown', onOutsideMouseDown, true)
    commit(openInput)
    close()
  }

  function onGridKeydown(e, input) {
    const current = e.target.closest('.datepicker-day')
    if (!current) return
    const date = isoToDate(current.dataset.date)
    let target = null
    switch (e.key) {
      case 'ArrowLeft':
        target = addDays(date, -1)
        break
      case 'ArrowRight':
        target = addDays(date, 1)
        break
      case 'ArrowUp':
        target = addDays(date, -7)
        break
      case 'ArrowDown':
        target = addDays(date, 7)
        break
      case 'Home':
        target = sundayOnOrBefore(date)
        break
      case 'End':
        target = saturdayOnOrAfter(date)
        break
      case 'PageUp':
        target = addMonths(date, -1)
        break
      case 'PageDown':
        target = addMonths(date, 1)
        break
      case 'Enter':
      case ' ':
        e.preventDefault()
        pick(input, date)
        return
      case 'Escape':
        e.preventDefault()
        commit(input)
        close()
        suppressOpen = true
        input.focus()
        suppressOpen = false
        return
      default:
        return
    }
    e.preventDefault()
    activeDate = target
    if (target.getMonth() !== viewMonth || target.getFullYear() !== viewYear) {
      viewYear = target.getFullYear()
      viewMonth = target.getMonth()
    }
    const cell = renderGrid(input)
    if (cell) cell.focus()
    place(input)
  }

  function addDays(d, n) {
    const x = new Date(d)
    x.setDate(x.getDate() + n)
    return x
  }

  function addMonths(d, n) {
    // Clamp to the last real day of the target month (e.g. 31 Jan + 1
    // month lands on the last day of February, never rolling into March).
    const target = new Date(d.getFullYear(), d.getMonth() + n, 1)
    const lastDay = new Date(target.getFullYear(), target.getMonth() + 1, 0).getDate()
    target.setDate(Math.min(d.getDate(), lastDay))
    return target
  }

  // ---- Attaching to every input.field-date, including ones added later --

  document.addEventListener('focusin', (e) => {
    const input = e.target.closest && e.target.closest('input.field-date')
    if (input) open(input)
  })

  // Down arrow, from the box itself, moves focus into the grid (SPEC B12.2).
  document.addEventListener('keydown', (e) => {
    const input = e.target.closest && e.target.closest('input.field-date')
    if (!input || openInput !== input || e.key !== 'ArrowDown') return
    if (e.target.closest('.datepicker')) return
    e.preventDefault()
    const cell = popover && popover.querySelector('.datepicker-day[tabindex="0"]')
    if (cell) cell.focus()
  })

  document.addEventListener('keydown', (e) => {
    const input = e.target.closest && e.target.closest('input.field-date')
    if (!input || openInput !== input || e.target !== input) return
    if (e.key === 'Escape') {
      close()
    }
  })

  // While the box is being typed in, the grid follows (gate 6.14).
  document.addEventListener('input', (e) => {
    const input = e.target.closest && e.target.closest('input.field-date')
    if (input) refresh(input)
  })

  // Leaving the box (not into the popover itself) commits and closes.
  document.addEventListener(
    'focusout',
    (e) => {
      const input = e.target.closest && e.target.closest('input.field-date')
      if (!input || openInput !== input) return
      const next = e.relatedTarget
      if (next && popover && popover.contains(next)) return
      commit(input)
      close()
    },
    true,
  )
})()
