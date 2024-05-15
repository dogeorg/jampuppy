package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	flag "github.com/spf13/pflag"
)

// net/http/httputil ReverseProxy, DumpRequest, DumpResponse
// net/http FileServer

type Config struct {
	Dir      string
	Port     int
	Host     string
	Index    string
	AppIndex string
	Proxy    []Proxy
	Verbose  bool
}

func main() {
	config := Config{
		Dir:   ".",
		Port:  8080,
		Host:  "localhost",
		Index: "index.html",
	}
	proxy := []string{}
	flag.IntVarP(&config.Port, "port", "p", config.Port, "Listen port")
	flag.StringVarP(&config.Host, "host", "h", config.Host, "Bind to network interface")
	flag.StringArrayVarP(&proxy, "proxy", "r", proxy, "Reverse Proxy: '/thing http://localhost:8085/thing' (one or more)")
	flag.BoolVarP(&config.Verbose, "verbose", "v", config.Verbose, "Colour prints ingoing/outgoing requests for debugging")
	flag.StringVarP(&config.Dir, "dir", "d", config.Dir, "Directory of static files to serve")
	flag.StringVarP(&config.Index, "index", "I", config.Index, "Index file to serve for directores")
	flag.StringVarP(&config.AppIndex, "app-index", "A", config.AppIndex, "Index file to serve in place of 404 (for SPA)")
	flag.Parse()
	config.Proxy = parseProxy(proxy)
	if !strings.HasPrefix(config.Dir, string(filepath.Separator)) {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		config.Dir = path.Join(cwd, config.Dir)
	}
	args := flag.Args()
	if len(args) > 0 {
		log.Printf("unexpected arguments: %v\n", args[1:])
		os.Exit(1)
	}

	log.Printf("Listening on: %v:%v (verbose %v)\n", config.Host, config.Port, config.Verbose)
	log.Printf("Serving: %v\n", config.Dir)
	for _, p := range config.Proxy {
		log.Printf("Proxy: %v => %v\n", p.Path, p.To)
	}

	handler := JamPuppyServer(config)
	http.ListenAndServe(fmt.Sprintf("%s:%d", config.Host, config.Port), handler)
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
