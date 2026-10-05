// A card's people circles and the short menu of names they open (SPEC gate
// 4.20 on the Board; gates 7.06-7.09 on Team and My jobs). One script for all
// three screens.
//
// The circles are a link to a small page that lists everyone with a button
// each, so assigning works with no script at all. With script the same list is
// fetched as a fragment and shown as a menu beside the circles; clicking them
// again, pressing Escape, or clicking elsewhere closes it and puts the
// keyboard back on the circles. A name that is on the job takes them off, one
// that isn't puts them on; the screen's own fragment (the Board, Team's lanes,
// My jobs) is then re-fetched, so the change shows without a reload, and the
// menu stays open for the next pick.
//
// Placement (gate 7.07): the menu is drawn against the window, not inside the
// card. Team's lanes scroll on their own (overflow-x: auto also clips
// vertically), and a menu positioned inside a lane would be cut off on the
// last card of a long column or in the right-most lane. It opens below the
// circles, or above them when there is no room, never wider than the window
// and never under the top bar (D-102).
;(function () {
  var MARGIN = 8

  function closePeopleMenus(exceptTrigger) {
    document.querySelectorAll('.people-menu-panel').forEach(function (panel) {
      var holder = panel.closest('.people-menu')
      var trigger = holder && holder.querySelector('.people-menu-trigger')
      if (trigger === exceptTrigger) return
      panel.remove()
      if (trigger) trigger.setAttribute('aria-expanded', 'false')
    })
  }

  function placePanel(panel, trigger) {
    var bar = document.querySelector('.topbar')
    var top0 = (bar ? bar.getBoundingClientRect().bottom : 0) + MARGIN
    panel.style.position = 'fixed'
    panel.style.right = 'auto'
    panel.style.margin = '0'
    panel.style.maxHeight = ''
    panel.style.overflowY = ''
    panel.style.left = '0px'
    panel.style.top = '0px'

    var t = trigger.getBoundingClientRect()
    var vw = document.documentElement.clientWidth
    var vh = window.innerHeight
    var p = panel.getBoundingClientRect()

    var left = Math.min(Math.max(MARGIN, t.right - p.width), vw - p.width - MARGIN)
    left = Math.max(MARGIN, left)

    var below = t.bottom + 4
    var top
    if (below + p.height <= vh - MARGIN) {
      top = below
    } else if (t.top - 4 - p.height >= top0) {
      top = t.top - 4 - p.height
    } else {
      // Too tall for either side: fill the room there is and scroll inside.
      top = top0
      panel.style.maxHeight = vh - top0 - MARGIN + 'px'
      panel.style.overflowY = 'auto'
    }
    panel.style.left = left + 'px'
    panel.style.top = top + 'px'
  }

  function replaceFragment(html) {
    var target = document.querySelector('[data-refresh-fragment-selector]')
    if (!target) return false
    var selector = target.dataset.refreshFragmentSelector
    var fresh = new DOMParser().parseFromString(html, 'text/html').querySelector(selector)
    var old = document.querySelector(selector)
    if (!fresh || !old) return false
    old.replaceWith(fresh)
    // Each screen's drag setup runs again on the new nodes.
    document.dispatchEvent(new CustomEvent('refresh:applied'))
    return true
  }

  async function openPeopleMenu(trigger) {
    closePeopleMenus(trigger)
    var holder = trigger.closest('.people-menu')
    if (!holder) return
    var url = new URL(trigger.getAttribute('href'), location.href)
    url.searchParams.set('fragment', '1')
    var res = await fetch(url.pathname + url.search)
    if (!res.ok) throw new Error('could not load the people menu')
    holder.querySelectorAll('.people-menu-panel').forEach(function (p) {
      p.remove()
    })
    holder.insertAdjacentHTML('beforeend', await res.text())
    var panel = holder.querySelector('.people-menu-panel')
    placePanel(panel, trigger)
    trigger.setAttribute('aria-expanded', 'true')
    var first = holder.querySelector('.people-menu-item')
    if (first) first.focus({ preventScroll: true })
  }

  function openPanelPlace() {
    var panel = document.querySelector('.people-menu > .people-menu-panel')
    if (!panel) return
    var trigger = panel.closest('.people-menu').querySelector('.people-menu-trigger')
    if (trigger) placePanel(panel, trigger)
  }

  document.addEventListener('click', async function (event) {
    var target = event.target instanceof Element ? event.target : null
    if (!target) return
    var trigger = target.closest('.people-menu-trigger')
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

  document.addEventListener('keydown', function (event) {
    if (event.key !== 'Escape') return
    var open = document.querySelector('.people-menu > .people-menu-panel')
    if (!open) return
    var trigger = open.closest('.people-menu').querySelector('.people-menu-trigger')
    closePeopleMenus(null)
    if (trigger) trigger.focus()
  })

  // The menu follows its circles when the page or the lanes scroll, and when
  // the window changes size.
  document.addEventListener('scroll', openPanelPlace, true)
  window.addEventListener('resize', openPanelPlace)

  // Picking a name: post, re-fetch the screen's fragment, and keep the menu
  // open on the same card.
  document.addEventListener('submit', async function (event) {
    var form = event.target
    if (!(form instanceof HTMLFormElement) || !form.classList.contains('people-menu-panel')) return
    event.preventDefault()
    var card = form.closest('[data-task-id]')
    var taskId = card ? card.dataset.taskId : null
    try {
      await fetch(form.action, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams(new FormData(form, event.submitter)).toString(),
        redirect: 'manual',
      })
      var res = await fetch(location.pathname + location.search)
      if (!res.ok || !replaceFragment(await res.text())) throw new Error('could not refresh')
      if (taskId) {
        var trigger = document.querySelector('[data-task-id="' + taskId + '"] .people-menu-trigger')
        if (trigger) await openPeopleMenu(trigger)
      }
    } catch (err) {
      form.submit() // the plain form does the same job
    }
  })
})()
