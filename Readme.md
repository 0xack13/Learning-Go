# A Simple Wine API in Go

## Available Endpoints
- GET /status
   ```
   curl -X GET -H "Content-Type: application/json" http://localhost:8080/status
   ```
- GET /wine
  - Optional query params start & count for pagination
   ```
   curl -X GET -H "Content-Type: application/json" "http://localhost:8080/wine?start=0&count=10"
  ```
- GET /wine/[id]
  ```
  curl -X GET -H "Content-Type: application/json" http://localhost:8080/wine/100000
  ```
- PUT /wine
  - Body should contain json formatted wine obj minus the id
   ```
  curl -X cation/json" -d '{"country":"US","description":"this is a wine","designation":"delicious","points":"104.9","price":"a lot","province":"here","region_1":"north","region_2":"carolina","taster_name":"Jazz Jackrabbit","taster_twitter_handle":"@jazzjackrabbit","title":"new wine","variety":"the spice of life","winery":"martha's vineyard"}' http://localhost:8080/wine```

## Build and run
Just run docker build in the source directory.
```
docker build -t go-wine .
```
Then run the container with the desired port binding.
```
docker run -p 8080:8080 --rm go-wine
```

## My Thoughts

This was my first experience with Go, so most of the time spent went into getting familiar with the language and its paradigms. I feel like I managed to take advantage of goroutines and channels to avoid race conditions (multiple http handlers/threads incrementing counters, modifying the records) in an idiomatic way. But by sharing a channel to a single state manager among the http handlers (it will block access to _all_ synchronized state on every msg), I made things far slower than necessary. I'd fix that with with mutliple channels or something with more time. But the server should avoid race conditions and I avoided falling back on mutexes.

I added receivers to the handler funcs to make them methods on a struct containing the state I needed to get into them. Being new to Go, this felt like such an odd way to get state into a function. But at least it wasn't global. The way Go's methods & function receivers work is really unique. I didn't go looking for reasons to use interfaces or type assertions.

If I had more time, I definitely would have modularized the project. I also might have found a real persistence solution for the data. Using Docker compose and throwing a postgres container into the mix would work, but ideally I'd target something on Azure/AWS and include a Terraform module.