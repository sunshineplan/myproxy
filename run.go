package main

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func runServer() {
	svr.Handler = http.HandlerFunc(serverHandler)
	svr.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	svr.ReadTimeout = time.Minute * 10
	svr.ReadHeaderTimeout = time.Second * 4
	svr.WriteTimeout = time.Minute * 10

	initLogger()
	initSecrets()
	initStatus()

	if err := svr.RunTLS(*cert, *privkey); err != nil {
		log.Fatal(err)
	}
}

func runClient() {
	svr.Handler = http.HandlerFunc(clientHandler)

	initProxy()
	initLogger()
	initSecrets()
	initStatus()
	if *autoproxy {
		initAutoproxy()
	}

	if err := svr.Run(); err != nil {
		log.Fatal(err)
	}
}

func testPort() error {
	port, err := strconv.Atoi(svr.Port)
	if err != nil {
		return err
	}
	l, err := net.ListenTCP("tcp", &net.TCPAddr{Port: port})
	if err != nil {
		return err
	}
	l.Close()
	return nil
}

func run() {
	switch *mode {
	case "client":
		runClient()
	case "server":
		runServer()
	default:
		log.Fatalln("unknow mode:", *mode)
	}
}

func test() error {
	if err := testPort(); err != nil {
		return err
	}

	if *mode == "client" {
		if _, err := url.Parse("https://" + *server); err != nil {
			return err
		}
		if *autoproxy {
			if _, err := getAutoproxy(); err != nil {
				return err
			}
		}
	}

	return nil
}
