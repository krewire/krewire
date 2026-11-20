# `assets`

Import: `github.com/krewire/krewire/pkg/assets`

## Purpose

Static asset management, bundling, minification, fingerprinting (content hashing), and cache-busting for web applications and static sites.

## Main API

- `assets.NewStore() *Store`: Searchable asset source set. `Mount(fsys)`, `MountDir(dir)`, and `MountEmbed(embedFS)` add sources; `Manifest()` returns name → fingerprinted name; `Handler()` and `HandlerImmutable()` serve the assets over HTTP.
- `(*Store).Fingerprint(name) (string, error)`: Fingerprinted name for one asset.
- `assets.NewBundler(opts ...BundlerOption) *Bundler`: Bundles CSS/JS streams. Options: `WithMinify(bool)`, `WithBanner(string)`, `WithOutputNames(cssPath, jsPath)`, `WithSourceComments(bool)`.
- `(*Bundler).Bundle(store, cssPaths, jsPaths) (*BundleResult, error)` / `WriteToDir(res, outDir) ([]string, error)`: Produce and write the bundle artifacts.
- `(*Bundler).BundleCSS(items)` / `BundleJS(items)`: Bundle pre-read `BundleItem` values.
- `assets.FingerprintName(name, content)`, `assets.ETag(content)`, `assets.ContentType(name)`: Naming, entity tags, and MIME lookup helpers.
- `assets.MinifyCSS(css)` / `assets.MinifyJS(js)`: Pure-Go minifiers.
- `assets.JSON[T](store, name)` / `assets.YAML[T](store, name)`: Typed decode of a mounted asset.
- Constants `assets.DefaultCSSBundlePath` (`assets/krewire.css`) and `assets.DefaultJSBundlePath` (`assets/krewire.js`).

## Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/krewire/krewire/pkg/assets"
)

func main() {
	store := assets.NewStore().MountDir("public")
	bundler := assets.NewBundler(assets.WithMinify(true))

	res, err := bundler.Bundle(store, []string{"app.css"}, []string{"app.js"})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := bundler.WriteToDir(res, ".krewire/build/assets"); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Fingerprinted CSS: %s\n", res.CSSFingerprint)
}
```

## Design boundary

Independent asset transformation pipeline. Standard library I/O only, no Node.js dependency required.
