module github.com/krewire/krewire/web

go 1.27.3

require (
	github.com/chai2010/webp v1.4.0
	github.com/disintegration/imaging v1.6.2
	github.com/tdewolff/minify/v2 v2.24.19
	golang.org/x/net v0.62.0
	github.com/krewire/ui/core v0.0.0
	github.com/krewire/ui/kiw v0.0.0
	github.com/krewire/ui/node v0.0.0
)

replace (
	github.com/krewire/ui/core => ../../ui/core
	github.com/krewire/ui/kiw => ../../ui/kiw
	github.com/krewire/ui/node => ../../ui/node
)
