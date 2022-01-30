package main

import (
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/sunshineplan/utils/cache"
	"github.com/sunshineplan/utils/executor"
	"github.com/sunshineplan/utils/txt"
)

var ac = cache.New(false)

type autoproxyList struct {
	domain  []string
	full    []string
	regexp  []*regexp.Regexp
	keyword []string
}

func match(u *url.URL) bool {
	i, _ := ac.Get("autoproxy")
	list, ok := i.(autoproxyList)
	if !ok {
		log.Print("failed to load autoproxy")
		return true
	}
	hostname := u.Hostname()
	for _, i := range list.domain {
		if strings.HasSuffix(hostname, i) && (len(hostname) == len(i) || hostname[len(hostname)-len(i)-1] == '.') {
			if *debug {
				log.Printf("[Debug] %s match domain:%s", hostname, i)
			}
			return true
		}
	}
	for _, i := range list.full {
		domain := strings.ReplaceAll(strings.ReplaceAll(i, "http://", ""), "https://", "")
		if domain == hostname {
			if *debug {
				log.Printf("[Debug] %s match full:%s", hostname, domain)
			}
			return true
		}
	}
	for _, i := range list.regexp {
		if i.MatchString(hostname) {
			if *debug {
				log.Printf("[Debug] %s match regexp:%s", hostname, i)
			}
			return true
		}
	}
	for _, i := range list.keyword {
		if strings.Contains(u.String(), i) {
			if *debug {
				log.Printf("[Debug] %s match keyword:%s", u, i)
			}
			return true
		}
	}
	if *debug {
		log.Println("[Debug] no match result:", u)
	}
	return false
}

func getAutoproxy() (interface{}, error) {
	var r io.ReadCloser
	if *autoproxyURL == "" {
		body, err := executor.ExecuteConcurrentArg(
			[]string{
				"https://raw.githubusercontent.com/sunshineplan/autoproxy/release/autoproxy.txt",
				"https://cdn.jsdelivr.net/gh/sunshineplan/autoproxy@release/autoproxy.txt",
			},
			func(url interface{}) (interface{}, error) {
				resp, err := http.Get(url.(string))
				if err != nil {
					return nil, err
				}
				return resp.Body, nil
			},
		)
		if err != nil {
			return nil, err
		}
		r = body.(io.ReadCloser)
	} else {
		resp, err := http.Get(*autoproxyURL)
		if err != nil {
			return nil, err
		}
		r = resp.Body
	}
	defer r.Close()

	rows, err := txt.ReadAll(base64.NewDecoder(base64.StdEncoding, r))
	if err != nil {
		return nil, err
	}

	var list autoproxyList
	for _, i := range rows {
		i = strings.TrimSpace(i)
		if i == "" || i[:1] == "[" || i[:1] == "!" {
			continue
		}
		switch {
		case strings.HasPrefix(i, "||"):
			list.domain = append(list.domain, strings.Replace(i, "||", "", 1))
		case strings.HasPrefix(i, "|"):
			list.full = append(list.full, strings.Replace(i, "|", "", 1))
		case regexp.MustCompile("/.+/").MatchString(i):
			re, err := regexp.Compile(i[1 : len(i)-1])
			if err != nil {
				log.Println("bad regular expression", i)
			} else {
				list.regexp = append(list.regexp, re)
			}
		default:
			list.keyword = append(list.keyword, i)
		}
	}
	return list, nil
}

func initAutoproxy() {
	list, err := getAutoproxy()
	if err != nil {
		log.Fatalln("failed to get autoproxy:", err)
	}
	ac.Set("autoproxy", list, 24*time.Hour, getAutoproxy)
	if *debug {
		log.Print("Autoproxy initialized")
	}
}
