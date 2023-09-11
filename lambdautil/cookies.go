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

func Cookie2(cookies []string, name string) *http.Cookie {
	for _, cookie := range Cookies2(cookies) {
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

func Cookies2(cookies []string) []*http.Cookie {
	req := &http.Request{Header: http.Header{
		"Cookie": cookies,
	}}
	return req.Cookies()
}
