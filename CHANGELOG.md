## CHANGELOG
2nd of April 
Working through the `scrapeManager` code and the flow involved in writing a sample to a `memSeries`.

15th of April
I've explored the flow of `scrapeManager` and decided to start from the very bottom with `bstream` and `xor` encoding of the series.
Will code up my own little `tsdb` as the first and most significant step, for now omitting the reading parts and just focusing and writing the data.

Gorilla paper has been immensely helpful to understand the encoding.
