package headroom

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SmartCrushConfig struct {
	Aggressiveness float64
}

// SmartCrushJSON 压缩 JSON，按三级策略处理。
// 非法 JSON → 返回原文 + nil（降级策略）。
// 输出始终是合法 JSON。
func SmartCrushJSON(content string, cfg SmartCrushConfig) (string, error) {
	trimmed := strings.TrimSpace(content)
	if !json.Valid([]byte(trimmed)) {
		return content, nil
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return content, nil
	}

	result := compress(parsed, cfg)

	out, err := json.Marshal(result)
	if err != nil {
		return content, nil
	}
	return string(out), nil
}

// isZeroValue 判断一个值是否为"零值"（null/false/0/空串/空对象/空数组）
// 仅用于对象字段的删除判定。数组元素不删除。
func isZeroValue(v interface{}) bool {
	switch x := v.(type) {
	case nil:
		return true
	case bool:
		return !x
	case float64:
		return x == 0
	case string:
		return x == ""
	case map[string]interface{}:
		return len(x) == 0
	case []interface{}:
		return len(x) == 0
	default:
		return false
	}
}

func compress(v interface{}, cfg SmartCrushConfig) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		return compressObject(x, cfg)
	case []interface{}:
		return compressArray(x, cfg)
	case float64:
		if cfg.Aggressiveness >= 0.7 {
			if x == float64(int64(x)) {
				return fmt.Sprintf("%d", int64(x))
			}
			return fmt.Sprintf("%.2f", x)
		}
		return x
	case bool:
		if cfg.Aggressiveness >= 0.7 {
			if x {
				return "T"
			}
			return "F"
		}
		return x
	default:
		return v
	}
}

func compressObject(obj map[string]interface{}, cfg SmartCrushConfig) interface{} {
	out := make(map[string]interface{}, len(obj))
	for k, v := range obj {
		// 保守模式：字段级删除零值（null/false/0/空串/空对象/空数组）
		if cfg.Aggressiveness < 0.7 && isZeroValue(v) {
			continue
		}
		compressed := compress(v, cfg)
		// 激进模式下：对象内的零值字段（可能被压缩后仍然是零）保留
		// 但仍需要清理压缩后的空对象
		if cfg.Aggressiveness >= 0.7 {
			if sub, ok := compressed.(map[string]interface{}); ok && len(sub) == 0 {
				continue
			}
		}
		out[k] = compressed
	}
	return out
}

func compressArray(arr []interface{}, cfg SmartCrushConfig) interface{} {
	// 标准模式及以上：>5 元素折叠为 "[...N items...]"
	if cfg.Aggressiveness >= 0.3 && len(arr) > 5 {
		return fmt.Sprintf("[...%d items...]", len(arr))
	}
	// 否则：递归压缩每个元素；数组中的 0/false 保留，不删除
	out := make([]interface{}, 0, len(arr))
	for _, item := range arr {
		out = append(out, compress(item, cfg))
	}
	return out
}
