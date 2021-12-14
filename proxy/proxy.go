package proxy

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

var (
	// API Key
	APIKey = os.Getenv("M3O_API_KEY")

	// API Url
	APIHost = "https://api.m3o.com"

	// host to proxy for URLs
	URLHost = "m3o.one"

	// host to proxy for Apps
	AppHost = "m3o.app"
)

func appProxy(w http.ResponseWriter, r *http.Request) {
	subdomain := strings.TrimSuffix(r.Host, "."+AppHost)

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

	r.Host = uri.Host
	r.Header.Set("Host", r.Host)

	httputil.NewSingleHostReverseProxy(uri).ServeHTTP(w, r)
}

func urlProxy(w http.ResponseWriter, r *http.Request) {
	// assuming /u/short-id
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		return
	}
	// assuming its /u/ for url
	if parts[1] != "u" {
		return
	}

	// get the url id
	//id := parts[2]

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

	apiURL := APIHost + "/url/proxy"

	// use /v1/
	if len(APIKey) > 0 {
		apiURL = APIHost + "/v1/url/proxy"
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
	http.Redirect(w, r, url, 301)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

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
}
