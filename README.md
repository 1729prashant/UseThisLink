# UseThisLink

Name: UseThisLink <br>
Goal: A scalable, production-ready URL shortening service with click tracking and basic analytics, built in Go with SQLite.



### Current State

Run app locally

```
PORT=3000 go run cmd/main.go
```


**Adding New URLs - UNIX**

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
  "short_url": "abc123"
}
```


**Adding New URLs - Windows**

asuming curl is installed...

```
curl -X POST http://localhost:3000/shorten ^
  -H "Content-Type: application/json" ^
  -d "{\"url\": \"https://example.com\"}"
```

Response should contain:

```
{
  "short_url": "abc123"
}
```

**Check redirection**

```
curl -i http://localhost:3000/abc123
```

Expected Response:
You should see a 302 Found HTTP status code and a Location header pointing to the original URL (e.g., https://example.com), like this:

```
HTTP/1.1 302 Found
Location: https://example.com
...
```

Paste url in browser to check as well..