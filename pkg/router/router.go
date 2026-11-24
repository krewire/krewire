package router

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Router registers routes and dispatches requests.
type Router struct {
	routes  []*Route
	statics []staticRoute
	// NotFound, when set, handles unmatched routes.
	NotFound HandlerFunc
	mws      []Middleware

	base   string
	parent *Router
	chain  []Middleware
	named  map[string][]segment

	namePrefix string
	where      map[string]string

	built http.Handler
	dirty bool
}

type staticRoute struct {
	prefix string
	dir    string
	fsys   fs.FS
}

// New returns an empty Router.
func New() *Router {
	return &Router{}
}

// NewRouter returns an empty Router.
func NewRouter() *Router {
	return New()
}

// Routes returns all registered routes on the root router.
func (r *Router) Routes() []*Route {
	return r.root().routes
}

// HasRoute reports whether a route with method and pattern is registered.
func (r *Router) HasRoute(method, pattern string) bool {
	resolved := joinPattern(r.base, pattern)
	for _, rt := range r.root().routes {
		if rt.method == method && rt.pattern == resolved {
			return true
		}
	}
	return false
}

// Get registers a handler for GET requests on a path pattern.
func (r *Router) Get(pattern string, h HandlerFunc) {
	r.Handle(http.MethodGet, pattern, h)
}

// Post registers a handler for POST requests on a path pattern.
func (r *Router) Post(pattern string, h HandlerFunc) {
	r.Handle(http.MethodPost, pattern, h)
}

// Put registers a handler for PUT requests on a path pattern.
func (r *Router) Put(pattern string, h HandlerFunc) {
	r.Handle(http.MethodPut, pattern, h)
}

// Delete registers a handler for DELETE requests on a path pattern.
func (r *Router) Delete(pattern string, h HandlerFunc) {
	r.Handle(http.MethodDelete, pattern, h)
}

// Patch registers a handler for PATCH requests on a path pattern.
func (r *Router) Patch(pattern string, h HandlerFunc) {
	r.Handle(http.MethodPatch, pattern, h)
}

// Options registers a handler for OPTIONS requests on a path pattern.
func (r *Router) Options(pattern string, h HandlerFunc) {
	r.Handle(http.MethodOptions, pattern, h)
}

// Head registers a handler for HEAD requests on a path pattern.
func (r *Router) Head(pattern string, h HandlerFunc) {
	r.Handle(http.MethodHead, pattern, h)
}

// Any registers a handler for all common HTTP methods on a path pattern.
func (r *Router) Any(pattern string, h HandlerFunc) {
	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodHead} {
		r.Handle(m, pattern, h)
	}
}

// Match registers a handler for the given HTTP methods on a path pattern.
func (r *Router) Match(methods []string, pattern string, h HandlerFunc) {
	for _, m := range methods {
		if m == "" {
			continue
		}
		r.Handle(strings.ToUpper(strings.TrimSpace(m)), pattern, h)
	}
}

// HandleFunc registers a standard http.HandlerFunc on pattern.
func (r *Router) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) {
	if h == nil {
		return
	}
	pattern = strings.TrimSpace(pattern)
	method := ""
	path := pattern
	if idx := strings.IndexByte(pattern, ' '); idx > 0 {
		method = strings.ToUpper(strings.TrimSpace(pattern[:idx]))
		path = strings.TrimSpace(pattern[idx+1:])
	}
	adapter := func(w http.ResponseWriter, req *http.Request, _ Params) {
		h(w, req)
	}
	if method == http.MethodGet {
		r.Handle(http.MethodGet, path, adapter)
		r.Handle(http.MethodHead, path, adapter)
	} else if method != "" {
		r.Handle(method, path, adapter)
	} else {
		r.Any(path, adapter)
	}
}

// Fallback registers a catch-all handler for unmatched routes.
func (r *Router) Fallback(h HandlerFunc) {
	r.root().NotFound = h
	r.root().dirty = true
}

// Redirect registers a GET route that redirects to `to`.
func (r *Router) Redirect(from, to string, code ...int) {
	c := http.StatusFound
	if len(code) > 0 {
		c = code[0]
	}
	r.Get(from, func(w http.ResponseWriter, req *http.Request, _ Params) {
		http.Redirect(w, req, to, c)
	})
}

// PermanentRedirect registers a GET route that permanently redirects (301) to `to`.
func (r *Router) PermanentRedirect(from, to string) {
	r.Redirect(from, to, http.StatusMovedPermanently)
}

// Prefix is an alias for Group(prefix).
func (r *Router) Prefix(prefix string) *Router { return r.Group(prefix) }

// Name returns a derived router that prefixes all route names registered through it.
func (r *Router) Name(prefix string) *Router {
	return r.namePrefixRouter(prefix)
}

// As is an alias for Name.
func (r *Router) As(prefix string) *Router { return r.Name(prefix) }

func (r *Router) namePrefixRouter(prefix string) *Router {
	np := r.namePrefix + prefix
	whereCopy := cloneWhere(r.where)
	chainCopy := append([]Middleware(nil), r.chain...)
	return &Router{base: r.base, parent: r.root(), chain: chainCopy, namePrefix: np, where: whereCopy}
}

// Middleware returns a derived router that adds middleware to the current chain.
func (r *Router) Middleware(mws ...Middleware) *Router { return r.Group("", mws...) }

// Where adds a param constraint to the group.
func (r *Router) Where(param, constraint string) *Router {
	nw := cloneWhere(r.where)
	if nw == nil {
		nw = map[string]string{}
	}
	nw[param] = constraint
	chainCopy := append([]Middleware(nil), r.chain...)
	return &Router{base: r.base, parent: r.root(), chain: chainCopy, namePrefix: r.namePrefix, where: nw}
}

func cloneWhere(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Handle registers a handler for the given HTTP method and path pattern.
func (r *Router) Handle(method, pattern string, h HandlerFunc) {
	if h == nil {
		h = func(http.ResponseWriter, *http.Request, Params) {}
	}
	h = applyChain(r.chain, h)
	root := r.root()
	resolved := joinPattern(r.base, pattern)
	route := &Route{method: method, pattern: resolved, segments: parsePattern(resolved), handle: h}
	if r.where != nil {
		route.constraints = cloneWhere(r.where)
	}
	root.routes = append(root.routes, route)
	root.dirty = true
}

// MustHandle is Handle that panics on duplicate method+pattern.
func (r *Router) MustHandle(method, pattern string, h HandlerFunc) {
	if r.HasRoute(method, pattern) {
		panic("router: duplicate route " + method + " " + joinPattern(r.base, pattern))
	}
	r.Handle(method, pattern, h)
}

// Mount mounts a sub-router or callback under prefix.
func (r *Router) Mount(prefix string, fn func(*Router), mws ...Middleware) *Router {
	g := r.Group(prefix, mws...)
	if fn != nil {
		fn(g)
	}
	return g
}

func applyChain(chain []Middleware, h HandlerFunc) HandlerFunc {
	for i := len(chain) - 1; i >= 0; i-- {
		mw := chain[i]
		next := h
		h = func(w http.ResponseWriter, req *http.Request, p Params) {
			var inner HandlerFunc = next
			stub := http.HandlerFunc(func(rw http.ResponseWriter, rr *http.Request) {
				inner(rw, rr, p)
			})
			mw(stub).ServeHTTP(w, req)
		}
	}
	return h
}

// Use registers middleware applied to every route in registration order.
func (r *Router) Use(mw ...Middleware) {
	root := r.root()
	root.mws = append(root.mws, mw...)
	root.dirty = true
}

// Group returns a scoped router sharing the parent's middleware.
func (r *Router) Group(prefix string, mws ...Middleware) *Router {
	base := strings.Trim(r.base, "/")
	if p := strings.Trim(prefix, "/"); p != "" {
		base += "/" + p
	}
	chain := make([]Middleware, 0, len(r.chain)+len(mws))
	chain = append(chain, r.chain...)
	chain = append(chain, mws...)
	return &Router{base: "/" + strings.Trim(base, "/"), parent: r.root(), chain: chain, namePrefix: r.namePrefix, where: cloneWhere(r.where)}
}

// Static serves files from dir under the URL prefix.
func (r *Router) Static(prefix, dir string) {
	root := r.root()
	root.statics = append(root.statics, staticRoute{prefix: joinPattern(r.base, prefix), dir: dir})
	root.dirty = true
}

// StaticFS serves files from fsys under the URL prefix.
func (r *Router) StaticFS(prefix string, fsys fs.FS) {
	root := r.root()
	root.statics = append(root.statics, staticRoute{prefix: joinPattern(r.base, prefix), fsys: fsys})
	root.dirty = true
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler().ServeHTTP(w, req)
}

func (r *Router) root() *Router {
	if r.parent != nil {
		return r.parent
	}
	return r
}

func (r *Router) handler() http.Handler {
	if r.built != nil && !r.dirty {
		return r.built
	}
	var h http.Handler = http.HandlerFunc(r.serve)
	for i := len(r.mws) - 1; i >= 0; i-- {
		h = r.mws[i](h)
	}
	r.built = h
	r.dirty = false
	return h
}

func (r *Router) serve(w http.ResponseWriter, req *http.Request) {
	for _, rt := range r.routes {
		if rt.method != req.Method {
			continue
		}
		if params, ok := rt.match(req.URL.Path); ok {
			rt.handle(w, req, params)
			return
		}
	}
	for _, st := range r.statics {
		if !strings.HasPrefix(req.URL.Path, st.prefix) {
			continue
		}
		if st.fsys != nil {
			http.StripPrefix(st.prefix, http.FileServer(http.FS(st.fsys))).ServeHTTP(w, req)
			return
		}
		rel := strings.TrimPrefix(req.URL.Path, st.prefix)
		targetPath := filepath.Join(st.dir, filepath.Clean("/"+rel))

		baseDir, err := filepath.EvalSymlinks(st.dir)
		if err != nil {
			baseDir = filepath.Clean(st.dir)
		}
		realPath, err := filepath.EvalSymlinks(targetPath)
		if err != nil {
			if r.NotFound != nil {
				r.NotFound(w, req, nil)
			} else {
				http.NotFound(w, req)
			}
			return
		}
		relCheck, err := filepath.Rel(baseDir, realPath)
		if err != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(filepath.Separator)) {
			if r.NotFound != nil {
				r.NotFound(w, req, nil)
			} else {
				http.NotFound(w, req)
			}
			return
		}
		fi, err := os.Stat(realPath)
		if err != nil {
			if r.NotFound != nil {
				r.NotFound(w, req, nil)
			} else {
				http.NotFound(w, req)
			}
			return
		}
		if fi.IsDir() {
			indexPath := filepath.Join(realPath, "index.html")
			realIndex, err := filepath.EvalSymlinks(indexPath)
			if err != nil {
				if r.NotFound != nil {
					r.NotFound(w, req, nil)
				} else {
					http.NotFound(w, req)
				}
				return
			}
			relIndexCheck, err := filepath.Rel(baseDir, realIndex)
			if err != nil || relIndexCheck == ".." || strings.HasPrefix(relIndexCheck, ".."+string(filepath.Separator)) {
				if r.NotFound != nil {
					r.NotFound(w, req, nil)
				} else {
					http.NotFound(w, req)
				}
				return
			}
		}
		http.ServeFile(w, req, realPath)
		return
	}
	if r.NotFound != nil {
		r.NotFound(w, req, nil)
		return
	}
	http.NotFound(w, req)
}
