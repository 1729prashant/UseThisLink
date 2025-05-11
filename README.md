# UseThisLink

Name: UseThisLink <br>
Goal: A scalable, production-ready URL shortening service with click tracking and basic analytics, built in Go with SQLite.



### Current State

UNIX ONLY FOR NOW, the env file is not included in this repo currently

1. Clone the repo

2. create an .env file and add the following

	```
	PORT=3000
	BASE_URL=http://localhost:3000
	DB_PATH=./internal/db/usethislink.db
	```

3. Run app locally
```
go run cmd/main.go
```

4. Open your browser and go to: http://localhost:3000/

**Testing via terminal**

```
curl -X POST http://localhost:3000/shorten \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://example.com"}'
```


Response should contain something like:

```
{"short_url": "http://localhost:3000/6cXdnlhM"}
```

**Work in Progress**

1. Handle multiple sessions/track url creation per sessions
2. Add analytics
3. Sign Up/Login features
4. UI Changes
5. Simpler Packaging/Installation
6. Monetisation related features
7. CI/CD pipelines
