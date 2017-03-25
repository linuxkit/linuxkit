# Sprig: Template functions for Go templates

The Go language comes with a [built-in template
language](http://golang.org/pkg/text/template/), but not
very many template functions. This library provides a group of commonly
used template functions.

It is inspired by the template functions found in
[Twig](http://twig.sensiolabs.org/documentation).

[![Build Status](https://travis-ci.org/Masterminds/sprig.svg?branch=master)](https://travis-ci.org/Masterminds/sprig)

## Usage

API documentation is available [at GoDoc.org](http://godoc.org/github.com/Masterminds/sprig), but
read on for standard usage.

### Load the Sprig library

To load the Sprig `FuncMap`:

```go

import (
  "github.com/Masterminds/sprig"
  "html/template"
)

// This example illustrates that the FuncMap *must* be set before the
// templates themselves are loaded.
tpl := template.Must(
  template.New("base").Funcs(sprig.FuncMap()).ParseGlob("*.html")
)


```

### Call the functions inside of templates

By convention, all functions are lowercase. This seems to follow the Go
idiom for template functions (as opposed to template methods, which are
TitleCase).


Example:

```
{{ "hello!" | upper | repeat 5 }}
```

Produces:

```
HELLO!HELLO!HELLO!HELLO!HELLO!
```

## Functions

### Date Functions

- date: Format a date, where a date is an integer type or a time.Time type, and
  format is a time.Format formatting string.
- dateModify: Given a date, modify it with a duration: `date_modify "-1.5h" now`. If the duration doesn't
parse, it returns the time unaltered. See `time.ParseDuration` for info on duration strings.
- now: Current time.Time, for feeding into date-related functions.
- htmlDate: Format a date for use in the value field of an HTML "date" form element.
- dateInZone: Like date, but takes three arguments: format, timestamp,
  timezone.
- htmlDateInZone: Like htmlDate, but takes two arguments: timestamp,
  timezone.

### String Functions

- trim: strings.TrimSpace
- trimAll: strings.Trim, but with the argument order reversed `trimAll "$" "$5.00"` or `"$5.00 | trimAll "$"`
- trimSuffix: strings.TrimSuffix, but with the argument order reversed `trimSuffix "-" "5-"`
- trimPrefix: strings.TrimPrefix, but with the argument order reversed `trimPrefix "$" "$5"`
- upper: strings.ToUpper
- lower: strings.ToLower
- title: strings.Title
- repeat: strings.Repeat, but with the arguments switched: `repeat count str`. (This simplifies common pipelines)
- substr: Given string, start, and length, return a substr.
- nospace: Remove all spaces from a string. `h e l l o` becomes
  `hello`.
- abbrev: Truncate a string with ellipses
- trunc: Truncate a string (no suffix). `trunc 5 "Hello World"` yields "hello".
- abbrevboth: Truncate both sides of a string with ellipses
- untitle: Remove title case
- intials: Given multiple words, return the first letter of each
  word
- randAlphaNum: Generate a random alpha-numeric string
- randAlpha: Generate a random alphabetic string
- randAscii: Generate a random ASCII string, including symbols
- randNumeric: Generate a random numeric string
- wrap: Wrap text at the given column count
- wrapWith: Wrap text at the given column count, and with the given
  string for a line terminator: `wrap 50 "\n\t" $string`
- contains: strings.Contains, but with the arguments switched: `contains "cat" "uncatch"`. (This simplifies common pipelines)
- hasPrefix: strings.hasPrefix, but with the arguments switched: `hasPrefix "cat" "catch"`.
- hasSuffix: strings.hasSuffix, but with the arguments switched: `hasSuffix "cat" "ducat"`.
- quote: Wrap strings in double quotes. `quote "a" "b"` returns `"a"
  "b"`
- squote: Wrap strings in single quotes.
- cat: Concatenate strings, separating them by spaces. `cat $a $b $c`.
- indent: Indent a string using space characters. `indent 4 "foo\nbar"` produces "    foo\n    bar"
- replace: Replace an old with a new in a string: `$name | replace " " "-"`
- plural: Choose singular or plural based on length: `len $fish | plural
  "one anchovy" "many anchovies"`
- uuidv4: Generate a UUID v4 string
- sha256sum: Generate a hex encoded sha256 hash of the input
- toString: Convert something to a string

### String Slice Functions:

- join: strings.Join, but as `join SEP SLICE`
- split: strings.Split, but as `split SEP STRING`. The results are returned
  as a map with the indexes set to _N, where N is an integer starting from 0.
  Use it like this: `{{$v := "foo/bar/baz" | split "/"}}{{$v._0}}` (Prints `foo`)
- splitList: strings.Split, but as `split SEP STRING`. The results are returned
  as an array.
- toStrings: convert a list to a list of strings. 'list 1 2 3 | toStrings' produces '["1" "2" "3"]'
- sortAlpha: sort a list lexicographically.

### Integer Slice Functions:

- until: Given an integer, returns a slice of counting integers from 0 to one
	  less than the given integer: `range $i, $e := until 5`
- untilStep: Given start, stop, and step, return an integer slice starting at
	  'start', stopping at `stop`, and incrementing by 'step'. This is the same
	  as Python's long-form of 'range'.

### Conversions:

- atoi: Convert a string to an integer. 0 if the integer could not be parsed.
- int: Convert a string or numeric to an int
- int64: Convert a string or numeric to an int64
- float64: Convert a string or numeric to a float64

### Defaults:

- default: Give a default value. Used like this: {{trim "   "| default "empty"}}.
  Since trim produces an empty string, the default value is returned. For
  things with a length (strings, slices, maps), len(0) will trigger the default.
  For numbers, the value 0 will trigger the default. For booleans, false will
  trigger the default. For structs, the default is never returned (there is
  no clear empty condition). For everything else, nil value triggers a default.
- empty: Returns true if the given value is the zero value for that
  type. Structs are always non-empty.
- coalesce: Given a list of items, return the first non-empty one.
  This follows the same rules as 'empty'. `{{ coalesce .someVal 0 "hello" }}`
  will return `.someVal` if set, or else return "hello". The 0 is skipped
  because it is an empty value.
- compact: Return a copy of a list with all of the empty values removed.
  `list 0 1 2 "" | compact` will return `[1 2]`

### OS:

- env: Read an environment variable.
- expandenv: Expand all environment variables in a string.

### File Paths:
- base: Return the last element of a path. https://golang.org/pkg/path#Base
- dir: Remove the last element of a path. https://golang.org/pkg/path#Dir
- clean: Clean a path to the shortest equivalent name.  (e.g. remove "foo/.."
  from "foo/../bar.html") https://golang.org/pkg/path#Clean
- ext: Get the extension for a file path: https://golang.org/pkg/path#Ext
- isAbs: Returns true if a path is absolute: https://golang.org/pkg/path#IsAbs

### Encoding:

- b32enc: Encode a string into a Base32 string
- b32dec: Decode a string from a Base32 string
- b64enc: Encode a string into a Base64 string
- b64dec: Decode a string from a Base64 string

### Data Structures:

- tuple: Takes an arbitrary list of items and returns a slice of items. Its
  tuple-ish properties are mainly gained through the template idiom, and not
  through an API provided here. WARNING: The implementation of tuple will
  change in the future.
- list: An arbitrary ordered list of items. (This is prefered over tuple.)
- dict: Takes a list of name/values and returns a map[string]interface{}.
  The first parameter is converted to a string and stored as a key, the
  second parameter is treated as the value. And so on, with odds as keys and
  evens as values. If the function call ends with an odd, the last key will
  be assigned the empty string. Non-string keys are converted to strings as
  follows: []byte are converted, fmt.Stringers will have String() called.
  errors will have Error() called. All others will be passed through
  fmt.Sprtinf("%v"). _dicts are unordered_.

List:

```
{{$t := list 1 "a" "foo"}}
{{index $t 2}}{{index $t 0 }}{{index $t 1}}
{{/* Prints foo1a *}}
```

Dict:
```
{{ $t := map "key1" "value1" "key2" "value2" }}
{{ $t.key2 }}
{{ /* Prints value2 *}}
```


### Lists Functions:

These are used to manipulate lists: `{{ list 1 2 3 | reverse | first }}`

- first: Get the first item in a 'list'. 'list 1 2 3 | first' prints '1'
- last: Get the last item in a 'list': 'list 1 2 3 | last ' prints '3'
- rest: Get all but the first item in a list: 'list 1 2 3 | rest' returns '[2 3]'
- initial: Get all but the last item in a list: 'list 1 2 3 | initial' returns '[1 2]'
- append: Add an item to the end of a list: 'append $list 4' adds '4' to the end of '$list'
- prepend: Add an item to the beginning of a list: 'prepend $list 4' puts 4 at the beginning of the list.
- reverse: Reverse the items in a list.
- uniq: Remove duplicates from a list.
- without: Return a list with the given values removed: 'without (list 1 2 3) 1' would return '[2 3]'
- has: Return 'tru' if the item is found in the list: 'has "foo" $list' will return 'true' if the list contains "foo"

### Dict Functions:

These are used to manipulate dicts.

- set: Takes a dict, a key, and a value, and sets that key/value pair in
  the dict. `set $dict $key $value`. For convenience, it returns the dict,
  even though the dict was modified in place.
- unset: Takes a dict and a key, and deletes that key/value pair from the
  dict. `unset $dict $key`. This returns the dict for convenience.
- hasKey: Takes a dict and a key, and returns boolean true if the key is in
  the dict.
- pluck: Given a key and one or more maps, get all of the values for that key.
- keys: Get an array of all of the keys in a dict. Order is not guaranteed.
- pick: Select just the given keys out of the dict, and return a new dict.
- omit: Return a dict without the given keys.

### Reflection:

- typeOf: Takes an interface and returns a string representation of the type.
  For pointers, this will return a type prefixed with an asterisk(`*`). So
  a pointer to type `Foo` will be `*Foo`.
- typeIs: Compares an interface with a string name, and returns true if they match.
  Note that a pointer will not match a reference. For example `*Foo` will not
  match `Foo`.
- typeIsLike: returns true if the interface is of the given type, or
  is a pointer to the given type.
- kindOf: Takes an interface and returns a string representation of its kind.
- kindIs: Returns true if the given string matches the kind of the given interface.

	Note: None of these can test whether or not something implements a given
	interface, since doing so would require compiling the interface in ahead of
	time.


### Math Functions:

Integer functions will convert integers of any width to `int64`. If a
string is passed in, functions will attempt to conver with
`strconv.ParseInt(s, 1064)`. If this fails, the value will be treated as 0.

- add1: Increment an integer by 1
- add: Sum integers. `add 1 2 3` renders `6`
- sub: Subtract the second integer from the first
- div: Divide the first integer by the second
- mod: Module of first integer divided by second
- mul: Multiply integers integers
- max (biggest): Return the biggest of a series of integers. `max 1 2 3`
  returns `3`.
- min: Return the smallest of a series of integers. `min 1 2 3` returns
  `1`.

### Cryptographic Functions:

- derivePassword: Derive a password from the given parameters according to the "Master Password" algorithm (http://masterpasswordapp.com/algorithm.html)
  Given parameters (in order) are:
      `counter` (starting with 1), `password_type` (maximum, long, medium, short, basic, or pin), `password`,
       `user`, and `site`. The following line generates a long password for the user "user" and with a master-password "password"  on the site "example.com":
       ```
       {{ derivePassword 1 "long" "password" "user" "example.com" }}
       ```

## SemVer Functions:

These functions provide version parsing and comparisons for SemVer 2 version
strings.

- semver: Parse a semantic version and return a Version object.
- semverCompare: Compare a SemVer range to a particular version.

## Principles:

The following principles were used in deciding on which functions to add, and
determining how to implement them.

- Template functions should be used to build layout. Therefore, the following
  types of operations are within the domain of template functions:
  - Formatting
  - Layout
  - Simple type conversions
  - Utilities that assist in handling common formatting and layout needs (e.g. arithmetic)
- Template functions should not return errors unless there is no way to print
  a sensible value. For example, converting a string to an integer should not
  produce an error if conversion fails. Instead, it should display a default
  value that can be displayed.
- Simple math is necessary for grid layouts, pagers, and so on. Complex math
  (anything other than arithmetic) should be done outside of templates.
- Template functions only deal with the data passed into them. They never retrieve
  data from a source.
- Finally, do not override core Go template functions.
