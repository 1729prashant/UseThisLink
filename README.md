# UseThisLink

Name: UseThisLink <br>
Goal: A scalable, production-ready URL shortening service with click tracking and basic analytics, built in Go with SQLite.



### Current State

Run app locally UNIX ONLY FOR NOW

```
PORT=3000 go run cmd/main.go
```


**Adding New URLs**

```
curl -X POST http://localhost:3000/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

OR

```
curl -X POST http://localhost:3000/shorten -d '{"url": "https://example.com"}' 
```

Response should contain:

```
{
  "short_url": "<hostname>abc123"
}
```


