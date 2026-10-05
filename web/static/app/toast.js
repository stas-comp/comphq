// The message at the bottom of the screen (SPEC B13.3, gates 7.15-7.17): it
// goes after 10 seconds, the same as a removed step's Undo (gate 5.06), or as
// soon as anything else is done on the screen (the Board swaps its region's
// contents on every action). It lives in #toast-region, outside anything
// the screen refreshes, so a refresh can't wipe it.
;(function () {
  var TOAST_MS = 10000
  var timer = null

  function arm() {
    clearTimeout(timer)
    var toast = document.querySelector('#toast-region .toast')
    if (!toast) return
    timer = setTimeout(function () {
      toast.remove()
    }, TOAST_MS)
    // Reaching the page by removing a job put ?removed= in the address;
    // a reload shouldn't offer the Undo again.
    var url = new URL(location.href)
    if (url.searchParams.has('removed')) {
      url.searchParams.delete('removed')
      history.replaceState(null, '', url.pathname + url.search + url.hash)
    }
  }

  window.comphqToast = { arm: arm }
  document.addEventListener('DOMContentLoaded', arm)

  // Undo: post, then show the board as it is now. Without script the form
  // does the same on its own.
  document.addEventListener('submit', async function (event) {
    var form = event.target
    if (!(form instanceof HTMLFormElement) || !form.classList.contains('toast-undo')) return
    event.preventDefault()
    try {
      await fetch(form.action, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams(new FormData(form)).toString(),
        redirect: 'manual',
      })
      await replaceBoardFromPage() // also empties the message
    } catch (err) {
      form.submit()
    }
  })
})()
