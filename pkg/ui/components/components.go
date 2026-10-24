package components

import "embed"

// FS embeds all built-in .kiw reusable and atomic UI components provided by Forge.
//
//go:embed *.kiw
var FS embed.FS
