package main

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
)

type CacheTag int

const (
	TagFile CacheTag = iota
	TagIndex
	TagDirList
	Tag404
)

// type cacheEntry struct {
// 	tag CacheTag
// 	to  string
// }

// Inspired by the code in net/http/fs.go:
// ServeHTTP:929 calls serveFile
// ServeFile:755 calls serveFile
// serveFile:628 calls serveContent
// ServeContent:194 calls serveContent
// serveContent:223 sends the response

type jamPuppyHandler struct {
	root     http.FileSystem
	index    string
	appindex string
	proxy    []Proxy
	verbose  bool
}

//reverseProxy httputil.ReverseProxy
// knownFiles map[string]cacheEntry

// Proxy from /{Path}/* to `{To}/*`
type Proxy struct {
	Path string   // MUST start with '/' (doesn't need to end with '/')
	To   *url.URL // MUST have scheme, host and path
}

func JamPuppyServer(config Config) http.Handler {
	// prefixes := []string{}
	// for _, p := range proxy {
	// 	prefixes = append(prefixes, p.Path)
	// }
	// var jp *jamPuppyHandler
	return &jamPuppyHandler{
		root:     http.Dir(config.Dir),
		index:    config.Index,
		appindex: config.AppIndex,
		proxy:    config.Proxy,
		// reverseProxy: httputil.ReverseProxy{Director: func(r *http.Request) {
		// 	jp.rewriteProxyURL(r)
		// }},
		verbose: config.Verbose,
		// knownFiles: map[string]cacheEntry{},
	}
}

// func (jp *jamPuppyHandler) rewriteProxyURL(r *http.Request) {
// 	// based on rewriteRequestURL:269 in net/http/httputil/reverseproxy.go
// 	target, _ := url.Parse("http://locahost/foo")
// 	targetQuery := target.RawQuery
// 	r.URL.Scheme = target.Scheme
// 	r.URL.Host = target.Host
// 	r.URL.Path, r.URL.RawPath = joinURLPath(target, r.URL)
// 	if targetQuery == "" || r.URL.RawQuery == "" {
// 		r.URL.RawQuery = targetQuery + r.URL.RawQuery
// 	} else {
// 		r.URL.RawQuery = targetQuery + "&" + r.URL.RawQuery
// 	}
// }

func (jp *jamPuppyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// support .html masking
	// i.e. urls without any extension are mapped to
	// the same name with '.html' extension on disk.

	// canonical path (from ServeHTTP:929 net/http/fs.go)
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}
	upath = path.Clean(upath)
	if containsDotDot(upath) {
		http.Error(w, "invalid URL path", http.StatusBadRequest) // cache: forever
		return
	}

	for _, p := range jp.proxy {
		if strings.HasPrefix(upath, p.Path) {
			// cretae a reverse-proxy to serve the request
			proxy := httputil.NewSingleHostReverseProxy(p.To)
			proxy.ServeHTTP(w, r)
			return
		}
	}

	// check if we have a cached classification
	// if entry, found := f.knownFiles[upath]; found {
	// 	switch entry.tag {
	// 	case TagFile:
	// 		r.URL.Path = entry.to
	// 		// this API supplies its own seek-based size function,
	// 		// even though we have Stat().Size() available sad face.
	// 		// http.ServeContent(w, r, upath, modtime, content)
	// 	}
	// 	return
	// }

	jp.serveFileModified(w, r, upath)
}

// based on serveFile:628 in net/http/fs.go with changes
func (jp *jamPuppyHandler) serveFileModified(w http.ResponseWriter, r *http.Request, name string) {

	// redirect .../index.html to .../
	if strings.HasSuffix(r.URL.Path, jp.index) {
		localRedirect(w, r, "./")
		return
	}

	var f http.File
	var err error
	f, err = jp.root.Open(name)
	if err != nil {
		if f = jp.choose404file(name); f == nil {
			// report the original error above
			msg, code := toHTTPError(err)
			http.Error(w, msg, code)
			return
		}
	}
	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		msg, code := toHTTPError(err)
		http.Error(w, msg, code)
		return
	}

	// redirect to canonical path: / at end of directory url
	// r.URL.Path always begins with /
	url := r.URL.Path
	if d.IsDir() {
		if url[len(url)-1] != '/' {
			localRedirect(w, r, path.Base(url)+"/")
			return
		}

		// use contents of index.html for directory, if present
		index := strings.TrimSuffix(name, "/") + jp.index
		ff, err := jp.root.Open(index)
		if err == nil {
			defer ff.Close()
			dd, err := ff.Stat()
			if err == nil {
				d = dd
				f = ff
			}
		}
	} else {
		if url[len(url)-1] == '/' {
			localRedirect(w, r, "../"+path.Base(url))
			return
		}
	}

	// Still a directory? (we didn't find an index file)
	if d.IsDir() {
		// use http.ServeFile to reply with a directory listing,
		// because that involves a lot of private code and is
		// only used for local development anyway.
		http.ServeFile(w, r, name)
		return
	}

	// ServeContent checks if-* conditions, detects content-type,
	// sets content-length and serves up the response.
	http.ServeContent(w, r, d.Name(), d.ModTime(), f)
}

func (jp *jamPuppyHandler) choose404file(name string) http.File {
	// check if no dot after the last slash
	// note: name always starts with '/' so -1 < 0 if no '.' is found
	if strings.LastIndexByte(name, 46 /* dot */) < strings.LastIndexByte(name, 47 /* slash */) {
		// html masking: try the name with '.html' on the end
		f, err := jp.root.Open(name + ".html")
		if err == nil {
			return f
		}
	}
	if jp.appindex != "" {
		// AppIndex: serve up the index page instead.
		f, err := jp.root.Open(jp.appindex)
		if err == nil {
			return f
		}
	}
	return nil
}

// The following are copied unmodified from net/http/fs.go
// in support of the code in serveFileModified above.

func toHTTPError(err error) (msg string, httpStatus int) {
	if errors.Is(err, fs.ErrNotExist) {
		return "404 page not found", http.StatusNotFound
	}
	if errors.Is(err, fs.ErrPermission) {
		return "403 Forbidden", http.StatusForbidden
	}
	// Default:
	return "500 Internal Server Error", http.StatusInternalServerError
}

func localRedirect(w http.ResponseWriter, r *http.Request, newPath string) {
	if q := r.URL.RawQuery; q != "" {
		newPath += "?" + q
	}
	w.Header().Set("Location", newPath)
	w.WriteHeader(http.StatusMovedPermanently)
}

func containsDotDot(v string) bool {
	if !strings.Contains(v, "..") {
		return false
	}
	for _, ent := range strings.FieldsFunc(v, isSlashRune) {
		if ent == ".." {
			return true
		}
	}
	return false
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }

// as usual, these are private helpers we needed to copy to implement
// a proxy like NewSingleHostReverseProxy with customization.

// func joinURLPath(a, b *url.URL) (path, rawpath string) {
// 	if a.RawPath == "" && b.RawPath == "" {
// 		return singleJoiningSlash(a.Path, b.Path), ""
// 	}
// 	// Same as singleJoiningSlash, but uses EscapedPath to determine
// 	// whether a slash should be added
// 	apath := a.EscapedPath()
// 	bpath := b.EscapedPath()

// 	aslash := strings.HasSuffix(apath, "/")
// 	bslash := strings.HasPrefix(bpath, "/")

// 	switch {
// 	case aslash && bslash:
// 		return a.Path + b.Path[1:], apath + bpath[1:]
// 	case !aslash && !bslash:
// 		return a.Path + "/" + b.Path, apath + "/" + bpath
// 	}
// 	return a.Path + b.Path, apath + bpath
// }

// func singleJoiningSlash(a, b string) string {
// 	aslash := strings.HasSuffix(a, "/")
// 	bslash := strings.HasPrefix(b, "/")
// 	switch {
// 	case aslash && bslash:
// 		return a + b[1:]
// 	case !aslash && !bslash:
// 		return a + "/" + b
// 	}
// 	return a + b
// }
