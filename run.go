package main

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

func runServer() {
	svr.Handler = http.HandlerFunc(serverHandler)
	svr.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))

	initLogger()
	initSecrets()

	if err := svr.RunTLS(*cert, *privkey); err != nil {
		log.Fatal(err)
	}
}

func runClient() {
	svr.Handler = http.HandlerFunc(clientHandler)

	initProxy()
	initLogger()
	initSecrets()
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
