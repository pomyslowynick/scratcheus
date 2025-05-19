## CHANGELOG

### 2nd of April 
Working through the `scrapeManager` code and the flow involved in writing a sample to a `memSeries`.

### 15th of April
I've explored the flow of `scrapeManager` and decided to start from the very bottom with `bstream` and `xor` encoding of the series.
Will code up my own little `tsdb` as the first and most significant step, for now omitting the reading parts and just focusing and writing the data.

Gorilla paper has been immensely helpful to understand the encoding.

### 29th of April
I got the lexing and parsing working by generously copy pasting parts of Prometheus codebase, I implemented mostly from memory `bstream` and `xor` encoding, filling in blanks with the Gorilla paper. Next I tried to implement from memory the append chain, with `head`, `memSeries` and `xorAppender` structs, I think that's the bare minimum for the logical break down.

I've added a hashing function to get back the series based on it's labels, used stdlib FNV for the first time. I saw that Golang uses a more performant `xxhash` as external dependency, but I wanted to keep things simple in terms of external packages.

Finally I've added some unit tests, I should have been writing those from the start, but it always feels like the least fun part. I should be writing them as I add new functionalities, instead I keep using `main.go` for that which could be instead spent on the tests. I caught few issues when writing those, order of operations in `xor.Append` wasn't correct in one of the cases among others, so it's good to write those. Going forward I'll write a test file for each source file I am adding, or at least try to do so.

- [x] Parser and lexer are there, but they could be simplified, given I don't use a lot of the code in there.
- [x] Xor encoding is there, there's no `varint` though, will have to take a second pass at that

Next I will try to have the scraping pools and loops, would be good to get the goroutines cracking too.

### 19th of May

Started working on unit tests, they are a great way of validating the TSDB encoding, spotted few mistakes thanks to them.
There's a lot that could be refactored in my code, it's pretty ugly at the moment, but it's functional. I'll start writing the first part of scratcheus article on the encoding, should also improve my website, it's almost as ugly as this code :D

- [x] Unit tests for xor encoding
- [x] Reader of xor encoded samples
