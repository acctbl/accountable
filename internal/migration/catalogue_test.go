package migration

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestCommittedCatalogueMatchesFilenameContract(t *testing.T) {
	t.Parallel()

	if _, err := mergeSources(Catalogue()); err != nil {
		t.Fatalf("committed migration catalogue: %v", err)
	}
}

func TestMergeSourcesRejectsBadNamesAndDuplicateVersions(t *testing.T) {
	t.Parallel()

	valid := []byte("-- +goose Up\nSELECT 1;\n")
	for name, sources := range map[string][]Source{
		"bad filename": {{Owner: "bad", FS: fs.FS(fstest.MapFS{"00001_bad.sql": &fstest.MapFile{Data: valid}})}},
		"duplicate version": {
			{Owner: "first", FS: fs.FS(fstest.MapFS{"20260801000000_first_init.sql": &fstest.MapFile{Data: valid}})},
			{Owner: "second", FS: fs.FS(fstest.MapFS{"20260801000000_second_init.sql": &fstest.MapFile{Data: valid}})},
		},
	} {
		if _, err := mergeSources(sources); err == nil {
			t.Errorf("%s: mergeSources accepted invalid catalogue", name)
		}
	}
}
