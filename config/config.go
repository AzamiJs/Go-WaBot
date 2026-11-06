package config

import (
    "os"
    "strings"
)

func readStringList(name string, fallback ...string) []string {
    if raw := strings.TrimSpace(os.Getenv(name)); raw != "" {
        values := strings.FieldsFunc(raw, func(r rune) bool {
            return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
        })
        cleaned := make([]string, 0, len(values))
        for _, value := range values {
            value = strings.TrimSpace(value)
            if value != "" {
                cleaned = append(cleaned, value)
            }
        }
        if len(cleaned) > 0 {
            return cleaned
        }
    }
    return fallback
}

var Owners = readStringList("BOT_OWNERS", "5213332329453", "5213123497515")
var Devs = readStringList("BOT_DEVS", "5213332329453", "5213123497515")
var Prefix = readStringList("BOT_PREFIX", "#", ".", "!")
