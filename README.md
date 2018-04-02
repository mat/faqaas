
```
psql -U postgres
\c faqaas
```

```sql
CREATE TABLE locales (
  id SERIAL PRIMARY KEY,
  code TEXT UNIQUE NOT NULL,
  name TEXT
);

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
  answer TEXT
);



CREATE TABLE categories (
  id SERIAL PRIMARY KEY,
  code TEXT UNIQUE NOT NULL
);

CREATE TABLE category_translations (
  id SERIAL PRIMARY KEY,
  category_id INTEGER REFERENCES categories (id),
  locale_code TEXT REFERENCES locales (code),
  name TEXT
);


// Upsert
INSERT INTO locales (code,name) VALUES ('de', 'German') ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name;

// Do nothing
INSERT INTO locales (code,name) VALUES ('en', 'English') ON CONFLICT (code) DO NOTHING;
INSERT INTO locales (code,name) VALUES ('fr', 'French') ON CONFLICT (code) DO NOTHING;

SELECT * FROM locales;


INSERT INTO faq_texts (faq_id,locale,question,answer) VALUES (10,'de', 'Wer hat an der Uhr gedreht?', 'Paulchen Panter');

```

## GET /locales
	{
		"locales": [{
			"code": "de",
			"name": "German"
		}, {
			"code": "en",
			"name": "English"
		}]
	}

## POST /locales
	{
		"code": "de-de",
		"name": "German"
	}

## PUT /locales/{code}
	{
		"name": "German"
	}
