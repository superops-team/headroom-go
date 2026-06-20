package headroom

import "testing"

func TestContentKindValues(t *testing.T) {
	if int(KindText) != 0 {
		t.Errorf("KindText != 0")
	}
	if int(KindJSON) != 1 {
		t.Errorf("KindJSON != 1")
	}
	if int(KindCode) != 2 {
		t.Errorf("KindCode != 2")
	}
	if KindText.String() != "Text" {
		t.Errorf("KindText.String() got %s", KindText.String())
	}
	if KindJSON.String() != "JSON" {
		t.Errorf("KindJSON.String() got %s", KindJSON.String())
	}
	if KindCode.String() != "Code" {
		t.Errorf("KindCode.String() got %s", KindCode.String())
	}
	cases := map[ContentKind]string{KindDiff: "Diff", KindLog: "Log", KindSearch: "Search", KindTabular: "Tabular", KindSpreadsheet: "Spreadsheet", KindHTML: "HTML", KindUnknown: "Unknown"}
	for kind, want := range cases {
		if kind.String() != want {
			t.Errorf("%v.String() got %s want %s", int(kind), kind.String(), want)
		}
	}
	if ContentKind(999).String() != "Text" {
		t.Errorf("unknown numeric kind should keep legacy Text fallback, got %s", ContentKind(999).String())
	}
}
