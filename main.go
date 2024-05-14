package main

import (
	"log"
	"os"
	"path"
	"strings"

	flag "github.com/spf13/pflag"
)

// net/http/httputil ReverseProxy, DumpRequest, DumpResponse
// net/http FileServer

type Config struct {
	Dir     string
	Port    int
	Host    string
	Proxy   []Proxy
	Verbose bool
}

func main() {
	config := Config{
		Port: 8080,
		Host: "localhost",
	}
	proxy := []string{}
	flag.IntVarP(&config.Port, "port", "p", config.Port, "Listen port")
	flag.StringVarP(&config.Host, "host", "h", config.Host, "Listen interface")
	flag.StringArrayVarP(&proxy, "proxy", "r", proxy, "Reverse Proxy: '/thing http://localhost:8085/thing' (one or more)")
	flag.BoolVarP(&config.Verbose, "verbose", "v", config.Verbose, "Colour prints ingoing/outgoing requests for debugging")
	flag.StringVarP(&config.Dir, "dir", "d", config.Dir, "Directory to serve")
	flag.Parse()
	config.Proxy = parseProxy(proxy)
	if !strings.HasPrefix(config.Dir, "/") {
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

}
