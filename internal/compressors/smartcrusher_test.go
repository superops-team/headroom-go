package compressors

import (
	"encoding/json"
	"strconv"
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

// 标准模式：低信号短数组不应粗暴折叠为字符串
func TestSmartCrusher_Standard_ArrayLowSignalPassthrough(t *testing.T) {
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
	items, ok := result["items"].([]interface{})
	if !ok || len(items) != 10 {
		t.Errorf("low-signal array should pass through as array, got %T %v", result["items"], result["items"])
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

func TestSmartCrusher_ArrayAnalysisPlanExecute(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"items":[`)
	for i := 0; i < 200; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		status := "ok"
		message := "noise"
		extra := `,"empty":""`
		if i == 13 {
			status = "ERROR"
			message = "critical failure"
		}
		if i == 31 {
			extra = `,"empty":"","extra":"outlier"`
		}
		sb.WriteString(`{"id":"item-` + strconv.Itoa(i) + `","status":"` + status + `","message":"` + message + `"` + extra + `}`)
	}
	sb.WriteString(`]}`)
	input := sb.String()
	out, steps, err := SmartCrushJSONWithSteps(input, SmartCrushConfig{Aggressiveness: 0.6})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("output must be valid JSON: %s", out)
	}
	if !strings.Contains(out, "_headroom_array") || !strings.Contains(out, "critical failure") || !strings.Contains(out, "outliers") {
		t.Fatalf("expected summarized array with critical/outlier retention, got %s", out)
	}
	if len(steps) == 0 || steps[len(steps)-1].Skipped {
		t.Fatalf("expected executed smartcrusher step, got %#v", steps)
	}
	for _, step := range steps {
		if step.Name == "smartcrusher_array" && (step.TokensBefore != 0 || step.TokensAfter != 0) {
			t.Fatalf("smartcrusher array step should not report byte/item counts as tokens: %#v", step)
		}
	}
}

func TestSmartCrusher_PrimitiveArrayKeepsCriticalMiddleValue(t *testing.T) {
	items := make([]string, 0, 60)
	for i := 0; i < 60; i++ {
		value := "normal-event-payload-with-low-signal"
		if i == 30 {
			value = "ERROR critical middle event"
		}
		items = append(items, strconv.Quote(value))
	}
	input := `{"items":[` + strings.Join(items, ",") + `]}`
	out, _, err := SmartCrushJSONWithSteps(input, SmartCrushConfig{Aggressiveness: 0.6})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ERROR critical middle event") || !strings.Contains(out, `"critical"`) {
		t.Fatalf("expected primitive critical value to be retained, got %s", out)
	}
}

func TestSmartCrusher_FieldProjectionPrioritizesCriticalFields(t *testing.T) {
	stats := map[string]FieldStat{
		"alpha":     {Count: 10, NonZero: 10},
		"bravo":     {Count: 10, NonZero: 10},
		"charlie":   {Count: 10, NonZero: 10},
		"delta":     {Count: 10, NonZero: 10},
		"echo":      {Count: 10, NonZero: 10},
		"foxtrot":   {Count: 10, NonZero: 10},
		"golf":      {Count: 10, NonZero: 10},
		"hotel":     {Count: 10, NonZero: 10},
		"status":    {Count: 10, NonZero: 10, Critical: true},
		"message":   {Count: 10, NonZero: 10, Critical: true},
		"timestamp": {Count: 10, NonZero: 10},
	}
	fields := chooseKeepFields(stats)
	joined := "," + strings.Join(fields, ",") + ","
	for _, field := range []string{"status", "message", "timestamp"} {
		if !strings.Contains(joined, ","+field+",") {
			t.Fatalf("critical field %q not retained in %v", field, fields)
		}
	}
	if len(fields) != 8 {
		t.Fatalf("expected field cap of 8, got %d: %v", len(fields), fields)
	}
}

func TestSmartCrusher_InsufficientSavingsFallback(t *testing.T) {
	input := `{"items":[{"id":"a"},{"id":"b"},{"id":"c"},{"id":"d"},{"id":"e"},{"id":"f"}]}`
	out, steps, err := SmartCrushJSONWithSteps(input, SmartCrushConfig{Aggressiveness: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"items":[{"id":"a"},{"id":"b"},{"id":"c"},{"id":"d"},{"id":"e"},{"id":"f"}]}` {
		t.Fatalf("expected insufficient-savings passthrough, got %s", out)
	}
	found := false
	for _, step := range steps {
		if step.Skipped && strings.Contains(step.Reason, "insufficient savings") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected insufficient savings step, got %#v", steps)
	}
}
