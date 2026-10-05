# Waiting for the owner — Phase 7

## 1. Your Word document wasn't in the samples folder (2026-10-05, P7-00 / P7-10)

**What happened:** the `samples\word` folder only holds its own README, with no Word documents in it. So the Word-picture fixes (gates 7.60–7.68) are based on reading the code and on test documents I build myself, not on the real document that lost pictures.

**What I did meanwhile:** built the fixes for every cause the spec found, with test documents for each shape of picture (25 pictures in one document, grouped pictures, odd file names, a picture beside the title, and so on). Once a document is in the folder, the test suite checks it automatically (every picture must be shown or clearly marked).

**What I recommend:** when you're back, put the Word document that lost pictures into `samples\word`, and tell me, or start a new session. If it still loses a picture, that becomes a v1.4.x fix.

**What you need to do:** copy the document into the `samples\word` folder inside the Staff HQ folder (nothing private). Nothing else.
