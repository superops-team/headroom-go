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
}
