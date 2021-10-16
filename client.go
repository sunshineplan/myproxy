package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/sunshineplan/cipher"
	"github.com/sunshineplan/utils/httpproxy"
)

var proxy *httpproxy.Proxy

func initProxy() {
	var forwardProxy *httpproxy.Proxy
	if *forward != "" {
		forwardURL, err := url.Parse(*forward)
		if err != nil {
			log.Fatalln("bad forward proxy:", *forward)
		}
		forwardProxy = httpproxy.New(forwardURL, nil)
	}
	proxyURL, err := url.Parse("https://" + *server)
	if err != nil {
		log.Fatalln("bad server address:", *server)
	}
	proxy = httpproxy.New(proxyURL, forwardProxy)
}

func clientTunneling(w http.ResponseWriter, r *http.Request) {
	dest_conn, resp, err := proxy.DialWithHeader(r.Host, r.Header)
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
	conn, resp, err := proxy.DialWithHeader(r.Host+":"+port, r.Header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		http.Error(w, resp.Status, resp.StatusCode)
		return
	}

	r.Header.Del(*header)
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

func clientHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set(*header, cipher.EncryptText(*psk, *username+":"+*password))

	accessLogger.Printf("%s %s", r.Method, r.URL)
	if r.Method == http.MethodConnect {
		clientTunneling(w, r)
	} else {
		clientHTTP(w, r)
	}
}
