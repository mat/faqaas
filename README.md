# FAQs as a Service (faqaas)


[![Build Status](https://travis-ci.com/mat/faqaas.svg?branch=master)](https://travis-ci.com/mat/faqaas)

## DB Setup

Load [schema.sql](https://github.com/mat/faqaas/blob/master/schema.sql) 

```
psql -U postgres -d faqaas_test -f schema.sql
```


## API

### GET /locales
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


## Configuration

See [start_server.example](https://github.com/mat/faqaas/blob/master/start_server.example) for a list of environment variables.

## Heroku / Setup


```bash
heroku labs:enable runtime-dyno-metadata
```
