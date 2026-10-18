// Package migrations embarque les fichiers SQL versionnés, appliqués dans l'ordre alphabétique.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
