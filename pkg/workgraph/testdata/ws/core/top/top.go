package top

import (
	"embed"

	"example.com/core/mid"
)

//go:embed assets
var Assets embed.FS

var _ = mid.Name

// Name is a fixture constant.
const Name = "top"
