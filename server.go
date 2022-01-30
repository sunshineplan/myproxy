package main

import (
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sunshineplan/cipher"
)

func transfer(dst io.WriteCloser, src io.ReadCloser, user string) {
	defer dst.Close()
	defer src.Close()
	n, _ := io.Copy(dst, src)
	if user != "" {
		count(user, uint64(n))
	}
}

func serverTunneling(user string, w http.ResponseWriter, r *http.Request) {
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

	go transfer(dest_conn, client_conn, "")
	go transfer(client_conn, dest_conn, user)
}

func serverHTTP(user string, w http.ResponseWriter, r *http.Request) {
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
	n, _ := io.Copy(w, resp.Body)
	count(user, uint64(n))
}

func parseAuth(auth string) (username, password string) {
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
	user, pass := parseAuth(r.Header.Get(*header))
	if !hasAccount(user, pass) {
		errorLogger.Printf("%s Authentication Failed", r.RemoteAddr)
		http.Error(w, "", http.StatusNoContent)
		return
	}
	r.Header.Del(*header)

	accessLogger.Printf("%s[%s] %s %s", r.RemoteAddr, user, r.Method, r.URL)
	if r.Method == http.MethodConnect {
		serverTunneling(user, w, r)
	} else {
		serverHTTP(user, w, r)
	}
}
