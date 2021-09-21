package assets

import (
	"embed"
)

//go:embed *.yaml rbac/*.yaml storageclass/*.yaml
var f embed.FS

// Asset reads and returns the content of the named file.
func ReadFile(name string) ([]byte, error) {
	return f.ReadFile(name)
}
