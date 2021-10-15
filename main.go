package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunshineplan/service"
	"github.com/sunshineplan/utils/httpsvr"
	"github.com/vharitonsky/iniflags"
)

// common
var (
	mode      = flag.String("mode", "client", "server or client mode")
	psk       = flag.String("psk", "", "pre-shared key")
	accesslog = flag.String("access-log", "", "Path to access log file")
	errorlog  = flag.String("error-log", "", "Path to error log file")
	debug     = flag.Bool("debug", false, "debug")
)

// server
var (
	secrets = flag.String("secrets", "", "Path to secrets file for Basic Authentication")
	cert    = flag.String("cert", "", "Path to certificate file")
	privkey = flag.String("privkey", "", "Path to private key file")
)

// client
var (
	server   = flag.String("server", "", "Server address")
	username = flag.String("username", "", "Username")
	password = flag.String("password", "", "Password")
)

var self string
var svr = httpsvr.New()

var svc = service.Service{
	Name:     "MyProxy",
	Desc:     "My Proxy",
	Exec:     run,
	TestExec: test,
	Options: service.Options{
		Dependencies: []string{"After=network.target"},
	},
}

func init() {
	var err error
	self, err = os.Executable()
	if err != nil {
		log.Fatalln("Failed to get self path:", err)
	}
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), `Usage of %s:
  --host <string>
    	Listening host
  --port <number>
    	Listening port
  --secrets <file>
    	Path to secrets file for Basic Authentication
  --cert <file>
    	Path to certificate file
  --privkey <file>
    	Path to private key file
  --access-log <file>
    	Path to access log file
  --error-log <file>
    	Path to error log file
  --update <url>
    	Update URL
%s`, os.Args[0], service.Usage)
}

func main() {
	flag.Usage = usage
	flag.StringVar(&svr.Host, "host", "", "Listening host")
	flag.StringVar(&svr.Port, "port", "", "Listening port")
	flag.StringVar(&svc.Options.UpdateURL, "update", "", "Update URL")
	iniflags.SetConfigFile(filepath.Join(filepath.Dir(self), "config.ini"))
	iniflags.SetAllowMissingConfigFile(true)
	iniflags.SetAllowUnknownFlags(true)
	iniflags.Parse()

	*mode = strings.ToLower(*mode)

	if *psk == "" {
		log.Fatal("pre-shared key can not be empty")
	}

	if *secrets == "" {
		if info, err := os.Stat(filepath.Join(filepath.Dir(self), "secrets")); err == nil && !info.IsDir() {
			*secrets = filepath.Join(filepath.Dir(self), "secrets")
		}
	}

	if service.IsWindowsService() {
		svc.Run(false)
		return
	}

	var err error
	switch flag.NArg() {
	case 0:
		run()
	case 1:
		switch flag.Arg(0) {
		case "run":
			svc.Run(false)
		case "debug":
			svc.Run(true)
		case "test":
			err = svc.Test()
		case "install":
			err = svc.Install()
		case "uninstall", "remove":
			err = svc.Uninstall()
		case "start":
			err = svc.Start()
		case "stop":
			err = svc.Stop()
		case "restart":
			err = svc.Restart()
		case "update":
			err = svc.Update()
		default:
			log.Fatalln(fmt.Sprintf("Unknown argument: %s", flag.Arg(0)))
		}
	default:
		log.Fatalln(fmt.Sprintf("Unknown arguments: %s", strings.Join(flag.Args(), " ")))
	}
	if err != nil {
		log.Fatalf("Failed to %s: %v", flag.Arg(0), err)
	}
}
