package main

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/sunshineplan/cipher"
	"github.com/sunshineplan/utils/httpproxy"
)

var proxyURL *url.URL
var proxy *httpproxy.Proxy

func initProxy() {
	var err error
	proxyURL, err = url.Parse("https://" + *server)
	if err != nil {
		log.Fatalln("bad server address:", *server)
	}
	proxy = httpproxy.New(proxyURL, nil)
	svr.Handler = http.HandlerFunc(clientHandler)
}

func clientTunneling(w http.ResponseWriter, r *http.Request) {
	dest_conn, _, err := proxy.DialWithHeader(r.Host, r.Header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
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
	tr := &http.Transport{
		Proxy:        http.ProxyURL(proxyURL),
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	}
	resp, err := tr.RoundTrip(r)
	if err != nil {
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

func clientHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Session-Authorization", cipher.EncryptText(*psk, *username+":"+*password))

	accessLogger.Printf("%s %s", r.Method, r.URL)
	if r.Method == http.MethodConnect {
		clientTunneling(w, r)
	} else {
		clientHTTP(w, r)
	}
}
