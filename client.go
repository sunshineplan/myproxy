package main

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/sunshineplan/cipher"
)

func clientTunneling(w http.ResponseWriter, r *http.Request) {
	// TODO
	//conn, err := net.DialTimeout("tcp", *server, 10*time.Second)
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusServiceUnavailable)
	//	return
	//}
	//defer conn.Close()
	//
	//colonPos := strings.LastIndex(*server, ":")
	//if colonPos == -1 {
	//	colonPos = len(*server)
	//}
	//hostname := (*server)[:colonPos]
	//
	//conn = tls.Client(conn, &tls.Config{ServerName: hostname, InsecureSkipVerify: true})
	//
	//if err := r.Write(conn); err != nil {
	//	http.Error(w, err.Error(), http.StatusServiceUnavailable)
	//	return
	//}
	//
	//br := bufio.NewReader(conn)
	//resp, err := http.ReadResponse(br, r)
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusServiceUnavailable)
	//	return
	//}
	//defer resp.Body.Close()
	//
	//header := w.Header()
	//for k, vv := range resp.Header {
	//	for _, v := range vv {
	//		header.Add(k, v)
	//	}
	//}
	//w.WriteHeader(resp.StatusCode)
	//io.Copy(w, resp.Body)
}

func clientHTTP(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse("https://" + *server)
	if err != nil {
		log.Fatalln("bad server address:", *server)
	}

	tr := &http.Transport{
		Proxy:              http.ProxyURL(u),
		ProxyConnectHeader: r.Header,
		TLSNextProto:       make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
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
