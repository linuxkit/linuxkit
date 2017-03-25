GoUtils
===========

GoUtils provides users with utility functions to manipulate strings in various ways. It is a Go implementation of some 
string manipulation libraries of Java Apache Commons. GoUtils includes the following Java Apache Commons classes:
* WordUtils    
* RandomStringUtils  
* StringUtils (partial implementation)

## Installation
If you have Go set up on your system, from the GOPATH directory within the command line/terminal, enter this:

	go get github.com/aokoli/goutils
    
If you do not have Go set up on your system, please follow the [Go installation directions from the documenation](http://golang.org/doc/install), and then follow the instructions above to install GoUtils.


## Documentation 
GoUtils doc is available here: [![GoDoc](https://godoc.org/github.com/aokoli/goutils?status.png)](https://godoc.org/github.com/aokoli/goutils)


## Usage
The code snippets below show examples of how to use GoUtils. Some functions return errors while others do not. The first instance below, which does not return an error, is the `Initials` function (located within the `wordutils.go` file).

    package main
    
    import (
        "fmt"
    	"github.com/aokoli/goutils"
    )
    
    func main() {

    	// EXAMPLE 1: A goutils function which returns no errors
        fmt.Println (goutils.Initials("John Doe Foo")) // Prints out "JDF"

    }
Some functions return errors mainly due to illegal arguements used as parameters. The code example below illustrates how to deal with function that returns an error. In this instance, the function is the `Random` function (located within the `randomstringutils.go` file).

    package main
    
    import (
        "fmt"
        "github.com/aokoli/goutils"
    )
    
    func main() {

        // EXAMPLE 2: A goutils function which returns an error
        rand1, err1 := goutils.Random (-1, 0, 0, true, true)  

        if err1 != nil { 
			fmt.Println(err1) // Prints out error message because -1 was entered as the first parameter in goutils.Random(...)
		} else {
			fmt.Println(rand1) 
		}

    }

## License
GoUtils is licensed under the Apache License, Version 2.0. Please check the LICENSE.txt file or visit http://www.apache.org/licenses/LICENSE-2.0 for a copy of the license. 

## Issue Reporting
Make suggestions or report issues using the Git issue tracker: https://github.com/aokoli/goutils/issues

## Website
* [GoUtils webpage](http://aokoli.github.io/goutils/)

## Mailing List
Contact [okolialex@gmail.com](mailto:okolialex@mail.com) to be added to the mailing list. You will get updates on the 
status of the project and the potential direction it will be heading.

