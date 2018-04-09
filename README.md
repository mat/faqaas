
```
psql -U postgres
\c faqaas
```

```sql
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


## Heroku / Setup


```bash
heroku labs:enable runtime-dyno-metadata
```
