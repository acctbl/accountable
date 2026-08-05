package migration

import (
	"fmt"
	"io/fs"
	"regexp"
	"testing/fstest"

	"github.com/acctbl/accountable/internal/platform/database"
)

var migrationFilenamePattern = regexp.MustCompile(`^(\d{14})_[a-z0-9]+_[a-z0-9_]+\.sql$`)

// Source is one owner's ordered Goose SQL tree.
type Source struct {
	Owner string
	FS    fs.FS
}

// Catalogue returns migration owners in apply order.
func Catalogue() []Source {
	return []Source{
		{Owner: "platform/database", FS: database.Migrations()},
	}
}

func mergeSources(sources []Source) (fs.FS, error) {
	files := make(fstest.MapFS)
	owners := make(map[string]string)
	versions := make(map[string]string)
	for _, source := range sources {
		entries, err := fs.ReadDir(source.FS, ".")
		if err != nil {
			return nil, fmt.Errorf("migration owner %s: %w", source.Owner, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			matches := migrationFilenamePattern.FindStringSubmatch(name)
			if len(matches) != 2 {
				return nil, fmt.Errorf("migration owner %s file %s does not match the timestamped module filename contract", source.Owner, name)
			}
			if other, exists := versions[matches[1]]; exists {
				return nil, fmt.Errorf("migration version %s is used by both %s and %s", matches[1], other, source.Owner+"/"+name)
			}
			if other, exists := owners[name]; exists {
				return nil, fmt.Errorf("migration %s owned by both %s and %s", name, other, source.Owner)
			}
			payload, err := fs.ReadFile(source.FS, name)
			if err != nil {
				return nil, fmt.Errorf("migration owner %s file %s: %w", source.Owner, name, err)
			}
			files[name] = &fstest.MapFile{Data: payload}
			owners[name] = source.Owner
			versions[matches[1]] = source.Owner + "/" + name
		}
	}
	return files, nil
}
