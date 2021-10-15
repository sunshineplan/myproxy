package main

import (
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sunshineplan/cipher"
)

func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}

func serverTunneling(w http.ResponseWriter, r *http.Request) {
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
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

func serverHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(r)
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

func parseBasicAuth(auth string) (username, password string) {
	if auth == "" {
		return
	}
	c, err := cipher.DecryptText(*psk, auth)
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:]
}

func serverHandler(w http.ResponseWriter, r *http.Request) {
	user, pass := parseBasicAuth(r.Header.Get("Session-Authorization"))
	if !hasAccount(user, pass) {
		errorLogger.Printf("%s Authentication Failed", r.RemoteAddr)
		http.Error(w, "", http.StatusNoContent)
		return
	}

	accessLogger.Printf("%s[%s] %s %s", r.RemoteAddr, user, r.Method, r.URL)
	if r.Method == http.MethodConnect {
		serverTunneling(w, r)
	} else {
		serverHTTP(w, r)
	}
}
