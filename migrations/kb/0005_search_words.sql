CREATE VIRTUAL TABLE kb_search_words USING fts5(
    title,
    body,
    article_id UNINDEXED,
    block_id UNINDEXED,
    tokenize = 'unicode61'
);

CREATE VIRTUAL TABLE kb_search_words_vocab USING fts5vocab(kb_search_words, row);
