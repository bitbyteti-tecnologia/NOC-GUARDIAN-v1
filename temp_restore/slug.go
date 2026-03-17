// slug.go
// - Gera um slug seguro para nomes de DB do Postgres.
// - Mantém apenas [a-z0-9_], converte espaços para underscore e remove múltiplos underscores.
// - Também limita tamanho do slug para ajudar a não exceder o limite de identificador do Postgres.

package main

import (
    "regexp"
    "strings"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9_]+`)
var multiUnd = regexp.MustCompile(`_+`)

func SlugifyDBName(s string) string {
    s = strings.ToLower(strings.TrimSpace(s))
    s = strings.ReplaceAll(s, " ", "_")
    s = nonAlnum.ReplaceAllString(s, "_")
    s = multiUnd.ReplaceAllString(s, "_")
    s = strings.Trim(s, "_")
    if s == "" {
        s = "cliente"
    }
    // Limite de tamanho do slug (o DB final tem prefixo + sufixo)
    if len(s) > 32 {
        s = s[:32]
    }
    return s
}
