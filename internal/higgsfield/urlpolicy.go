package higgsfield

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	productionAPIHost = "api.higgsfield.ai"
	cdnHost           = "cdn.higgsfield.ai"
)

// pinClient keeps the caller's transport and timeout, and refuses redirects
// that leave the host of the previous request. Cookies are not forwarded.
func pinClient(httpClient *http.Client) *http.Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &http.Client{
		Transport:     httpClient.Transport,
		Timeout:       httpClient.Timeout,
		CheckRedirect: refuseCrossOriginRedirect,
	}
}

func refuseCrossOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	if req.URL == nil || req.URL.User != nil {
		return fmt.Errorf("refusing redirect")
	}
	prev := via[len(via)-1]
	if prev.URL == nil || !sameOrigin(prev.URL, req.URL) {
		return fmt.Errorf("refusing redirect to %s", req.URL.Host)
	}
	return nil
}

func (c *Client) requireAPIOrigin(raw string) error {
	base, err := url.Parse(c.baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return fmt.Errorf("api base url is invalid")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("refusing url")
	}
	if u.User != nil {
		return fmt.Errorf("refusing url with credentials")
	}
	if strings.Contains(u.Path, "..") || strings.Contains(u.EscapedPath(), "..") {
		return fmt.Errorf("refusing url path")
	}
	if !sameOrigin(base, u) {
		return fmt.Errorf("refusing url on %s", u.Host)
	}
	return nil
}

// allowMedia permits the configured API origin (tests, and same-host files)
// and, for the production API only, HTTPS CDN hosts used by Higgsfield.
func (c *Client) allowMedia(u *url.URL) error {
	if u == nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("media url is invalid")
	}
	if u.User != nil {
		return fmt.Errorf("refusing media url with credentials")
	}
	base, err := url.Parse(c.baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return fmt.Errorf("api base url is invalid")
	}
	if sameOrigin(base, u) {
		return nil
	}
	if !productionAPI(base) {
		return fmt.Errorf("refusing media url on %s", u.Host)
	}
	if u.Scheme != "https" || portOrDefault(u) != "443" {
		return fmt.Errorf("media url must be https")
	}
	if !allowedCDNHost(u.Hostname()) {
		return fmt.Errorf("refusing media url on %s", u.Host)
	}
	return nil
}

func productionAPI(base *url.URL) bool {
	return strings.EqualFold(base.Scheme, "https") &&
		strings.EqualFold(base.Hostname(), productionAPIHost) &&
		portOrDefault(base) == "443"
}

func allowedCDNHost(host string) bool {
	host = strings.ToLower(host)
	if host == cdnHost {
		return true
	}
	// Generation responses use a CloudFront distribution such as
	// d8j0ntlcm91z4.cloudfront.net. The suffix is public CDN, not a private net.
	return strings.HasSuffix(host, ".cloudfront.net") && strings.Count(host, ".") >= 2
}

func sameOrigin(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	if !strings.EqualFold(a.Scheme, b.Scheme) {
		return false
	}
	if !strings.EqualFold(a.Hostname(), b.Hostname()) {
		return false
	}
	return portOrDefault(a) == portOrDefault(b)
}

func portOrDefault(u *url.URL) string {
	if u == nil {
		return ""
	}
	if p := u.Port(); p != "" {
		return p
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}
