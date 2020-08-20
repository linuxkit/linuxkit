#Snooze

Snooze is a type safe REST library that allows you to specify describe API endpoints as functions in Go.  Allows for fast integration with 3rd party APIs

## Example

``` go
package snooze

import "net/http"

type api struct {
	Login   func(loginData) (userData, error)             `method:"POST" path:"/auth/login"`
	Friends func(page int, count int) ([]userData, error) `method:"GET" path:"/me/friends?page={0}&count={1}"`
	Profile func(id string) (userData, error)             `method:"GET" path:"/user/{0}"`
}

func Example() {
	client := Client{
		Root: "http://example.com",
		Before: func(r *http.Request, c *http.Client) {
			values := r.URL.Query()
			values.Add("session", "123456")
			r.URL.RawQuery = values.Encode()
		}}

	api := new(api)
	client.Create(api)

	api.Login(loginData{"test@example.com", "password"})
	api.Friends(1, 100)
	api.Profile("1234")
}

type loginData struct {
	Email    string
	Password string
}

type userData struct {
	loginData
	Id      string
	Picture string
}
```
