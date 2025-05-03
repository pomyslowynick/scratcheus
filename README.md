## Scratcheus

Or Prometheus from scratch, it's my attempt at teaching myself about the internals of Prometheus by recreating its core components.

The plan is to have:
- Minimal parser and lexer for metrics and promql
- Basic service discovery module
- Managers, scrape pools and scrape loops
- Reloader
- Chunking
- Compression
- Xor & Varint encoding
- Rule evaluation 
- Promql query engine
    - Particularly, I want to implement the most popular functions like `rate` and `sum`
- Minimal TSDB with WAL, maybe WBL

It's a lot of stuff!
Each of the components will be created by diving into Prometheus code, copying relevant parts of it and dumbing it down.

I'll write a blog post about each part and try to make it into a series tutorials to be followed, at the of which we will "deploy" our Prometheus into a `kind` cluster and have it scrape some targets.

Run with `go run main.go` :)

### tests/ directory

It's not real unit tests, but free recall exercise writing functions which I explore from memory, to deepen my understanding.
