# UseThisLink

Name: UseThisLink <br>
Goal: A scalable, production-ready URL shortening service with click tracking and basic analytics, built in Go with SQLite.

---

## Docker Compose Setup

To run all services locally with Docker Compose:

```sh
docker-compose up --build
```

### Service URLs and Ports

| Service      | URL/Port           | Environment Variables (default)                |
|--------------|--------------------|-----------------------------------------------|
| Gateway      | http://localhost:8080 | PORT=8080, LINK_SERVICE_URL, ANALYTICS_SERVICE_URL, USER_SERVICE_URL |
| Link         | http://localhost:8081 | PORT=8081, LINK_DB_PATH, BASE_URL             |
| Analytics    | http://localhost:8082 | PORT=8082, ANALYTICS_DB_PATH, BASE_URL        |
| User/Auth    | http://localhost:8083 | PORT=8083, USER_DB_PATH, BASE_URL, SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS |

- All services use `BASE_URL=http://localhost:8080` for generating links.
- Database files are stored in named Docker volumes for persistence.

### SMTP Configuration (User Service)
- `SMTP_HOST`: SMTP server hostname (e.g., smtp.example.com)
- `SMTP_PORT`: SMTP server port (e.g., 587)
- `SMTP_USER`: SMTP username
- `SMTP_PASS`: SMTP password

---

### Example .env (for local dev, not used in Docker Compose)
```
PORT=8080
BASE_URL=http://localhost:8080
DB_PATH=./internal/db/usethislink.db
```

---

### API Examples

```
curl -X POST -d '{"original_url": "https://example.com"}' http://localhost:8080/shorten
curl http://localhost:8080/r/Ab1XyZ # Redirects
curl http://localhost:8080/stats/Ab1XyZ
```

---

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
