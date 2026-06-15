package headroom

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSmartCrusher_InvalidJSON(t *testing.T) {
	cfg := SmartCrushConfig{Aggressiveness: 0.5}
	out, err := SmartCrushJSON(`{"a": 1, "b": 2,}`, cfg)
	if err != nil {
		t.Errorf("should return nil error for invalid JSON, got %v", err)
	}
	if out != `{"a": 1, "b": 2,}` {
		t.Errorf("invalid JSON should pass through, got %s", out)
	}
}

// 保守模式：移除空对象、空数组、null、false、0
func TestSmartCrusher_Conservative_RemoveZeros(t *testing.T) {
	cfg := SmartCrushConfig{Aggressiveness: 0.2}
	out, err := SmartCrushJSON(`{"x":null,"y":[],"z":{},"b":false,"n":0,"s":"","keep":"yes"}`, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("output must be valid JSON, got: %s", out)
	}
	// 检查是否仍包含 keep=yes
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if _, ok := result["keep"]; !ok {
		t.Errorf("\"keep\" field should be preserved")
	}
	// 所有零值字段应被移除
	for _, field := range []string{"x", "y", "z", "b", "n", "s"} {
		if _, ok := result[field]; ok {
			t.Errorf("zero-value field %q should be removed, still present", field)
		}
	}
}

// 标准模式：数组折叠（>5 元素折叠为 [...N items...]）
func TestSmartCrusher_Standard_ArrayCollapse(t *testing.T) {
	cfg := SmartCrushConfig{Aggressiveness: 0.5}
	out, err := SmartCrushJSON(`{"items":[1,2,3,4,5,6,7,8,9,10]}`, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("output must be valid JSON, got: %s", out)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	items, _ := result["items"]
	if !strings.Contains(items.(string), "...") {
		t.Errorf("items should be collapsed to a string containing '...', got %v", items)
	}
	if !strings.Contains(items.(string), "10") {
		t.Errorf("items collapsed string should contain count '10', got %v", items)
	}
}

// 标准模式：短数组（3 元素）不应折叠
func TestSmartCrusher_Standard_ShortArrayNotCollapsed(t *testing.T) {
	cfg := SmartCrushConfig{Aggressiveness: 0.5}
	out, err := SmartCrushJSON(`{"items":[1,2,3]}`, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("output must be valid JSON, got: %s", out)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	items := result["items"].([]interface{})
	if len(items) != 3 {
		t.Errorf("short array should not be collapsed, got len=%d", len(items))
	}
}

// 激进模式：数字截断为字符串（保留 2 位小数）
func TestSmartCrusher_Aggressive_NumberTruncation(t *testing.T) {
	cfg := SmartCrushConfig{Aggressiveness: 0.8}
	out, err := SmartCrushJSON(`{"ratio":3.1415926535}`, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("output must be valid JSON, got: %s", out)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	ratio, _ := result["ratio"].(string)
	if ratio != "3.14" {
		t.Errorf("aggressive number should be \"3.14\", got %v", ratio)
	}
}

// 激进模式：布尔转 "T" / "F"
func TestSmartCrusher_Aggressive_BooleanString(t *testing.T) {
	cfg := SmartCrushConfig{Aggressiveness: 0.8}
	out, err := SmartCrushJSON(`{"a":true,"b":false}`, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("output must be valid JSON, got: %s", out)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if result["a"] != "T" {
		t.Errorf("true should become \"T\", got %v", result["a"])
	}
	if result["b"] != "F" {
		t.Errorf("false should become \"F\", got %v", result["b"])
	}
}

// 输出必须始终合法 JSON
func TestSmartCrusher_OutputAlwaysValidJSON(t *testing.T) {
	inputs := []string{
		`{"a":1,"b":2}`,
		`{"nested":{"deep":{"value":1.234567}}}`,
		`{"arr":[1,2,3,4,5,6,7,8,9,10,11,12]}`,
		`{"bool":true,"null":null,"zero":0,"emptyStr":""}`,
	}
	for _, in := range inputs {
		for _, agg := range []float64{0.2, 0.5, 0.8} {
			out, err := SmartCrushJSON(in, SmartCrushConfig{Aggressiveness: agg})
			if err != nil {
				t.Fatalf("agg=%.1f err: %v", agg, err)
			}
			if !json.Valid([]byte(out)) {
				t.Errorf("agg=%.1f invalid JSON output: %s\ninput: %s", agg, out, in)
			}
		}
	}
}
