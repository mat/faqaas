```sql
CREATE TABLE locales (
  id SERIAL PRIMARY KEY,
  code TEXT UNIQUE NOT NULL,
  name TEXT
);


// Upsert
INSERT INTO locales (code,name) VALUES ('de', 'German') ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name;

// Do nothing
INSERT INTO locales (code,name) VALUES ('en', 'English') ON CONFLICT (code) DO NOTHING;
INSERT INTO locales (code,name) VALUES ('fr', 'French') ON CONFLICT (code) DO NOTHING;

SELECT * FROM locales;
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
