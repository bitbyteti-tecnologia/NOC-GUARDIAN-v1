// util.go
// - Funções utilitárias compartilhadas (muxVar, getenv).
// - Evita duplicação e compilações quebradas.

package main

import (
    "net/http"
    "os"

    "github.com/gorilla/mux"
)

func muxVar(r *http.Request, key string) string {
    vars := mux.Vars(r)
    return vars[key]
}

func getenv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}
