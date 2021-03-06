package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	// API Key
	APIKey = os.Getenv("M3O_API_KEY")

	// API Url
	APIHost = "https://api.m3o.com"

	// dot com host
	ComHost = "m3o.com"

	// host to proxy for URLs
	URLHost = "m3o.one"

	// host to proxy for Apps
	AppHost = "m3o.app"

	// host to proxy for Functions
	FunctionHost = "m3o.sh"

	// community host
	CommunityHost = "community.m3o.com"

	// host for user auth
	UserHost = "user.m3o.com"
)

var (
	mtx sync.RWMutex

	// local cache
	appMap = map[string]*backend{}

	ftx sync.RWMutex

	functionMap = map[string]*backend{}
)

type backend struct {
	url     *url.URL
	created time.Time
}

func functionProxy(w http.ResponseWriter, r *http.Request) {
	// no subdomain
	if r.Host == FunctionHost {
		return
	}

	// lookup the app map
	ftx.RLock()
	bk, ok := functionMap[r.Host]
	ftx.RUnlock()

	// check the url map
	if ok && time.Since(bk.created) < time.Minute {
		r.Host = bk.url.Host
		r.Header.Set("Host", r.Host)
		httputil.NewSingleHostReverseProxy(bk.url).ServeHTTP(w, r)
		return
	}

	subdomain := strings.TrimSuffix(r.Host, "."+FunctionHost)

	// only process one part for now
	parts := strings.Split(subdomain, ".")
	if len(parts) > 1 {
		log.Print("[function/proxy] more parts than expected", parts)
		return
	}

	// currently service id is the subdomain
	id := subdomain

	log.Printf("[function/proxy] resolving host %s to id %s\n", r.Host, id)

	apiURL := APIHost + "/function/proxy"

	// use /v1/
	if len(APIKey) > 0 {
		apiURL = APIHost + "/v1/function/proxy"
	}

	// make new request
	log.Printf("[function/proxy] Calling: %v", apiURL+"?id="+id)
	req, err := http.NewRequest("GET", apiURL+"?id="+id, nil)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if req.Header == nil {
		req.Header = make(http.Header)
	}

	// set the api key after we're given the header
	if len(APIKey) > 0 {
		req.Header.Set("Authorization", "Bearer "+APIKey)
	}

	// call the backend for the url
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if rsp.StatusCode != 200 {
		log.Printf("[function/proxy] Error calling api: status: %v %v", rsp.StatusCode, string(b))
		http.Error(w, "unexpected error", 500)
		return
	}

	result := map[string]interface{}{}

	if err := json.Unmarshal(b, &result); err != nil {
		log.Print("[function/proxy] failed to unmarshal response")
		http.Error(w, err.Error(), 500)
		return
	}

	// get the destination url
	u, _ := result["url"].(string)
	if len(u) == 0 {
		log.Print("[function/proxy] no response url")
		return
	}

	uri, err := url.Parse(u)
	if err != nil {
		log.Print("[function/proxy] failed to parse url", err.Error())
		return
	}

	ftx.Lock()
	functionMap[r.Host] = &backend{
		url:     uri,
		created: time.Now(),
	}
	ftx.Unlock()

	r.Host = uri.Host
	r.Header.Set("Host", r.Host)

	httputil.NewSingleHostReverseProxy(uri).ServeHTTP(w, r)
}

func appProxy(w http.ResponseWriter, r *http.Request) {
	// no subdomain
	if r.Host == AppHost {
		return
	}

	// lookup the app map
	mtx.RLock()
	bk, ok := appMap[r.Host]
	mtx.RUnlock()

	// check the url map
	if ok && time.Since(bk.created) < time.Minute {
		r.Host = bk.url.Host
		r.Header.Set("Host", r.Host)
		httputil.NewSingleHostReverseProxy(bk.url).ServeHTTP(w, r)
		return
	}

	subdomain := strings.TrimSuffix(r.Host, "."+AppHost)
	subdomain = strings.TrimSuffix(subdomain, "."+ComHost)

	// only process one part for now
	parts := strings.Split(subdomain, ".")
	if len(parts) > 1 {
		log.Print("[app/proxy] more parts than expected", parts)
		return
	}

	// currently service id is the subdomain
	id := subdomain

	log.Printf("[app/proxy] resolving host %s to id %s\n", r.Host, id)

	apiURL := APIHost + "/app/resolve"

	// use /v1/
	if len(APIKey) > 0 {
		apiURL = APIHost + "/v1/app/resolve"
	}

	// make new request
	log.Printf("[app/proxy] Calling: %v", apiURL+"?id="+id)
	req, err := http.NewRequest("GET", apiURL+"?id="+id, nil)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if req.Header == nil {
		req.Header = make(http.Header)
	}

	// set the api key after we're given the header
	if len(APIKey) > 0 {
		req.Header.Set("Authorization", "Bearer "+APIKey)
	}

	// call the backend for the url
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if rsp.StatusCode != 200 {
		log.Printf("[app/proxy] Error calling api: status: %v %v", rsp.StatusCode, string(b))
		http.Error(w, "unexpected error", 500)
		return
	}

	result := map[string]interface{}{}

	if err := json.Unmarshal(b, &result); err != nil {
		log.Print("[app/proxy] failed to unmarshal response")
		http.Error(w, err.Error(), 500)
		return
	}

	// get the destination url
	u, _ := result["url"].(string)
	if len(u) == 0 {
		log.Print("[app/proxy] no response url")
		return
	}

	uri, err := url.Parse(u)
	if err != nil {
		log.Print("[app/proxy] failed to parse url", err.Error())
		return
	}

	mtx.Lock()
	appMap[r.Host] = &backend{
		url:     uri,
		created: time.Now(),
	}
	mtx.Unlock()

	r.Host = uri.Host
	r.Header.Set("Host", r.Host)

	httputil.NewSingleHostReverseProxy(uri).ServeHTTP(w, r)
}

func urlProxy(w http.ResponseWriter, r *http.Request) {
	uri := url.URL{
		Scheme: r.URL.Scheme,
		Host:   r.URL.Host,
		Path:   r.URL.Path,
	}

	// if the host is blank we have to set it
	if len(uri.Host) == 0 {
		if len(r.Host) > 0 {
			log.Printf("[url/proxy] Host is set from r.Host %v", r.Host)
			uri.Host = r.Host
		} else {
			log.Printf("[url/proxy] Host is nil, defaulting to: %v", URLHost)
			uri.Host = URLHost
			uri.Scheme = "https"
		}
	}

	if len(uri.Scheme) == 0 {
		uri.Scheme = "https"
	}

	apiURL := APIHost + "/url/resolve"

	// use /v1/
	if len(APIKey) > 0 {
		apiURL = APIHost + "/v1/url/resolve"
	}

	// make new request
	log.Printf("[url/proxy] Calling: %v", apiURL+"?shortURL="+uri.String())
	req, err := http.NewRequest("GET", apiURL+"?shortURL="+uri.String(), nil)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if req.Header == nil {
		req.Header = make(http.Header)
	}

	// set the api key after we're given the header
	if len(APIKey) > 0 {
		req.Header.Set("Authorization", "Bearer "+APIKey)
	}

	// call the backend for the url
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if rsp.StatusCode != 200 {
		log.Printf("[url/proxy] Error calling api: status: %v %v", rsp.StatusCode, string(b))
		http.Error(w, "unexpected error", 500)
		return
	}

	result := map[string]interface{}{}

	if err := json.Unmarshal(b, &result); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// get the destination url
	url, _ := result["destinationURL"].(string)
	if len(url) == 0 {
		return
	}

	// return the redirect url to caller
	http.Redirect(w, r, url, 302)
}

func userProxy(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// can't operate without the api key
	if len(APIKey) == 0 {
		return
	}

	token := r.Form.Get("token")
	if len(token) == 0 {
		log.Print("Missing token")
		return
	}

	redirectUrl := r.Form.Get("redirectUrl")
	if len(redirectUrl) == 0 {
		log.Print("Missing redirect url")
		return
	}

	// check access and redirect
	uri := APIHost + "/v1/user/VerifyEmail"

	b, _ := json.Marshal(map[string]interface{}{
		"token": token,
	})

	req, err := http.NewRequest("POST", uri, bytes.NewReader(b))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	req.Header.Set("Authorization", "Bearer "+APIKey)

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rsp.Body.Close()

	// discard the body
	io.Copy(ioutil.Discard, rsp.Body)

	if rsp.StatusCode == 200 {
		http.Redirect(w, r, redirectUrl, 302)
		return
	}

	// non 200 status code

	// redirect to failure url
	failureUrl := r.Form.Get("failureRedirectUrl")
	if len(failureUrl) == 0 {
		return
	}

	// redirect
	http.Redirect(w, r, failureUrl, 302)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	// m3o.one
	if strings.HasSuffix(r.Host, URLHost) {
		urlProxy(w, r)
		return
	}

	// m3o.app
	if strings.HasSuffix(r.Host, AppHost) {
		appProxy(w, r)
		return
	}

	// m3o.sh
	if strings.HasSuffix(r.Host, FunctionHost) {
		functionProxy(w, r)
		return
	}

	if r.Host == UserHost {
		userProxy(w, r)
		return
	}

	if r.Host == CommunityHost {
		appProxy(w, r)
		return
	}
}
