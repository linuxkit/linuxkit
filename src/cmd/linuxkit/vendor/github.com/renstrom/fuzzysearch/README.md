# Fuzzy Search

[![Build Status](https://img.shields.io/travis/renstrom/fuzzysearch.svg?style=flat-square)](https://travis-ci.org/renstrom/fuzzysearch)
[![Godoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/renstrom/fuzzysearch/fuzzy)

Inspired by _[bevacqua/fuzzysearch][1]_, a fuzzy matching library written in JavaScript. But contains some extras like ranking using _[Levenshtein distance][2]_ (see [`RankMatch()`](https://godoc.org/github.com/renstrom/fuzzysearch/fuzzy#RankMatch)) and finding matches in a list of words (see [`Find()`](https://godoc.org/github.com/renstrom/fuzzysearch/fuzzy#Find)).

Fuzzy searching allows for flexibly matching a string with partial input, useful for filtering data very quickly based on lightweight user input.

The current implementation uses the algorithm suggested by Mr. Aleph, a russian compiler engineer working at V8.

## Usage

```go
fuzzy.Match("twl", "cartwheel")  // true
fuzzy.Match("cart", "cartwheel") // true
fuzzy.Match("cw", "cartwheel")   // true
fuzzy.Match("ee", "cartwheel")   // true
fuzzy.Match("art", "cartwheel")  // true
fuzzy.Match("eeel", "cartwheel") // false
fuzzy.Match("dog", "cartwheel")  // false

fuzzy.RankMatch("kitten", "sitting") // 3

words := []string{"cartwheel", "foobar", "wheel", "baz"}
fuzzy.Find("whl", words) // [cartwheel wheel]

fuzzy.RankFind("whl", words) // [{whl cartwheel 6} {whl wheel 2}]
```

You can sort the result of a `fuzzy.RankFind()` call using the [`sort`](https://golang.org/pkg/sort/) package in the standard library:

```go
matches := fuzzy.RankFind("whl", words) // [{whl cartwheel 6} {whl wheel 2}]
sort.Sort(matches) // [{whl wheel 2} {whl cartwheel 6}]
```

## License

MIT

[1]: https://github.com/bevacqua/fuzzysearch
[2]: http://en.wikipedia.org/wiki/Levenshtein_distance
