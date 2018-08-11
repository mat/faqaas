CREATE TABLE faqs (
  id SERIAL PRIMARY KEY,
  question TEXT,
  answer TEXT
);

CREATE TABLE faq_texts (
  id SERIAL PRIMARY KEY,
  faq_id INTEGER REFERENCES faqs (id),
  locale TEXT,
  question TEXT,
  answer TEXT,
  CONSTRAINT texts_faq_id_locale unique(faq_id,locale)
);

CREATE MATERIALIZED VIEW search_index AS
SELECT faq_texts.id,
       faq_texts.faq_id,
       faq_texts.locale,
--       faq_texts.question,
--       faq_texts.answer,
--       setweight(to_tsvector(post.language::regconfig, faq_texts.question), 'A') ||
--       setweight(to_tsvector(post.language::regconfig, faq_texts.answer), 'B') ||
       setweight(to_tsvector('simple', faq_texts.question), 'A') ||
       setweight(to_tsvector('simple', faq_texts.answer), 'B') as document
--       setweight(to_tsvector('simple', author.name), 'C') ||
--       setweight(to_tsvector('simple', coalesce(string_agg(tag.name, ' '))), 'A') as document
FROM faq_texts;

CREATE INDEX idx_fts_search ON search_index USING gin(document);

REFRESH MATERIALIZED VIEW search_index;
