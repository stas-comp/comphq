// Shared refresh mechanism (SPEC gate 2.10, PLAN.md P2-07/D-15). A page
// opts in by marking one element with data-refresh-version-url and
// data-refresh-fragment-selector; everything else — polling, the
// "Updated — refresh" notice, staying quiet while something on the
// page would be disrupted by a swap — is handled here so a section only
// wires up the marker and, if it has its own interactive state (a drag,
// an open editor), toggles document.body's data-refresh-busy attribute.
document.addEventListener('DOMContentLoaded', initRefresh)

const POLL_INTERVAL_MS = 60000

function initRefresh() {
  const target = document.querySelector('[data-refresh-version-url]')
  if (!target) return

  const versionURL = target.dataset.refreshVersionUrl
  const fragmentSelector = target.dataset.refreshFragmentSelector
  const notice = document.getElementById('refresh-notice')

  let knownVersion = null

  async function fetchVersion() {
    try {
      const res = await fetch(versionURL)
      if (!res.ok) return null
      const data = await res.json()
      return data.version
    } catch {
      return null
    }
  }

  function isBusy() {
    return document.body.dataset.refreshBusy === '1'
  }

  // Re-fetches this same page and swaps in just the fragment, rather
  // than a full reload — the server is still the source of truth for
  // what changed, this just avoids losing scroll position and anything
  // else outside the fragment.
  async function applyUpdate() {
    const res = await fetch(location.pathname + location.search)
    if (!res.ok) return
    const html = await res.text()
    const newFragment = new DOMParser().parseFromString(html, 'text/html').querySelector(fragmentSelector)
    const oldFragment = document.querySelector(fragmentSelector)
    if (newFragment && oldFragment) {
      oldFragment.replaceWith(newFragment)
      document.dispatchEvent(new CustomEvent('refresh:applied'))
    }
    if (notice) notice.hidden = true
  }

  async function check() {
    const version = await fetchVersion()
    if (version === null) return
    if (knownVersion === null) {
      knownVersion = version
      return
    }
    if (version === knownVersion) return
    knownVersion = version
    if (isBusy()) {
      if (notice) notice.hidden = false
    } else {
      await applyUpdate()
    }
  }

  if (notice) {
    notice.addEventListener('click', applyUpdate)
  }

  check()
  setInterval(() => {
    if (document.visibilityState === 'visible') check()
  }, POLL_INTERVAL_MS)
  window.addEventListener('focus', check)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') check()
  })
}
