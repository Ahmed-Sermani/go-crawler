/*
   Manages the crawling of links
*/

package crawler

import "net/http"

// URLGetter is implemented by objects that can perform HTTP GET requests.
type URLGetter interface {
	Get(url string) (*http.Response, error)
}

// PrivateNetworkDetector is implemented by objects that can detect whether a
// host resolves to a private network address.
// used as a secuity machnism to prevent exposing internal services to the crawler
type PrivateNetworkDetector interface {
	IsPrivate(host string) (bool, error)
}
