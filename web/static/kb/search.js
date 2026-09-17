// The search box's live panel (SPEC gates 1.26, 1.27, 1.32), on every
// page via layout.html. Pressing Enter without picking a result falls
// through to the form's own normal GET submission to /kb/search — the
// full results page needs no JS at all.
document.addEventListener('DOMContentLoaded', () => {
  const input = document.getElementById('search-box')
  if (!input) return
  const form = input.closest('form')

  let panel = null
  let debounceTimer = null
  let activeIndex = -1
  let requestSeq = 0

  function closePanel() {
    if (panel) {
      panel.remove()
      panel = null
    }
    activeIndex = -1
  }

  function escapeHTML(text) {
    const span = document.createElement('span')
    span.textContent = text
    return span.innerHTML
  }

  function renderResults(query, results) {
    closePanel()
    panel = document.createElement('div')
    panel.className = 'search-panel'
    panel.setAttribute('role', 'listbox')
    panel.setAttribute('aria-label', 'Search results')

    if (results.length === 0) {
      const empty = document.createElement('p')
      empty.className = 'search-panel-empty'
      empty.innerHTML = window.ComphqMessages.noArticlesMatch(query)
      panel.appendChild(empty)
    } else {
      for (const result of results) {
        const a = document.createElement('a')
        a.href = result.url
        a.className = 'search-result'
        a.setAttribute('role', 'option')
        a.innerHTML =
          '<span class="search-result-title">' + escapeHTML(result.title) + '</span> ' +
          '<span class="search-result-category">' + escapeHTML(result.category) + '</span>' +
          '<div class="search-snippet">' + result.snippet + '</div>'
        panel.appendChild(a)
      }
    }
    form.appendChild(panel)
  }

  async function runSearch() {
    const query = input.value
    if (query.trim().length < 2) {
      closePanel()
      return
    }
    const seq = ++requestSeq
    let data
    try {
      const res = await fetch('/kb/search.json?q=' + encodeURIComponent(query))
      if (!res.ok) return
      data = await res.json()
    } catch (err) {
      return
    }
    if (seq !== requestSeq) return // a newer keystroke has already moved on
    renderResults(query, data.results)
  }

  input.addEventListener('input', () => {
    clearTimeout(debounceTimer)
    debounceTimer = setTimeout(runSearch, 250)
  })

  function setActive(items, index) {
    activeIndex = index
    items.forEach((el, i) => el.classList.toggle('active', i === activeIndex))
    if (activeIndex >= 0) items[activeIndex].scrollIntoView({ block: 'nearest' })
  }

  input.addEventListener('keydown', (event) => {
    if (!panel) return
    const items = Array.from(panel.querySelectorAll('.search-result'))
    if (items.length === 0) return

    if (event.key === 'ArrowDown') {
      event.preventDefault()
      setActive(items, Math.min(activeIndex + 1, items.length - 1))
    } else if (event.key === 'ArrowUp') {
      event.preventDefault()
      setActive(items, Math.max(activeIndex - 1, 0))
    } else if (event.key === 'Escape') {
      closePanel()
    } else if (event.key === 'Enter' && activeIndex >= 0) {
      event.preventDefault()
      window.location.href = items[activeIndex].href
    }
  })

  document.addEventListener('click', (event) => {
    if (panel && !form.contains(event.target)) closePanel()
  })
})
