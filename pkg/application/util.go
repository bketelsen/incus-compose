package application

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/gosimple/slug"
)

func createName(names ...string) string {
	name := slug.Make(strings.Join(names, "-"))
	if len(name) > 64 {
		sha256sum := sha256.Sum256([]byte(name))
		name = hex.EncodeToString(sha256sum[:16])
	}
	return name
}
