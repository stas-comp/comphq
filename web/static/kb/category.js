// Putting a category's articles in order by dragging (SPEC B13.8, gate 7.55).
// The move icons beside each article are plain forms that do the same with
// no script; dragging is the extra way. A drop posts the article it was
// dropped beside, and the list is then re-fetched, so the page shows what the
// server really holds.
document.addEventListener('DOMContentLoaded', initArticleOrder)

function initArticleOrder() {
  const list = document.querySelector('.kb-article-list')
  if (!list || typeof comphqSortable !== 'function') return
  comphqSortable(list, {
    // A row is picked up by its handle, so a click on the title is only a
    // click, and the icons are never taken for the start of a drag.
    handle: '.kb-article-handle',
    onStart: () => { document.body.dataset.refreshBusy = '1' },
    onEnd: (evt) => {
      delete document.body.dataset.refreshBusy
      dropArticle(evt.item)
    },
  })
}

async function dropArticle(item) {
  const next = item.nextElementSibling
  const prev = item.previousElementSibling
  const params = new URLSearchParams()
  if (next) params.set('before_id', next.dataset.articleId)
  else if (prev) params.set('after_id', prev.dataset.articleId)
  else return
  try {
    await fetch('/kb/articles/' + item.dataset.articleId + '/move', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: params.toString(),
      redirect: 'manual',
    })
    const res = await fetch(location.pathname)
    const fresh = new DOMParser().parseFromString(await res.text(), 'text/html').querySelector('.kb-article-list')
    const old = document.querySelector('.kb-article-list')
    if (fresh && old) {
      old.replaceWith(fresh)
      initArticleOrder()
    }
  } catch (err) {
    window.location.reload() // the page shows what the server holds
  }
}
