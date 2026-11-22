# Waiting for the owner — Phase 8

One item. Nothing here stopped the release.

## 1. "Search questions instead" is on the article results page, not in the live panel

**What the spec says (gate 8.78):** "Article and job results gain **Search questions instead**."

**What I built:** job results (the live panel and the full page) and article results (the full results page) all have it. The little panel that drops down as you type in the Knowledge Base box does not, because the foot of that panel is covered by an older design test (gate 4.42) that wasn't on the list of tests I was allowed to change. Pressing **Enter** shows the full article results page, which does have the link.

**My recommendation:** leave it as it is. The link is one Enter away, and it keeps the article panel exactly as you approved it. If you would rather have it in the panel too, say so and a later update can add it (it means changing that one design test, and you'd approve that).

**What happens if you don't answer:** nothing. 1.6.0 is released as it is.
