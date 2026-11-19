package ssg

import (
	"bytes"
	"path"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// DefaultIslandsAssetPath is the logical asset path where the client island loader
// is served and written.
const DefaultIslandsAssetPath = "assets/kiw-islands.js"

// IslandsConfig configures client island hydration for modular frontend frameworks
// (Svelte, Vue, React, etc.).
type IslandsConfig struct {
	// Enabled forces the client island loader script to be injected even on pages
	// with no static framework mount markers (e.g. for dynamic client-side islands).
	Enabled bool `yaml:"enabled"`
	// LoaderURL specifies a custom loader script URL, overriding DefaultIslandsAssetPath.
	LoaderURL string `yaml:"loader_url"`
}

// Islands configures client island hydration on the site.
func (s *Site) Islands(config IslandsConfig) *Site {
	s.islands = config
	return s
}

// hasFrameworkMounts reports whether any page or layout contains framework mount markers.
func (s *Site) hasFrameworkMounts() bool {
	if s.islands.Enabled || s.hasIslandsEmitted {
		return true
	}
	for _, p := range s.pages {
		if l, ok := s.layouts[p.Layout]; ok && strings.Contains(l.Body, "data-kiw-framework") {
			return true
		}
		if c, ok := s.comps[p.Root]; ok && strings.Contains(c.Body, "data-kiw-framework") {
			return true
		}
	}
	return false
}

// injectIslandsLoader appends the client island loader module script before </body>.
func (s *Site) injectIslandsLoader(doc string) string {
	loaderURL := s.islands.LoaderURL
	if loaderURL == "" {
		loaderURL = s.asset(DefaultIslandsAssetPath)
	}

	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return doc
	}
	bodyNode := findElement(root, "body")
	if bodyNode == nil {
		return doc
	}
	referenced := referencedAssets(root)

	// Inject kiw-islands.js if not already present
	if !referenced[loaderURL] && !referenced[path.Base(loaderURL)] {
		bodyNode.AppendChild(&html.Node{
			Type:     html.ElementNode,
			Data:     "script",
			DataAtom: atom.Script,
			Attr: []html.Attribute{
				{Key: "type", Val: "module"},
				{Key: "src", Val: loaderURL},
			},
		})
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, root); err != nil {
		return doc
	}
	return buf.String()
}

// DefaultIslandsScript is the vanilla ES-module island hydration runtime script
// supporting Svelte 5/4, Vue 3, and custom framework adapters with progressive
// hydration schedules (load, idle, visible, media).
const DefaultIslandsScript = `/**
 * KiwIslands - Client Island Hydration Runtime for Krewire
 * Supports progressive hydration (load, idle, visible, media) for modular frontend frameworks (Svelte, Vue, custom).
 */

const ATTR_MOUNT = 'data-kiw-mount';
const ATTR_FRAMEWORK = 'data-kiw-framework';
const ATTR_HYDRATE = 'data-kiw-hydrate';
const ATTR_PROPS = 'data-kiw-props';
const ATTR_HYDRATED = 'data-kiw-hydrated';
const ATTR_HYDRATING = 'data-kiw-hydrating';

const registry = new Map();
const componentRegistry = new Map();
let customResolver = null;
let basePath = '/components';

// Default framework adapters
const adapters = {
  // Svelte adapter (Svelte 5 & 4 compatible)
  svelte: async function (el, Comp, props) {
    const Component = (Comp && Comp.default) ? Comp.default : Comp;
    let svelteModule = (typeof window !== 'undefined' && window.svelte) ? window.svelte : null;
    if (!svelteModule) {
      try {
        svelteModule = await import(/* @vite-ignore */ 'svelte');
      } catch (_) {}
    }

    if (svelteModule && typeof svelteModule.hydrate === 'function' && el.childNodes.length > 0) {
      return svelteModule.hydrate(Component, { target: el, props: props });
    } else if (svelteModule && typeof svelteModule.mount === 'function') {
      el.innerHTML = '';
      return svelteModule.mount(Component, { target: el, props: props });
    } else if (typeof Component === 'function' && Component.prototype && Component.prototype.$set) {
      return new Component({
        target: el,
        props: props,
        hydrate: el.childNodes.length > 0,
      });
    } else if (typeof Component === 'function') {
      try {
        return Component(el, props);
      } catch (_) {
        el.innerHTML = '';
        return new Component({ target: el, props: props });
      }
    }
    throw new Error('No compatible Svelte runtime found to mount component');
  },

  // Vue adapter (Vue 3 compatible)
  vue: async function (el, Comp, props) {
    const Component = (Comp && Comp.default) ? Comp.default : Comp;
    let vueModule = (typeof window !== 'undefined' && window.Vue) ? window.Vue : null;
    if (!vueModule) {
      try {
        vueModule = await import(/* @vite-ignore */ 'vue');
      } catch (_) {}
    }

    if (vueModule) {
      const create = (vueModule.createSSRApp && el.childNodes.length > 0)
        ? vueModule.createSSRApp
        : vueModule.createApp;
      if (create) {
        const app = create(Component, props);
        return app.mount(el);
      }
    }
    throw new Error('Vue 3 runtime (createSSRApp/createApp) not found');
  }
};

// Safe props parser
function parseProps(el) {
  const raw = el.getAttribute(ATTR_PROPS);
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch (e) {
    console.warn('[kiw-islands] Malformed props JSON on ' + el.getAttribute(ATTR_MOUNT), e);
    return {};
  }
}

// Component resolver
async function resolveComponent(name, framework, el) {
  if (componentRegistry.has(name)) {
    return componentRegistry.get(name);
  }
  if (typeof customResolver === 'function') {
    const res = await customResolver(name, framework, el);
    if (res) return res;
  }
  if (typeof window !== 'undefined') {
    if (window.KiwComponents && window.KiwComponents[name]) {
      return window.KiwComponents[name];
    }
    if (window[name]) {
      return window[name];
    }
  }
  const ext = framework === 'vue' ? '.vue' : (framework === 'svelte' ? '.svelte' : '.js');
  const path = basePath.replace(/\/+$/, '') + '/' + name + ext;
  try {
    const mod = await import(/* @vite-ignore */ path);
    return mod.default || mod[name] || mod;
  } catch (err) {
    try {
      const jsPath = basePath.replace(/\/+$/, '') + '/' + name + '.js';
      const mod = await import(/* @vite-ignore */ jsPath);
      return mod.default || mod[name] || mod;
    } catch (_) {
      throw new Error('[kiw-islands] Unable to resolve component "' + name + '" (' + framework + '): ' + err.message);
    }
  }
}

// Hydrate a single element
async function hydrateElement(el) {
  if (el.hasAttribute(ATTR_HYDRATED) || el.hasAttribute(ATTR_HYDRATING)) {
    return;
  }
  el.setAttribute(ATTR_HYDRATING, 'true');

  const name = el.getAttribute(ATTR_MOUNT);
  const framework = (el.getAttribute(ATTR_FRAMEWORK) || '').toLowerCase();
  const props = parseProps(el);

  try {
    el.dispatchEvent(new CustomEvent('kiw:island:hydrate', {
      bubbles: true,
      detail: { name, framework, props }
    }));

    const adapter = registry.get(framework) || adapters[framework];
    if (!adapter) {
      if (!framework || framework === 'wasm' || framework === 'go') {
        el.removeAttribute(ATTR_HYDRATING);
        return;
      }
      throw new Error('Unsupported island framework: "' + framework + '"');
    }

    const Component = await resolveComponent(name, framework, el);
    const instance = await adapter(el, Component, props);

    el.setAttribute(ATTR_HYDRATED, 'true');
    el.removeAttribute(ATTR_HYDRATING);

    el.dispatchEvent(new CustomEvent('kiw:island:mounted', {
      bubbles: true,
      detail: { name, framework, props, instance }
    }));
  } catch (err) {
    el.removeAttribute(ATTR_HYDRATING);
    console.error('[kiw-islands] Failed to hydrate island "' + name + '":', err);
    el.dispatchEvent(new CustomEvent('kiw:island:error', {
      bubbles: true,
      detail: { name, framework, error: err }
    }));
  }
}

// Progressive hydration scheduler
function scheduleHydration(el) {
  if (el.hasAttribute(ATTR_HYDRATED) || el.hasAttribute(ATTR_HYDRATING)) {
    return;
  }
  const fw = (el.getAttribute(ATTR_FRAMEWORK) || '').toLowerCase();
  if (fw === 'wasm' || fw === 'go') {
    return;
  }

  const strategy = (el.getAttribute(ATTR_HYDRATE) || 'load').toLowerCase();

  switch (strategy) {
    case 'idle':
      if ('requestIdleCallback' in window) {
        window.requestIdleCallback(() => hydrateElement(el), { timeout: 2000 });
      } else {
        setTimeout(() => hydrateElement(el), 200);
      }
      break;

    case 'visible':
      if ('IntersectionObserver' in window) {
        const observer = new IntersectionObserver((entries) => {
          for (const entry of entries) {
            if (entry.isIntersecting) {
              observer.disconnect();
              hydrateElement(el);
              break;
            }
          }
        }, { rootMargin: '50px' });
        observer.observe(el);
      } else {
        hydrateElement(el);
      }
      break;

    case 'media': {
      const query = el.getAttribute('data-kiw-media');
      if (query && 'matchMedia' in window) {
        const mql = window.matchMedia(query);
        if (mql.matches) {
          hydrateElement(el);
        } else {
          const handler = (e) => {
            if (e.matches) {
              mql.removeEventListener('change', handler);
              hydrateElement(el);
            }
          };
          mql.addEventListener('change', handler);
        }
      } else {
        hydrateElement(el);
      }
      break;
    }

    case 'load':
    default:
      hydrateElement(el);
      break;
  }
}

// Scan root for islands
function scan(root) {
  const doc = root || (typeof document !== 'undefined' ? document : null);
  if (!doc) return;
  const elements = doc.querySelectorAll('[' + ATTR_MOUNT + ']:not([' + ATTR_HYDRATED + '])');
  for (let i = 0; i < elements.length; i++) {
    scheduleHydration(elements[i]);
  }
}

// Initialize
function init(root) {
  if (typeof document === 'undefined') return;
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => scan(root), { once: true });
  } else {
    scan(root);
  }
}

const KiwIslands = {
  register: function (framework, adapter) {
    registry.set(framework.toLowerCase(), adapter);
  },
  component: function (name, comp) {
    componentRegistry.set(name, comp);
  },
  components: function (map) {
    for (const k in map) {
      if (Object.prototype.hasOwnProperty.call(map, k)) {
        componentRegistry.set(k, map[k]);
      }
    }
  },
  resolve: function (fn) {
    customResolver = fn;
  },
  config: function (opts) {
    if (opts && opts.basePath) basePath = opts.basePath;
  },
  hydrate: hydrateElement,
  scan: scan,
  init: init
};

if (typeof window !== 'undefined') {
  window.KiwIslands = KiwIslands;
  init();
}

export default KiwIslands;
export const register = KiwIslands.register;
export const component = KiwIslands.component;
export const components = KiwIslands.components;
export const resolve = KiwIslands.resolve;
export const config = KiwIslands.config;
export const hydrate = KiwIslands.hydrate;
export const scan = KiwIslands.scan;
`
