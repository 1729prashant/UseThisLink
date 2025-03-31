# UseThisLink: Functionality-Wise Breakdown

**Name**: UseThisLink  
**Goal**: A scalable, production-ready URL shortening service with click tracking and basic analytics, built in Go with SQLite.  
**Target**: Impress recruiters/tech leads with modern engineering practices and offer a monetizable foundation (e.g., premium features later).

---

## Functionality-Wise Breakdown

### 1. Project Setup & Core Infrastructure
**What**: Establish the foundation—code structure, dependencies, database, and basic server.  
**Why**: Sets the stage for a clean, maintainable app.  
**Tasks**:  
- **Init Project**: 
  - `mkdir usethislink && cd usethislink && go mod init usethislink`
  - Install: `go get github.com/gorilla/mux github.com/sirupsen/logrus github.com/joho/godotenv github.com/mattn/go-sqlite3`
- **Structure**:
```
usethislink/
  ├── cmd/          # Entry point
  │   └── main.go
  ├── internal/     # Core logic
  │   ├── models/   # Data structs
  │   ├── db/       # SQLite interactions
  │   └── shortener/# Shortening logic
  ├── api/          # HTTP handlers
  ├── logs/         # Log files
  └── tests/        # Unit tests
```

- **SQLite Setup** (`internal/db/db.go`): 
- Table: `urls (id INTEGER PRIMARY KEY AUTOINCREMENT, original_url TEXT NOT NULL, short_code TEXT UNIQUE NOT NULL, created_at TIMESTAMP, click_count INTEGER)`
- Function: `InitDB(dbPath string) (*sql.DB, error)` to connect and create table
- **Basic Server** (`cmd/main.go`): 
- `/health` endpoint: `{"status": "ok"}`
- Load `.env` (e.g., `PORT=8080`, `DB_PATH=usethislink.db`, `BASE_URL=http://localhost:8080`)

---

### 2. Data Models & Shortening Logic
**What**: Define the URL data structure and logic to generate short codes.  
**Why**: Core of the app—how links are stored and created.  
**Tasks**:  
- **Model** (`internal/models/url.go`): 
- Struct: `URL {ID int64, Original string, ShortCode string, CreatedAt time.Time, ClickCount int}`
- **Shortening Logic** (`internal/shortener/shortener.go`): 
- Function: `GenerateShortCode() string`—6-char base62 (`a-zA-Z0-9`) using `crypto/rand`
- Collision check: Query DB, regenerate if exists
- **DB Interaction** (`internal/db/db.go`): 
- `SaveURL(url models.URL) (models.URL, error)`: Insert into `urls`, return populated struct

---

### 3. API Endpoints
**What**: Build RESTful endpoints for shortening, redirecting, and stats.  
**Why**: User-facing functionality that showcases API design skills.  
**Tasks**:  
- **POST /shorten** (`api/handlers.go`): 
- Input: `{"url": "https://example.com/long-url"}`
- Output: `{"short_url": "http://localhost:8080/Ab1XyZ"}`
- Validate URL with `net/url`
- **GET /{shortcode}**: 
- Fetch `original_url` from DB, increment `click_count`, redirect (HTTP 302)
- **GET /stats/{shortcode}**: 
- Return: `{"original_url": "...", "click_count": 42, "created_at": "..."}`
- **Routing**: Use `gorilla/mux` for clean path handling

---

### 4. Error Handling & Logging
**What**: Add robustness with proper errors and logs.  
**Why**: Production-grade apps need reliability and traceability.  
**Tasks**:  
- **Error Handling**: 
- Validate inputs (e.g., 400 for invalid URLs)
- 404 for unknown shortcodes
- JSON errors: `{"error": "message"}`
- **Logging**: 
- Use `logrus` to log to `logs/usethislink.log`
- Log requests (method, path, duration) and key actions (e.g., “Shortened URL: Ab1XyZ”)
- **Middleware**: Wrap handlers to log every request

---

### 5. Testing
**What**: Write unit tests for critical components.  
**Why**: Shows you care about quality—huge for tech leads.  
**Tasks**:  
- **Tests** (`tests/shortener_test.go`): 
- `GenerateShortCode()`: Check length, uniqueness
- Redirect: Mock DB, verify 302
- Stats: Mock data, check JSON
- **Tools**: Use `testing` and `github.com/stretchr/testify/assert`
- **Goal**: ~70% coverage on `internal/shortener`

---

### 6. Deployment Prep
**What**: Package and deploy the app.  
**Why**: Proves it’s not just a toy—ready for the real world.  
**Tasks**:  
- **Config**: 
- `.env` for `PORT`, `DB_PATH`, `BASE_URL`
- **Dockerfile**:
```
  FROM golang:1.21-alpine
  WORKDIR /app
  COPY . .
  RUN go build -o usethislink ./cmd/main.go
  CMD ["./usethislink"]
```

- **Deploy**: 
- Test: `docker build -t usethislink . && docker run -p 8080:8080 usethislink`
- Push to Fly.io/Render (free tier, persist SQLite)

---

### 7. Polish & Presentation
**What**: Document and showcase the project.  
**Why**: Makes it recruiter-ready and shareable.  
**Tasks**:  
- **README.md**: 
- Purpose: “Production-grade URL shortener with analytics”
- Setup, API examples:
  ```
  curl -X POST -d '{"url": "https://example.com"}' http://localhost:8080/shorten
  curl http://localhost:8080/Ab1XyZ # Redirects
  curl http://localhost:8080/stats/Ab1XyZ
  ```
