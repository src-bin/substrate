package lambdautil

import (
	"net/http"
)

func Cookie(headers map[string][]string, name string) *http.Cookie {
	for _, cookie := range Cookies(headers) {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func Cookies(headers map[string][]string) []*http.Cookie {
	req := &http.Request{Header: http.Header{
		"Cookie": headers["cookie"], // beware the case-sensitivity
	}}
	return req.Cookies()
}
