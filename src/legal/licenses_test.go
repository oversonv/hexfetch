package legal

import (
	"bytes"
	"os"
	"testing"
)

func TestEmbeddedLicensesMatchRepositoryFiles(t *testing.T) {
	tests := []struct {
		name string
		got  string
		path string
	}{
		{name: "project license", got: License, path: "../../LICENSE"},
		{name: "third-party licenses", got: ThirdParty, path: "../../THIRD_PARTY_LICENSES"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal([]byte(test.got), want) {
				t.Fatalf("embedded content does not match %s", test.path)
			}
		})
	}
}
