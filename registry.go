package mmmigrate

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var registry []Migration

// Called from migration file's init(). Gets name from file name
func Register(up, down func(tx *sql.Tx) error) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("migrate.Register: could not determine filename")
	}

	name := parseName(file)

	for _, m := range registry {
		if m.Name == name {
			panic(fmt.Sprintf("migrate: duplicate migration name %s (from %s)", name, file))
		}
	}
	registry = append(registry, Migration{
		Name: name,
		Up:   up,
		Down: down,
	})
}

func parseName(file string) string {
	base := strings.TrimSuffix(filepath.Base(file), ".go")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 || len(parts[0]) != 14 {
		panic(fmt.Sprintf("migrate: filename %q invalid - must match <14 digit timestamp>_<description>", base))
	}
	return base
}

func All() []Migration {
	sorted := make([]Migration, len(registry))
	copy(sorted, registry)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}
