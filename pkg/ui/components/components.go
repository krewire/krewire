package components

import "embed"

// FS embeds all built-in .kiw reusable and atomic UI components provided by Krewire UI.
//
//go:embed *.kiw
var FS embed.FS
