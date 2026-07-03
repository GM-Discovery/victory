package characters

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

func randomShortID() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return strings.ToLower(hex.EncodeToString(buf[:]))
	}
	return "fallback"
}

func customStableID(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "custom"
	}
	return strings.ToUpper(prefix) + "_" + randomShortID()
}
