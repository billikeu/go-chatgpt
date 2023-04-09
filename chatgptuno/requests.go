package chatgptuno

import (
	"bytes"
	"io"

	http "github.com/bogdanfinn/fhttp"

	tls_client "github.com/bogdanfinn/tls-client"
)

const UA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36"

type Requests struct {
	jar          http.CookieJar
	headers      http.Header
	timeout      int
	proxy        string
	allowRediret bool
	response     *http.Response
	reqReader    io.Reader
	cookies      map[string]string
}

func NewRequests(jar http.CookieJar) *Requests {
	r := &Requests{
		jar:          jar,
		headers:      http.Header{},
		timeout:      60,
		allowRediret: true,
		reqReader:    bytes.NewReader([]byte(``)),
		cookies:      make(map[string]string, 0),
	}
	return r
}

func (r *Requests) SetHeaders(headers http.Header) {
	r.headers = headers
}

func (r *Requests) SetCookie(name, value string) {
	r.cookies[name] = value
}

func (r *Requests) SetTimeout(timeout int) {
	if timeout > 0 {
		r.timeout = timeout
	}

}

func (r *Requests) SetProxy(proxy string) {
	r.proxy = proxy
}

func (r *Requests) SetNoRedirects() {
	r.allowRediret = false
}

func (r *Requests) Do(method, baseUrl string) (*http.Response, error) {
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(r.timeout),
		tls_client.WithClientProfile(tls_client.Chrome_110),
		tls_client.WithCookieJar(r.jar),
	}
	if r.allowRediret {
		options = append(options, tls_client.WithNotFollowRedirects())
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, baseUrl, r.reqReader)
	if err != nil {
		return nil, err
	}
	req.Header = r.headers
	if r.proxy != "" {
		client.SetProxy(r.proxy)
	}
	resp, err := client.Do(req)
	r.response = resp
	return resp, err
}

func (r *Requests) SetBody(reader io.Reader) {
	r.reqReader = reader
}

func (r *Requests) Get(baseUrl string) (*http.Response, error) {
	return r.Do(http.MethodGet, baseUrl)
}

func (r *Requests) Post(baseUrl string) (*http.Response, error) {
	return r.Do(http.MethodPost, baseUrl)
}

func (r *Requests) Patch(baseUrl string) (*http.Response, error) {
	return r.Do(http.MethodPatch, baseUrl)
}
