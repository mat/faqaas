
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


INSERT INTO faq_texts (faq_id,locale,question,answer) VALUES (10,'de', 'Wer hat an der Uhr gedreht?', 'Paulchen Panter');

```

## GET /locales
	[
	  {
	    "code": "en",
	    "name": "English",
	    "locale_name": "English"
	  },
	  {
	    "code": "fr",
	    "name": "French",
	    "locale_name": "français"
	  },
	  {
	    "code": "es",
	    "name": "Spanish",
	    "locale_name": "español"
	  },
	  {
	    "code": "pt-BR",
	    "name": "Brazilian Portuguese",
	    "locale_name": "português"
	  },
	  {
	    "code": "zh",
	    "name": "Chinese",
	    "locale_name": "中文"
	  }
	]

## Heroku / Setup


```bash
heroku labs:enable runtime-dyno-metadata
```
