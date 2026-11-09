// Package i18n provides the same offline catalog to native controls and the web UI.
package i18n

import (
	"embed"
	"encoding/json"
)

//go:embed catalog.json
var assets embed.FS
var Catalog, _ = assets.ReadFile("catalog.json")
var messages = func() map[string]map[string]string {
	var out map[string]map[string]string
	if err := json.Unmarshal(Catalog, &out); err != nil {
		panic(err)
	}
	return out
}()

func Text(language, source string) string {
	if value, ok := messages[language][source]; ok {
		return value
	}
	return source
}
