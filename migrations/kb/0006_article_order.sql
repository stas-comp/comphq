-- Articles in their own order within a category (SPEC B13.8, D-98, gates
-- 7.55-7.59). Add-only, so a rollback to 1.3.0 is safe: 1.3.0 never reads the
-- column and its inserts leave it NULL. The application renumbers every
-- category 1..n at each start-up, which puts anything written while rolled
-- back at the bottom of its category.
ALTER TABLE kb_articles ADD COLUMN position INTEGER;

-- Day one looks exactly as 1.3.0 did: the most recently edited article first.
UPDATE kb_articles
SET position = (
    SELECT ranked.rn
    FROM (
        SELECT id AS article_id,
               ROW_NUMBER() OVER (PARTITION BY category_id ORDER BY updated_at DESC, id) AS rn
        FROM kb_articles
        WHERE status = 'published'
    ) AS ranked
    WHERE ranked.article_id = kb_articles.id
)
WHERE status = 'published';
