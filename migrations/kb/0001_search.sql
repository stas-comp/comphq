CREATE VIRTUAL TABLE kb_search USING fts5(
    title,
    body,
    article_id UNINDEXED,
    block_id UNINDEXED,
    tokenize = 'porter unicode61'
);
