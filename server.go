package main

import (
	"errors"
	"io/fs"
	"log"
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

// Proxy from /{Path}/* to `{To}/*`
type Proxy struct {
	Path string   // MUST start with '/' (doesn't need to end with '/')
	To   *url.URL // MUST have scheme, host and path
}

func JamPuppyServer(config Config) http.Handler {
	return &jamPuppyHandler{
		root:     http.Dir(config.Dir),
		index:    config.Index,
		appindex: config.AppIndex,
		proxy:    config.Proxy,
		verbose:  config.Verbose,
	}
}

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

	log.Printf("[%v] %v\n", r.Method, upath)

	// for development, avoid browser-side caching.
	w.Header().Add("Cache-Control", "private; max-age=0")

	for _, p := range jp.proxy {
		if strings.HasPrefix(upath, p.Path) {
			// strip off the matching prefix to allow remapping.
			// to preserve the prefix, append it to the `To` URL.
			subpath := upath[len(p.Path):]
			if !strings.HasPrefix(subpath, "/") {
				subpath = "/" + subpath
			}
			r.URL.Path = subpath
			result, raw := joinURLPath(p.To, r.URL)
			log.Printf("proxy rewrite: %v -> %v [%v]", upath, result, raw)
			// cretae a reverse-proxy to serve the request
			proxy := httputil.NewSingleHostReverseProxy(p.To)
			proxy.ServeHTTP(w, r)
			return
		}
	}

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

// private helper from reverseproxy.go so we can log the
// proxy destination path (useful for debugging)

func joinURLPath(a, b *url.URL) (path, rawpath string) {
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()
	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")
	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}
