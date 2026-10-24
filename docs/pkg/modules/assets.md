# `assets`

Import: `github.com/krewire/krewire/pkg/assets`

## Purpose

Static asset management, bundling, minification, fingerprinting (content hashing), and cache-busting for web applications and static sites.

## Main API

- `assets.Pipeline`: Configurable asset compilation pipeline.
- `assets.Fingerprint(path, content)`: Appends SHA-256 hash digests to asset paths.
- `assets.Manifest`: Maps original file paths to their fingerprinted filenames.
- `assets.MinifyCSS(css)`: Fast pure-Go CSS minifier.
- `assets.MinifyJS(js)`: Basic JavaScript minification.

## Example

```go
package main

import (
	"fmt"
	"github.com/krewire/krewire/pkg/assets"
)

func main() {
	pipe := assets.NewPipeline(assets.WithMinify(true), assets.WithFingerprint(true))
	manifest, err := pipe.ProcessDir("public/assets", "dist/assets")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Fingerprinted CSS: %s\n", manifest["app.css"])
}
```

## Design boundary

Independent asset transformation pipeline. Standard library I/O only, no Node.js dependency required.
