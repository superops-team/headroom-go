package headroom

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type SmartCrushConfig struct {
	Aggressiveness float64
	Observer       Observer
}

type Crushability string

const (
	CrushabilityLow    Crushability = "low"
	CrushabilityMedium Crushability = "medium"
	CrushabilityHigh   Crushability = "high"
)

type DetectedPattern string

const (
	DetectedPatternLowSignal          DetectedPattern = "low_signal"
	DetectedPatternPrimitiveSequence  DetectedPattern = "primitive_sequence"
	DetectedPatternHomogeneousObjects DetectedPattern = "homogeneous_objects"
	DetectedPatternMixedObjects       DetectedPattern = "mixed_objects"
)

type RecommendedStrategy string

const (
	RecommendedStrategyPassthrough         RecommendedStrategy = "passthrough"
	RecommendedStrategySummarizePrimitives RecommendedStrategy = "summarize_primitives"
	RecommendedStrategySummarizeObjects    RecommendedStrategy = "summarize_objects"
)

type FieldStat struct {
	Count    int  `json:"count"`
	NonZero  int  `json:"non_zero"`
	Critical bool `json:"critical,omitempty"`
}

type ArrayAnalysis struct {
	Length              int                  `json:"length"`
	ObjectCount         int                  `json:"object_count"`
	PrimitiveCount      int                  `json:"primitive_count"`
	FieldStats          map[string]FieldStat `json:"field_stats,omitempty"`
	CriticalIndexes     []int                `json:"critical_indexes,omitempty"`
	ErrorIndexes        []int                `json:"error_indexes,omitempty"`
	OutlierIndexes      []int                `json:"outlier_indexes,omitempty"`
	Crushability        Crushability         `json:"crushability"`
	DetectedPattern     DetectedPattern      `json:"detected_pattern"`
	RecommendedStrategy RecommendedStrategy  `json:"recommended_strategy"`
}

type CompressionPlan struct {
	Analysis        ArrayAnalysis       `json:"analysis"`
	Strategy        RecommendedStrategy `json:"strategy"`
	KeepFields      []string            `json:"keep_fields,omitempty"`
	KeepIndexes     []int               `json:"keep_indexes,omitempty"`
	MinSavingsRatio float64             `json:"min_savings_ratio"`
	Reason          string              `json:"reason,omitempty"`
}

// SmartCrushJSON 压缩 JSON，按三级策略处理。
// 非法 JSON → 返回原文 + nil（降级策略）。
// 输出始终是合法 JSON。
func SmartCrushJSON(content string, cfg SmartCrushConfig) (string, error) {
	out, _, err := SmartCrushJSONWithSteps(content, cfg)
	return out, err
}

func SmartCrushJSONWithSteps(content string, cfg SmartCrushConfig) (string, []CompressionStep, error) {
	trimmed := strings.TrimSpace(content)
	if !json.Valid([]byte(trimmed)) {
		return content, []CompressionStep{{Name: "smartcrusher", Kind: KindJSON.String(), Skipped: true, Reason: "invalid json passthrough"}}, nil
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return content, []CompressionStep{{Name: "smartcrusher", Kind: KindJSON.String(), Skipped: true, Reason: "invalid json passthrough"}}, nil
	}

	steps := make([]CompressionStep, 0, 4)
	result := compress(parsed, cfg, &steps)

	out, err := json.Marshal(result)
	if err != nil {
		return content, steps, nil
	}
	if len(steps) == 0 {
		steps = append(steps, CompressionStep{Name: "smartcrusher", Kind: KindJSON.String(), Skipped: true, Reason: "no array compression opportunity"})
	}
	for _, step := range steps {
		if cfg.Observer != nil {
			cfg.Observer.ObserveCompressionStep(step)
		}
	}
	return string(out), steps, nil
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

func compress(v interface{}, cfg SmartCrushConfig, steps *[]CompressionStep) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		return compressObject(x, cfg, steps)
	case []interface{}:
		return compressArray(x, cfg, steps)
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

func compressObject(obj map[string]interface{}, cfg SmartCrushConfig, steps *[]CompressionStep) interface{} {
	out := make(map[string]interface{}, len(obj))
	for k, v := range obj {
		// 保守模式：字段级删除零值（null/false/0/空串/空对象/空数组）
		if cfg.Aggressiveness < 0.7 && isZeroValue(v) {
			continue
		}
		compressed := compress(v, cfg, steps)
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

func compressArray(arr []interface{}, cfg SmartCrushConfig, steps *[]CompressionStep) interface{} {
	plan := BuildCompressionPlan(AnalyzeArray(arr), cfg)
	if plan.Strategy == RecommendedStrategyPassthrough {
		appendSmartStep(steps, true, plan.Reason)
		return passthroughArray(arr, cfg, steps)
	}

	var candidate interface{}
	switch plan.Strategy {
	case RecommendedStrategySummarizeObjects:
		candidate = executeObjectArrayPlan(arr, plan, cfg, steps)
	case RecommendedStrategySummarizePrimitives:
		candidate = executePrimitiveArrayPlan(arr, plan, cfg, steps)
	default:
		return passthroughArray(arr, cfg, steps)
	}

	originalBytes, _ := json.Marshal(arr)
	candidateBytes, _ := json.Marshal(candidate)
	minSaving := int(float64(len(originalBytes)) * plan.MinSavingsRatio)
	if len(candidateBytes) == 0 || len(originalBytes)-len(candidateBytes) < minSaving {
		appendSmartStep(steps, true, "insufficient savings; passthrough")
		return passthroughArray(arr, cfg, steps)
	}
	appendSmartStep(steps, false, string(plan.Strategy))
	return candidate
}

func AnalyzeArray(arr []interface{}) ArrayAnalysis {
	analysis := ArrayAnalysis{Length: len(arr), FieldStats: make(map[string]FieldStat), Crushability: CrushabilityLow, DetectedPattern: DetectedPatternLowSignal, RecommendedStrategy: RecommendedStrategyPassthrough}
	keySetCounts := make(map[int]int)
	keyCountsByIndex := make([]int, len(arr))
	for i, item := range arr {
		obj, ok := item.(map[string]interface{})
		if !ok {
			analysis.PrimitiveCount++
			if isCriticalValue(item) {
				analysis.CriticalIndexes = appendUniqueInt(analysis.CriticalIndexes, i)
			}
			continue
		}
		analysis.ObjectCount++
		keyCountsByIndex[i] = len(obj)
		keySetCounts[len(obj)]++
		rowCritical := false
		rowError := false
		for k, v := range obj {
			stat := analysis.FieldStats[k]
			stat.Count++
			if !isZeroValue(v) {
				stat.NonZero++
			}
			if isCriticalField(k) || isCriticalValue(v) {
				stat.Critical = true
			}
			if isCriticalValue(v) {
				rowCritical = true
			}
			if isErrorFieldOrValue(k, v) {
				rowError = true
			}
			analysis.FieldStats[k] = stat
		}
		if rowCritical {
			analysis.CriticalIndexes = appendUniqueInt(analysis.CriticalIndexes, i)
		}
		if rowError {
			analysis.ErrorIndexes = appendUniqueInt(analysis.ErrorIndexes, i)
		}
	}
	commonKeyCount := mostCommonKeyCount(keySetCounts)
	for i, count := range keyCountsByIndex {
		if count > 0 && commonKeyCount > 0 && count != commonKeyCount {
			analysis.OutlierIndexes = appendUniqueInt(analysis.OutlierIndexes, i)
		}
	}
	if analysis.ObjectCount == analysis.Length && analysis.Length >= 6 {
		analysis.DetectedPattern = DetectedPatternHomogeneousObjects
		if len(analysis.OutlierIndexes) > 0 {
			analysis.DetectedPattern = DetectedPatternMixedObjects
		}
		analysis.Crushability = CrushabilityHigh
		analysis.RecommendedStrategy = RecommendedStrategySummarizeObjects
	} else if analysis.PrimitiveCount == analysis.Length && analysis.Length >= 20 {
		analysis.DetectedPattern = DetectedPatternPrimitiveSequence
		analysis.Crushability = CrushabilityMedium
		analysis.RecommendedStrategy = RecommendedStrategySummarizePrimitives
	}
	return analysis
}

func BuildCompressionPlan(analysis ArrayAnalysis, cfg SmartCrushConfig) CompressionPlan {
	plan := CompressionPlan{Analysis: analysis, Strategy: analysis.RecommendedStrategy, MinSavingsRatio: 0.08, Reason: "analysis selected strategy"}
	if cfg.Aggressiveness < 0.3 || analysis.Length <= 5 || analysis.Crushability == CrushabilityLow {
		plan.Strategy = RecommendedStrategyPassthrough
		plan.Reason = "low signal or conservative mode"
		return plan
	}
	plan.KeepIndexes = append(plan.KeepIndexes, 0)
	if analysis.Length > 1 {
		plan.KeepIndexes = appendUniqueInt(plan.KeepIndexes, analysis.Length-1)
	}
	plan.KeepIndexes = appendLimitedIndexes(plan.KeepIndexes, analysis.CriticalIndexes, 8)
	plan.KeepIndexes = appendLimitedIndexes(plan.KeepIndexes, analysis.ErrorIndexes, 8)
	plan.KeepIndexes = appendLimitedIndexes(plan.KeepIndexes, analysis.OutlierIndexes, 8)
	sort.Ints(plan.KeepIndexes)
	if plan.Strategy == RecommendedStrategySummarizeObjects {
		plan.KeepFields = chooseKeepFields(analysis.FieldStats)
	}
	return plan
}

func executeObjectArrayPlan(arr []interface{}, plan CompressionPlan, cfg SmartCrushConfig, steps *[]CompressionStep) interface{} {
	keptRows := make([]interface{}, 0, len(plan.KeepIndexes))
	criticalRows := make([]interface{}, 0)
	errorRows := make([]interface{}, 0)
	outlierRows := make([]interface{}, 0)
	for _, idx := range plan.KeepIndexes {
		if idx >= 0 && idx < len(arr) {
			keptRows = append(keptRows, projectArrayItem(arr[idx], plan.KeepFields, cfg, steps))
		}
	}
	for _, idx := range plan.Analysis.CriticalIndexes {
		criticalRows = append(criticalRows, projectArrayItem(arr[idx], plan.KeepFields, cfg, steps))
	}
	for _, idx := range plan.Analysis.ErrorIndexes {
		errorRows = append(errorRows, projectArrayItem(arr[idx], plan.KeepFields, cfg, steps))
	}
	for _, idx := range plan.Analysis.OutlierIndexes {
		outlierRows = append(outlierRows, compress(arr[idx], cfg, steps))
	}
	fields := make(map[string]interface{}, len(plan.Analysis.FieldStats))
	for k, stat := range plan.Analysis.FieldStats {
		fields[k] = map[string]interface{}{"count": stat.Count, "non_zero": stat.NonZero, "critical": stat.Critical}
	}
	summary := map[string]interface{}{
		"_headroom_array": true,
		"count":           plan.Analysis.Length,
		"pattern":         string(plan.Analysis.DetectedPattern),
		"fields":          fields,
		"sample":          keptRows,
		"omitted":         plan.Analysis.Length - len(plan.KeepIndexes),
	}
	if len(criticalRows) > 0 {
		summary["critical"] = criticalRows
	}
	if len(errorRows) > 0 {
		summary["errors"] = errorRows
	}
	if len(outlierRows) > 0 {
		summary["outliers"] = outlierRows
	}
	return summary
}

func executePrimitiveArrayPlan(arr []interface{}, plan CompressionPlan, cfg SmartCrushConfig, steps *[]CompressionStep) interface{} {
	first := make([]interface{}, 0, 3)
	last := make([]interface{}, 0, 3)
	critical := make([]interface{}, 0)
	for i := 0; i < len(arr) && i < 3; i++ {
		first = append(first, compress(arr[i], cfg, steps))
	}
	start := len(arr) - 3
	if start < 0 {
		start = 0
	}
	for i := start; i < len(arr); i++ {
		last = append(last, compress(arr[i], cfg, steps))
	}
	for _, idx := range plan.Analysis.CriticalIndexes {
		if idx < 0 || idx >= len(arr) {
			continue
		}
		if idx < 3 || idx >= start {
			continue
		}
		critical = append(critical, compress(arr[idx], cfg, steps))
	}
	summary := map[string]interface{}{"_headroom_array": true, "count": plan.Analysis.Length, "pattern": string(plan.Analysis.DetectedPattern), "first": first, "last": last, "omitted": len(arr) - len(first) - len(last) - len(critical)}
	if len(critical) > 0 {
		summary["critical"] = critical
	}
	return summary
}

func passthroughArray(arr []interface{}, cfg SmartCrushConfig, steps *[]CompressionStep) []interface{} {
	out := make([]interface{}, 0, len(arr))
	for _, item := range arr {
		out = append(out, compress(item, cfg, steps))
	}
	return out
}

func projectArrayItem(item interface{}, fields []string, cfg SmartCrushConfig, steps *[]CompressionStep) interface{} {
	obj, ok := item.(map[string]interface{})
	if !ok {
		return compress(item, cfg, steps)
	}
	out := make(map[string]interface{}, len(fields))
	for _, field := range fields {
		if v, ok := obj[field]; ok {
			out[field] = compress(v, cfg, steps)
		}
	}
	if len(out) == 0 {
		return compress(item, cfg, steps)
	}
	return out
}

func chooseKeepFields(stats map[string]FieldStat) []string {
	critical := make([]string, 0, len(stats))
	regular := make([]string, 0, len(stats))
	for k, stat := range stats {
		if stat.Critical || isCriticalField(k) {
			critical = append(critical, k)
			continue
		}
		if stat.NonZero == stat.Count {
			regular = append(regular, k)
		}
	}
	sort.Slice(critical, func(i, j int) bool {
		leftRank := criticalFieldRank(critical[i])
		rightRank := criticalFieldRank(critical[j])
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return critical[i] < critical[j]
	})
	sort.Strings(regular)
	fields := append(critical, regular...)
	if len(fields) > 8 {
		fields = fields[:8]
	}
	return fields
}

func criticalFieldRank(field string) int {
	lower := strings.ToLower(field)
	for rank, needle := range []string{"id", "key", "name", "type", "status", "severity", "level", "error", "message", "code", "time", "timestamp"} {
		if lower == needle || strings.Contains(lower, needle) {
			return rank
		}
	}
	return 100
}

func isCriticalField(field string) bool {
	lower := strings.ToLower(field)
	for _, needle := range []string{"id", "key", "name", "type", "status", "severity", "level", "error", "message", "code", "time", "timestamp"} {
		if lower == needle || strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func isCriticalValue(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	upper := strings.ToUpper(s)
	return strings.Contains(upper, "CRITICAL") || strings.Contains(upper, "FATAL") || strings.Contains(upper, "ERROR") || strings.Contains(upper, "FAIL")
}

func isErrorFieldOrValue(field string, v interface{}) bool {
	lower := strings.ToLower(field)
	return strings.Contains(lower, "error") || strings.Contains(lower, "exception") || isCriticalValue(v)
}

func appendUniqueInt(values []int, value int) []int {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func appendLimitedIndexes(dst []int, src []int, limit int) []int {
	for _, idx := range src {
		if len(dst) >= limit {
			return dst
		}
		dst = appendUniqueInt(dst, idx)
	}
	return dst
}

func mostCommonKeyCount(counts map[int]int) int {
	bestKey := 0
	bestCount := 0
	for key, count := range counts {
		if count > bestCount {
			bestKey = key
			bestCount = count
		}
	}
	return bestKey
}

func appendSmartStep(steps *[]CompressionStep, skipped bool, reason string) {
	if steps == nil {
		return
	}
	*steps = append(*steps, CompressionStep{Name: "smartcrusher_array", Kind: KindJSON.String(), Skipped: skipped, Reason: reason})
}
