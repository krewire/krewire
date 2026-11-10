package commands

// Hints maps names developers reach for from other tools to the kiw command
// that does that job. `kiw <name>` prints the hint instead of a generic
// "unknown command" suggestion.
var Hints = map[string]string{
	"start":     "kiw dev  (live dev server with reload) or kiw run  (run once)",
	"watch":     "kiw dev  (rebuilds and reloads on change)",
	"preview":   "kiw serve  (serves the production build; run kiw build first)",
	"exec":      "kiw run <file.go|task>",
	"create":    "kiw new <project>  (new directory) or kiw init  (current directory)",
	"install":   "kiw add <package> or kiw add ui/<component>",
	"i":         "kiw add <package>",
	"generate":  "kiw make <kind> <name>",
	"g":         "kiw make <kind> <name>",
	"component": "kiw add ui/<name>  (copy a kui component) or kiw make component <Name>  (blank)",
	"lint":      "kiw vet  (go vet, krewire.yaml schema, .kiw templates)",
	"check":     "kiw vet or kiw config",
	"upgrade":   "kiw update  (project dependencies) or kiw self-update  (the kiw CLI)",
	"admin":     "kiw forge panel <name>  (scaffold admin panel dashboard)",
	"crud":      "kiw forge resource <name>  (scaffold CRUD resource and forms)",
	"form":      "kiw forge form <name>  (scaffold declarative form)",
	"workspace": "kiw dash  (AI-native virtual workspace & sandboxes) or kiw ws  (go.work multi-repo)",
	"dashboard": "kiw dash  (local dev dashboard, services, logs, sandboxes)",
}
