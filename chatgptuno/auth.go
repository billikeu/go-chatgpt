package chatgptuno

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	// "net/http"
	"net/url"
	"regexp"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/tidwall/gjson"
)

// const UA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36"

type AuthConfig struct {
	EmailAddr string
	Passwd    string
	Proxy     string
}

// OpenAI Authentication Reverse Engineered
type Authenticator struct {
	SessionToken string
	CsrfToken    string
	accessToken  string
	jar          tls_client.CookieJar
	cfg          *AuthConfig
}

func NewAuthenticator(cfg *AuthConfig) *Authenticator {
	auth := &Authenticator{
		jar: tls_client.NewCookieJar(),
		cfg: cfg,
	}
	return auth
}

func (auth *Authenticator) Loin() error {
	endpint := "https://explorer.api.openai.com/api/auth/csrf"
	headers := http.Header{
		"Host":            {"explorer.api.openai.com"},
		"Accept":          {"*/*"},
		"Connection":      {"keep-alive"},
		"User-Agent":      {UA},
		"Accept-Language": {"en-GB,en-US;q=0.9,en;q=0.8"},
		"Referer":         {"https://explorer.api.openai.com/auth/login"},
		"Accept-Encoding": {"gzip, deflate, br"},
	}
	client := NewRequests(auth.jar)
	client.SetProxy(auth.cfg.Proxy)
	client.SetTimeout(30)
	client.SetHeaders(headers)
	resp, err := client.Get(endpint)
	if err != nil {
		return fmt.Errorf("login openai failed: %s", err.Error())
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read request err:%s", err.Error())
	}
	body := string(resBody)

	if resp.StatusCode == 200 && strings.Contains(resp.Header.Get("Content-Type"), "json") {
		j := gjson.Parse(body)
		csrfToken := j.Get("csrfToken").String()
		if csrfToken == "" {
			return fmt.Errorf("login openai failed: %s, %d", body, resp.StatusCode)
		}
		auth.CsrfToken = csrfToken
		auth.partOne(csrfToken)
		return nil
	}
	return fmt.Errorf("login openai failed: %s, %d", body, resp.StatusCode)
}

func (auth *Authenticator) partOne(csrfToken string) error {
	endpoint := "https://explorer.api.openai.com/api/auth/signin/auth0?prompt=login"
	payload := `callbackUrl=%2F&` + fmt.Sprintf(`csrfToken=%s&json=true`, csrfToken)
	headers := http.Header{
		"Host":            {"explorer.api.openai.com"},
		"User-Agent":      {UA},
		"Content-Type":    {"application/x-www-form-urlencoded"},
		"Accept":          {"*/*"},
		"Sec-Gpc":         {"1"},
		"Accept-Language": {"en-US,en;q=0.8"},
		"Origin":          {"https://explorer.api.openai.com"},
		"Sec-Fetch-Site":  {"same-origin"},
		"Sec-Fetch-Mode":  {"cors"},
		"Sec-Fetch-Dest":  {"empty"},
		"Referer":         {"https://explorer.api.openai.com/auth/login"},
		"Accept-Encoding": {"gzip, deflate"},
	}
	client := NewRequests(auth.jar)
	client.SetProxy(auth.cfg.Proxy)
	client.SetTimeout(30)
	client.SetHeaders(headers)
	client.SetBody(bytes.NewReader([]byte(payload)))
	resp, err := client.Post(endpoint)
	if err != nil {
		return fmt.Errorf("login part one failed: %s", err.Error())
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read part one err:%s", err.Error())
	}
	body := string(resBody)

	if resp.StatusCode == 200 && strings.Contains(resp.Header.Get("Content-Type"), "json") {
		j := gjson.Parse(body)
		url := j.Get("url").String()
		if url == "https://explorer.api.openai.com/api/auth/error?error=OAuthSignin" || strings.Contains(url, "error") {
			return errors.New("you have been rate limited. Please try again later")
		}
		// part_two
		err = auth.partTwo(url)
		if err != nil {
			return err
		}
		return nil

	}
	return fmt.Errorf("part one failed: %s, %d", body, resp.StatusCode)
}

func (auth *Authenticator) partTwo(endpoint string) error {
	headers := http.Header{
		"Host":            {"auth0.openai.com"},
		"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"Connection":      {"keep-alive"},
		"User-Agent":      {UA},
		"Accept-Language": {"en-US,en;q=0.9"},
		"Referer":         {"https://explorer.api.openai.com/"},
	}
	client := NewRequests(auth.jar)
	client.SetProxy(auth.cfg.Proxy)
	client.SetTimeout(30)
	client.SetHeaders(headers)
	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("login partTwo failed: %s", err.Error())
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read partTwo err:%s", err.Error())
	}
	body := string(resBody)
	if resp.StatusCode != 302 && resp.StatusCode != 200 {
		return fmt.Errorf("login openai partTwo failed:%s, %d", body, resp.StatusCode)
	}
	reg := regexp.MustCompile(`state=(.*)`)
	if reg == nil {
		return errors.New("login openai partTwo regexp err")
	}
	r := reg.FindAllStringSubmatch(body, -1)
	if len(r) == 0 {
		return fmt.Errorf("login openai partTwo failed:%s, %d", body, resp.StatusCode)
	}
	state := strings.Split(r[0][0], `"`)[0]
	err = auth.partThree(state)
	if err != nil {
		return err
	}
	return nil
}

// We use the state to get the login page
func (auth *Authenticator) partThree(state string) error {
	endpoint := fmt.Sprintf("https://auth0.openai.com/u/login/identifier?state=%s", state)
	headers := http.Header{
		"Host":            {"auth0.openai.com"},
		"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"Connection":      {"keep-alive"},
		"User-Agent":      {UA},
		"Accept-Language": {"en-US,en;q=0.9"},
		"Referer":         {"https://explorer.api.openai.com/"},
	}
	client := NewRequests(auth.jar)
	client.SetProxy(auth.cfg.Proxy)
	client.SetTimeout(30)
	client.SetHeaders(headers)
	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("login partThree failed: %s", err.Error())
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read partThree err:%s", err.Error())
	}
	body := string(resBody)
	if resp.StatusCode != 200 {
		return fmt.Errorf("login openai partThree failed:%s, %d", body, resp.StatusCode)
	}
	err = auth.partFour(state)
	if err != nil {
		return err
	}
	return nil
}

// We make a POST request to the login page with the captcha, email
func (auth *Authenticator) partFour(state string) error {
	endpoint := fmt.Sprintf("https://auth0.openai.com/u/login/identifier?state=%s", state)
	encodeEmail := url.QueryEscape(auth.cfg.EmailAddr) //123@gmail.com --> 123%40gmail.com
	payload := fmt.Sprintf("state=%s&username=%s&js-available=false&webauthn-available=true&is", state, encodeEmail)
	payload += "-brave=false&webauthn-platform-available=true&action=default "
	headers := http.Header{
		"Host":            {"auth0.openai.com"},
		"Origin":          {"https://auth0.openai.com"},
		"Connection":      {"keep-alive"},
		"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"User-Agent":      {UA},
		"Referer":         {fmt.Sprintf("https://auth0.openai.com/u/login/identifier?state=%s", state)},
		"Accept-Language": {"en-US,en;q=0.9"},
		"Content-Type":    {"application/x-www-form-urlencoded"},
	}
	client := NewRequests(auth.jar)
	client.SetProxy(auth.cfg.Proxy)
	client.SetTimeout(30)
	client.SetHeaders(headers)
	client.SetBody(bytes.NewBuffer([]byte(payload)))
	resp, err := client.Post(endpoint)
	if err != nil {
		return fmt.Errorf("login partFour failed: %s", err.Error())
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read partFour err:%s", err.Error())
	}
	body := string(resBody)
	if resp.StatusCode != 302 && resp.StatusCode != 200 {
		return fmt.Errorf("login openai partFour failed:%s, %d", body, resp.StatusCode)
	}
	err = auth.partFive(state)
	if err != nil {
		return err
	}
	return nil
}

// We enter the password
func (auth *Authenticator) partFive(state string) error {
	endpoint := fmt.Sprintf("https://auth0.openai.com/u/login/password?state=%s", state)
	encodeEmail := url.QueryEscape(auth.cfg.EmailAddr)
	encodedPasswd := url.QueryEscape(auth.cfg.Passwd)
	payload := fmt.Sprintf("state=%s&username=%s&password=%s&action=default", state, encodeEmail, encodedPasswd)
	headers := http.Header{
		"Host":            {"auth0.openai.com"},
		"Origin":          {"https://auth0.openai.com"},
		"Connection":      {"keep-alive"},
		"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"User-Agent":      {UA},
		"Referer":         {fmt.Sprintf("https://auth0.openai.com/u/login/password?state=%s", state)},
		"Accept-Language": {"en-US,en;q=0.9"},
		"Content-Type":    {"application/x-www-form-urlencoded"},
	}
	client := NewRequests(auth.jar)
	client.SetProxy(auth.cfg.Proxy)
	client.SetTimeout(30)
	client.SetHeaders(headers)
	client.SetBody(bytes.NewBuffer([]byte(payload)))
	resp, err := client.Post(endpoint)
	if err != nil {
		return fmt.Errorf("login partFive failed: %s", err.Error())
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read partFive err:%s", err.Error())
	}
	body := string(resBody)
	if resp.StatusCode != 302 && resp.StatusCode != 200 {
		return fmt.Errorf("login openai partFive failed, your credentials are invalid.:%s, %d", body, resp.StatusCode)
	}
	reg := regexp.MustCompile(`state=(.*)`)
	if reg == nil {
		return errors.New("login openai partFive regexp err")
	}
	r := reg.FindAllStringSubmatch(body, -1)
	if len(r) == 0 {
		return fmt.Errorf("login openai partFive failed:%s, %d", body, resp.StatusCode)
	}
	newState := strings.Split(r[0][0], `"`)[0]
	err = auth.partSix(state, newState)
	if err != nil {
		return err
	}
	return nil
}

func (auth *Authenticator) partSix(state, newState string) error {
	endpoint := fmt.Sprintf("https://auth0.openai.com/authorize/resume?state=%s", newState)
	headers := http.Header{
		"Host":            {"auth0.openai.com"},
		"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"Connection":      {"keep-alive"},
		"User-Agent":      {UA},
		"Accept-Language": {"en-GB,en-US;q=0.9,en;q=0.8"},
		"Referer":         {fmt.Sprintf("https://auth0.openai.com/u/login/password?state=%s", state)},
	}
	client := NewRequests(auth.jar)
	client.SetProxy(auth.cfg.Proxy)
	client.SetTimeout(30)
	client.SetHeaders(headers)
	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("login partSix failed: %s", err.Error())
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read partSix err:%s", err.Error())
	}
	body := string(resBody)
	if resp.StatusCode != 302 {
		return fmt.Errorf("login openai partSix failed:%s, %d", body, resp.StatusCode)
	}
	redirectUrl := resp.Header.Get("location")
	err = auth.partSeven(redirectUrl, endpoint)
	if err != nil {
		return err
	}
	return nil
}

func (auth *Authenticator) partSeven(redirectUrl, previousUrl string) error {
	endpoint := redirectUrl
	headers := http.Header{
		"Host":            {"explorer.api.openai.com"},
		"Accept":          {"application/json"},
		"Connection":      {"keep-alive"},
		"User-Agent":      {UA},
		"Accept-Language": {"en-GB,en-US;q=0.9,en;q=0.8"},
		"Referer":         {previousUrl},
	}
	client := NewRequests(auth.jar)
	client.SetProxy(auth.cfg.Proxy)
	client.SetTimeout(30)
	client.SetHeaders(headers)
	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("login partSeven failed: %s", err.Error())
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read partSeven err:%s", err.Error())
	}
	body := string(resBody)
	if resp.StatusCode != 302 {
		return fmt.Errorf("login openai partSeven failed:%s, %d", body, resp.StatusCode)
	}

	for _, item := range resp.Cookies() {
		if item.Name == "__Secure-next-auth.session-token" {
			auth.SessionToken = item.Value
			return auth.GetAccessToken()
		}
	}
	return fmt.Errorf("login openai partSeven failed:%s, %d", body, resp.StatusCode)
}

// Gets access token
func (auth *Authenticator) GetAccessToken() error {
	// auth.jar.SetCookies()
	endpoint := "https://explorer.api.openai.com/api/auth/session"
	u, _ := url.Parse(endpoint)
	auth.jar.SetCookies(u, []*http.Cookie{
		{
			Name:  "__Secure-next-auth.session-token",
			Value: auth.SessionToken,
		},
	})
	client := NewRequests(auth.jar)
	client.SetProxy(auth.cfg.Proxy)
	client.SetTimeout(30)
	// client.SetCookie()
	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("login getAccessToken failed: %s", err.Error())
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read getAccessToken err:%s", err.Error())
	}
	body := string(resBody)
	if resp.StatusCode != 200 {
		return fmt.Errorf("login openai get access token failed:%s, %d", body, resp.StatusCode)
	}
	j := gjson.Parse(body)
	accessToken := j.Get("accessToken").String()
	if accessToken == "" {
		return fmt.Errorf("login openai get access token failed:%s, %d", body, resp.StatusCode)
	}
	auth.accessToken = accessToken
	return nil
}

func (auth *Authenticator) AccessToken() string {
	return auth.accessToken
}
