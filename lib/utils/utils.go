package utils

import (
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
)

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

var (
	lidCacheMu sync.RWMutex
	lidCache   = make(map[string]cacheEntry)
)

func setLidCache(key, value string) {
	lidCacheMu.Lock()
	defer lidCacheMu.Unlock()
	lidCache[key] = cacheEntry{value: value, expiresAt: time.Now().Add(30 * time.Minute)}
}

func getLidCache(key string) (string, bool) {
	lidCacheMu.RLock()
	defer lidCacheMu.RUnlock()
	e, ok := lidCache[key]
	if !ok || time.Now().After(e.expiresAt) {
		return "", false
	}
	return e.value, true
}

func normalizeToJid(phone string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)
	if cleaned == "" {
		return ""
	}
	return cleaned + "@s.whatsapp.net"
}

func ResolveLidToRealJid(lid string, meta *types.GroupInfo) string {
	input := strings.TrimSpace(lid)
	if input == "" || strings.HasSuffix(input, "@s.whatsapp.net") {
		return input
	}
	if cached, ok := getLidCache(input); ok {
		return cached
	}
	if meta == nil {
		return input
	}
	lidPrefix := strings.Split(strings.Split(input, ":")[0], "@")[0]
	for _, p := range meta.Participants {
		pId := p.JID.String()
		pPrefix := strings.Split(strings.Split(pId, ":")[0], "@")[0]
		if pPrefix == lidPrefix {
			phoneStr := p.PhoneNumber.User
			if phoneStr != "" {
				realJid := normalizeToJid(phoneStr)
				setLidCache(input, realJid)
				return realJid
			}
		}
	}
	return input
}
