// The search box's live panel (SPEC gates 1.26, 1.27, 1.32), on every
// page via layout.html. On Tasks pages the same box searches jobs (gates
// 7.20-7.24): the form says which, and where, in data attributes, and a job
// result is drawn a little differently; articles behave exactly as before. Pressing Enter without picking a result falls
// through to the form's own normal GET submission to /kb/search — the
// full results page needs no JS at all.
document.addEventListener('DOMContentLoaded', () => {
  const input = document.getElementById('search-box')
  if (!input) return
  const form = input.closest('form')
  const jsonURL = form.dataset.searchJson || '/kb/search.json'
  const jobs = form.dataset.searchKind === 'jobs'
  const questions = form.dataset.searchKind === 'questions'
  // The other searches, offered under the results (gates 7.24, 8.78).
  let insteads = []
  try {
    insteads = form.dataset.insteads ? JSON.parse(form.dataset.insteads) : []
  } catch (err) {
    insteads = []
  }

  // The "/" hint in the box is a real shortcut (SPEC gate 4.14): pressing
  // it anywhere outside a text field puts the keyboard in the search box.
  document.addEventListener('keydown', (event) => {
    if (event.key !== '/' || event.ctrlKey || event.metaKey || event.altKey) return
    const target = event.target
    if (target instanceof HTMLElement && (target.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName))) return
    event.preventDefault()
    input.focus()
  })

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

  // The other kinds of search, offered under the results (gates 7.24, 8.78).
  function insteadLinks(query) {
    return insteads.map((link) => {
      const a = document.createElement('a')
      a.className = 'search-instead-link'
      a.href = link.href + '?q=' + encodeURIComponent(query)
      a.textContent = link.label
      return a
    })
  }

  // Puts the links at the end of a foot, each after a dot.
  function appendInsteads(foot, query, needsLeadingDot) {
    insteadLinks(query).forEach((a, i) => {
      if (i > 0 || needsLeadingDot) foot.appendChild(document.createTextNode(' · '))
      foot.appendChild(a)
    })
  }

  // A job result (gate 7.21): the title with the matching words marked, its
  // column, people and due date, and a passage of the notes when that is
  // where the words are. title and passage arrive escaped, with <mark>.
  function renderJobs(query, results) {
    closePanel()
    // The panel holds the list of results and, under it, a link that is not
    // one of them; only the list is a listbox.
    panel = document.createElement('div')
    panel.className = 'search-panel'
    if (results.length === 0) {
      const empty = document.createElement('p')
      empty.className = 'search-panel-empty'
      empty.innerHTML = window.ComphqMessages.noJobsMatch(query)
      panel.appendChild(empty)
    }
    const list = document.createElement('div')
    list.setAttribute('role', 'listbox')
    list.setAttribute('aria-label', 'Search results')
    for (const r of results) {
      const a = document.createElement('a')
      a.href = r.url
      a.className = 'search-result'
      a.setAttribute('role', 'option')
      const meta = [r.column].concat(r.people || [])
      if (r.due) meta.push(r.due)
      a.innerHTML =
        '<span class="search-result-title">' + r.title +
        (r.finished ? ' <span class="stamp stamp-date">Finished</span>' : '') + '</span>' +
        '<span class="search-job-meta">' + escapeHTML(meta.join(' · ')) + '</span>' +
        (r.passage ? '<div class="search-snippet">' + r.passage + '</div>' : '')
      list.appendChild(a)
    }
    if (results.length > 0) panel.appendChild(list)
    const foot = document.createElement('div')
    foot.className = 'search-panel-foot'
    if (results.length > 0) {
      const text = document.createElement('span')
      text.textContent = window.ComphqMessages.jobsFoot(results.length)
      foot.appendChild(text)
      foot.appendChild(document.createTextNode(' · '))
    }
    appendInsteads(foot, query, false)
    panel.appendChild(foot)
    form.appendChild(panel)
    // A job opens in the task window over this page (window.js); the panel
    // has done its job.
    panel.addEventListener('click', (event) => {
      if (event.target instanceof Element && event.target.closest('a.search-result')) setTimeout(closePanel, 0)
    })
  }

  // A question result (gate 8.78): the question with the matching words marked,
  // who asked and when, how many answers, and a passage when the words are in the
  // details or an answer. title and passage arrive escaped, with <mark>.
  function renderQuestions(query, results) {
    closePanel()
    panel = document.createElement('div')
    panel.className = 'search-panel'
    if (results.length === 0) {
      const empty = document.createElement('p')
      empty.className = 'search-panel-empty'
      empty.innerHTML = window.ComphqMessages.noQuestionsMatch(query)
      panel.appendChild(empty)
    }
    const list = document.createElement('div')
    list.setAttribute('role', 'listbox')
    list.setAttribute('aria-label', 'Search results')
    for (const r of results) {
      const a = document.createElement('a')
      a.href = r.url
      a.className = 'search-result'
      a.setAttribute('role', 'option')
      const meta = ['Asked by ' + r.asked_by + ', ' + r.asked, r.answers + (r.answers === 1 ? ' answer' : ' answers')]
      a.innerHTML =
        '<span class="search-result-title">' + r.title +
        (r.answered ? ' <span class="stamp stamp-date">Answered</span>' : '') + '</span>' +
        '<span class="search-job-meta">' + escapeHTML(meta.join(' · ')) + '</span>' +
        (r.passage ? '<div class="search-snippet">' + r.passage + '</div>' : '')
      list.appendChild(a)
    }
    if (results.length > 0) panel.appendChild(list)
    const foot = document.createElement('div')
    foot.className = 'search-panel-foot'
    if (results.length > 0) {
      const text = document.createElement('span')
      text.textContent = window.ComphqMessages.questionsFoot(results.length)
      foot.appendChild(text)
    }
    appendInsteads(foot, query, results.length > 0)
    panel.appendChild(foot)
    form.appendChild(panel)
  }

  function renderResults(query, results) {
    if (jobs) return renderJobs(query, results)
    if (questions) return renderQuestions(query, results)
    closePanel()
    panel = document.createElement('div')
    panel.className = 'search-panel'

    if (results.length === 0) {
      const empty = document.createElement('p')
      empty.className = 'search-panel-empty'
      empty.innerHTML = window.ComphqMessages.noArticlesMatch(query)
      panel.appendChild(empty)
    } else {
      // Only the list is a listbox: the foot's links are not options (gate 8.78).
      const list = document.createElement('div')
      list.setAttribute('role', 'listbox')
      list.setAttribute('aria-label', 'Search results')
      for (const result of results) {
        const a = document.createElement('a')
        a.href = result.url
        a.className = 'search-result'
        a.setAttribute('role', 'option')
        a.innerHTML =
          '<span class="search-result-title">' + escapeHTML(result.title) + '</span> ' +
          '<span class="search-result-category">' + escapeHTML(result.category) + '</span>' +
          '<div class="search-snippet">' + result.snippet + '</div>'
        list.appendChild(a)
      }
      panel.appendChild(list)
    }
    // The foot: how many, and the other searches (gate 8.78, "Search questions
    // instead" on article results).
    const foot = document.createElement('div')
    foot.className = 'search-panel-foot'
    if (results.length > 0) {
      const text = document.createElement('span')
      text.textContent = window.ComphqMessages.searchFoot(results.length)
      foot.appendChild(text)
    }
    appendInsteads(foot, query, results.length > 0)
    if (foot.childNodes.length > 0) panel.appendChild(foot)
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
      const res = await fetch(jsonURL + '?q=' + encodeURIComponent(query))
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
