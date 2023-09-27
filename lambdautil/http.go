package lambdautil

import "net/url"

// SingleToMultiValue takes the API Gateway-provided map[string]string
// representations of headers, query string parameters, etc. and turns them
// into url.Values (which is a map[string][]string with more methods) that
// is actually useful for constructing and manipulating URLs and requests.
func SingleToMultiValue(m map[string]string) url.Values {
	values := url.Values{}
	for k, v := range m {
		values[k] = []string{v}
	}
	return values
}
