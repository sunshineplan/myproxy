package main

import (
	"bufio"
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/sunshineplan/cipher"
	"github.com/sunshineplan/utils/httpproxy"
)

var p *httpproxy.Proxy

func initProxy() {
	var forwardProxy httpproxy.Dialer
	if *forward != "" {
		forwardURL, err := url.Parse(*forward)
		if err != nil {
			log.Fatalln("bad forward proxy:", *forward)
		}
		forwardProxy = httpproxy.New(forwardURL, nil)
	} else {
		forwardProxy = httpproxy.Direct
	}
	proxyURL, err := url.Parse("https://" + *server)
	if err != nil {
		log.Fatalln("bad server address:", *server)
	}
	p = httpproxy.New(proxyURL, forwardProxy)
	if *debug {
		log.Print("Proxy ready")
	}
}

func clientTunneling(w http.ResponseWriter, r *http.Request) {
	dest_conn, resp, err := p.DialWithHeader(r.Host, r.Header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if resp.StatusCode != http.StatusOK {
		http.Error(w, resp.Status, resp.StatusCode)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func clientHTTP(w http.ResponseWriter, r *http.Request) {
	port := r.URL.Port()
	if port == "" {
		port = "80"
	}
	conn, resp, err := p.DialWithHeader(r.Host+":"+port, r.Header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		http.Error(w, resp.Status, resp.StatusCode)
		return
	}

	if err := r.Write(conn); err != nil {
		conn.Close()
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	br := bufio.NewReader(conn)
	resp, err = http.ReadResponse(br, r)
	if err != nil {
		conn.Close()
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	header := w.Header()
	for k, vv := range resp.Header {
		for _, v := range vv {
			header.Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

func clientHandler(w http.ResponseWriter, r *http.Request) {
	user := "anonymous"
	var pass string
	var ok bool
	if len(accounts) != 0 {
		user, pass, ok = parseBasicAuth(r.Header.Get("Proxy-Authorization"))
		if !ok {
			accessLogger.Printf("%s Proxy Authentication Required", r.RemoteAddr)
			w.Header().Add("Proxy-Authenticate", `Basic realm="My Proxy"`)
			http.Error(w, "", http.StatusProxyAuthRequired)
			return
		} else if !hasAccount(user, pass) {
			errorLogger.Printf("%s Proxy Authentication Failed", r.RemoteAddr)
			w.Header().Add("Proxy-Authenticate", `Basic realm="My Proxy"`)
			http.Error(w, "", http.StatusProxyAuthRequired)
			return
		}
		r.Header.Del("Proxy-Authorization")
	}

	if *autoproxy && !match(r.URL) {
		accessLogger.Printf("[direct] %s[%s] %s %s", r.RemoteAddr, user, r.Method, r.URL)
		if r.Method == http.MethodConnect {
			serverTunneling(w, r)
		} else {
			serverHTTP(w, r)
		}
		return
	}

	r.Header.Set(*header, cipher.EncryptText(*psk, *username+":"+*password))

	accessLogger.Printf("%s[%s] %s %s", r.RemoteAddr, user, r.Method, r.URL)
	if r.Method == http.MethodConnect {
		clientTunneling(w, r)
	} else {
		clientHTTP(w, r)
	}
}
