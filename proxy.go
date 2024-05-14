package main

import (
	"log"
	"net/url"
	"os"
	"strings"
)

type Proxy struct {
	Path string
	To   *url.URL
}

func parseProxy(proxy []string) (res []Proxy) {
	for _, spec := range proxy {
		parts := strings.Fields(spec)
		if len(parts)%2 != 0 {
			log.Printf("wrong number of items in --proxy flag: '%v' (expecting pairs, found %d)\n", spec, len(parts))
			os.Exit(1)
		}
		for i := 0; i < len(parts); i += 2 {
			path := parts[i]
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			if path == "/" {
				log.Printf("invalid path in --proxy flag: '%v' (a path is required e.g. /foo)\n", path)
				os.Exit(1)
			}
			to_url := parts[i+1]
			if to_url == "" { // edge-case
				log.Printf("invalid URL in --proxy flag: proxy-to url is missing\n")
				os.Exit(1)
			}
			u, err := url.Parse(to_url)
			if err != nil {
				log.Printf("invalid URL in --proxy flag: '%v' (%v)\n", to_url, err)
				os.Exit(1)
			}
			if u.Scheme == "" || u.Host == "" || u.Path == "" || u.Path == "/" {
				log.Printf("invalid URL in --proxy flag: '%v' (scheme, host and path are required e.g. http://localhost/foo)\n", to_url)
				os.Exit(1)
			}
			p := Proxy{Path: path, To: u}
			res = append(res, p)
		}
	}
	return
}
