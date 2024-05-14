package main

import (
	"log"
	"os"

	flag "github.com/spf13/pflag"
)

// net/http/httputil ReverseProxy, DumpRequest, DumpResponse
// net/http FileServer

type Config struct {
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
	flag.Parse()
	config.Proxy = parseProxy(proxy)
	args := flag.Args()
	if len(args) > 0 {
		log.Printf("unrecognised arguments: %v\n", args)
		os.Exit(1)
	}

	log.Printf("Listening on: %v:%v (verbose %v)\n", config.Host, config.Port, config.Verbose)
	for _, p := range config.Proxy {
		log.Printf("Proxy: %v => %v\n", p.Path, p.To)
	}

}
