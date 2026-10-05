// One place for how every list of jobs is dragged (Board, My jobs, Team;
// SPEC B13.2, gate 7.13). Each page's script calls comphqSortable(list,
// options) instead of Sortable.create, so a fix here reaches all of them.
//
// Auto-scroll: with the browser's own drag-and-drop (SortableJS's default),
// SortableJS scrolls only while the pointer is over one of its lists, and
// leaves the rest to the browser, which scrolls only at the very edge of the
// window. Since v1.3 the top 76px of the window is the sticky top bar, so
// the very edge is the bar, not the page: a card could never be carried up
// to a column's top. forceFallback makes SortableJS follow the pointer
// itself (pointer events, a clone under the pointer), so its scrolling works
// wherever the pointer is, with a zone that reaches past the bar (top bar
// height + 40px) at both edges (D-101).
;(function () {
  var SCROLL_ZONE = 120

  window.comphqSortable = function (list, options) {
    return Sortable.create(
      list,
      Object.assign(
        {
          animation: 150,
          forceFallback: true,
          scroll: true,
          forceAutoScrollFallback: true,
          scrollSensitivity: SCROLL_ZONE,
          scrollSpeed: 18,
          bubbleScroll: true,
        },
        options,
      ),
    )
  }
})()
