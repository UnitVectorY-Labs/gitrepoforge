package output

import (
	"encoding/json/v2"
	"io"
	"os"
	"testing"
)

func TestPrintJSONEmptyRepos(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = write
	err = NewReport("test", "validate", "", "").PrintJSON()
	os.Stdout = old
	write.Close()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(read)
	read.Close()
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Repos []any `json:"repos"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Repos == nil || len(report.Repos) != 0 {
		t.Fatalf("repos = %#v, want empty array", report.Repos)
	}
}
