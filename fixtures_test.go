package headroom

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixturesLoad(t *testing.T) {
	paths := []string{"json/sample.json", "diff/sample.diff", "log/sample.log", "search/sample.txt", "html/sample.html", "tags/sample.txt"}
	for _, p := range paths {
		data, err := ioutil.ReadFile(filepath.Join("testdata", p))
		if err != nil {
			t.Fatalf("fixture %s: %v", p, err)
		}
		if len(data) == 0 {
			t.Fatalf("fixture %s is empty", p)
		}
	}
}

func TestGoldenFixturesStableKeyAssertions(t *testing.T) {
	opts := DefaultOptions()
	opts.EnablePipeline = true
	opts.Reversible = false
	cases := []struct {
		name     string
		path     string
		contains []string
	}{
		{name: "json", path: "json/sample.json", contains: []string{"items", "status", "ok"}},
		{name: "diff", path: "diff/sample.diff", contains: []string{"diff --git", "+new"}},
		{name: "log", path: "log/sample.log", contains: []string{"ERROR"}},
		{name: "search", path: "search/sample.txt", contains: []string{"a.go", "return nil"}},
		{name: "html", path: "html/sample.html", contains: []string{"<article>", "Hello", "</article>"}},
		{name: "tag", path: "tags/sample.txt", contains: []string{"<system-reminder>", "</system-reminder>"}},
	}
	for _, tc := range cases {
		data, err := ioutil.ReadFile(filepath.Join("testdata", tc.path))
		if err != nil {
			t.Fatalf("fixture %s: %v", tc.path, err)
		}
		res, err := Compress([]Message{{Role: "user", Content: string(data)}}, opts)
		if err != nil {
			t.Fatalf("%s compress: %v", tc.name, err)
		}
		if len(res.Messages) != 1 {
			t.Fatalf("%s messages len=%d", tc.name, len(res.Messages))
		}
		out := res.Messages[0].Content
		if out == "" {
			t.Fatalf("%s empty output", tc.name)
		}
		for _, want := range tc.contains {
			if !strings.Contains(out, want) {
				t.Fatalf("%s output missing %q: %s", tc.name, want, out)
			}
		}
	}
}
