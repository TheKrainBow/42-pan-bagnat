// core/role_rules.go
package core

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"backend/database"
)

/* ================================
   Public API
   ================================ */

// SetRoleRulesJSON validates and stores the canonical (compacted) rules JSON for a role.
func SetRoleRulesJSON(ctx context.Context, roleID string, rulesJSON []byte) error {
	// if roleID <= 0 {
	// 	return fmt.Errorf("invalid roleID")
	// }

	// Basic shape validation: top-level must be a group
	var root map[string]any
	if err := json.Unmarshal(rulesJSON, &root); err != nil {
		return fmt.Errorf("invalid rules JSON: %w", err)
	}
	if kind := strVal(root["kind"]); strings.ToLower(kind) != "group" {
		return fmt.Errorf("rules root must have kind \"group\"")
	}
	// Ensure rules[] exists (empty allowed)
	if _, ok := root["rules"].([]any); !ok {
		if root["rules"] != nil {
			return fmt.Errorf("rules.root.rules must be an array")
		}
		root["rules"] = []any{}
	}

	// Compact/canonicalize before storage
	var buf bytes.Buffer
	if err := json.Compact(&buf, mustMarshal(root)); err != nil {
		return fmt.Errorf("compact: %w", err)
	}

	if err := database.UpdateRoleRulesJSON(ctx, roleID, buf.Bytes()); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// CanonicalizeRoleRulesJSON validates and returns the compacted JSON without persisting.
func CanonicalizeRoleRulesJSON(rulesJSON []byte) ([]byte, error) {
	var root map[string]any
	if err := json.Unmarshal(rulesJSON, &root); err != nil {
		return nil, fmt.Errorf("invalid rules JSON: %w", err)
	}
	if kind := strVal(root["kind"]); strings.ToLower(kind) != "group" {
		return nil, fmt.Errorf("rules root must have kind \"group\"")
	}
	if _, ok := root["rules"].([]any); !ok {
		if root["rules"] != nil {
			return nil, fmt.Errorf("rules.root.rules must be an array")
		}
		root["rules"] = []any{}
	}

	var buf bytes.Buffer
	if err := json.Compact(&buf, mustMarshal(root)); err != nil {
		return nil, fmt.Errorf("compact: %w", err)
	}
	return buf.Bytes(), nil
}

// EvaluateRoleRulesJSON canonicalizes the rule JSON and evaluates it against a payload.
// Returns (matched, canonicalJSON, error).
func EvaluateRoleRulesJSON(rulesJSON []byte, payload any) (bool, []byte, error) {
	matched, can, _, err := EvaluateRoleRulesJSONTrace(rulesJSON, payload)
	return matched, can, err
}

// TraceNode captures per-node evaluation details for debugging/UX.
type TraceNode struct {
	Kind   string `json:"kind"`
	Result bool   `json:"result"`
	// group
	Logic string `json:"logic,omitempty"`
	// scalar
	Path   string `json:"path,omitempty"`
	Op     string `json:"op,omitempty"`
	Value  any    `json:"value,omitempty"`
	Value2 any    `json:"value2,omitempty"`
	Actual any    `json:"actual,omitempty"`
	// array
	Quantifier string `json:"quantifier,omitempty"`
	Size       *int   `json:"size,omitempty"`
	Matches    *int   `json:"matches,omitempty"`
	MatchedIdx []int  `json:"matched_indices,omitempty"`
	Index      *int   `json:"index,omitempty"`
	// message
	Message  string      `json:"message,omitempty"`
	Children []TraceNode `json:"children,omitempty"`
}

// EvaluateRoleRulesJSONTrace canonicalizes and evaluates, returning a debug trace.
func EvaluateRoleRulesJSONTrace(rulesJSON []byte, payload any) (bool, []byte, TraceNode, error) {
	can, err := CanonicalizeRoleRulesJSON(rulesJSON)
	if err != nil {
		return false, nil, TraceNode{}, err
	}
	var node any
	if err := json.Unmarshal(can, &node); err != nil {
		return false, can, TraceNode{}, fmt.Errorf("invalid canonical JSON: %w", err)
	}
	m, err := toEvalMap(payload)
	if err != nil {
		return false, can, TraceNode{}, fmt.Errorf("invalid payload: %w", err)
	}
	tr := evalRuleTrace(node, m)
	return tr.Result, can, tr, nil
}

func evalRuleTrace(rule any, ctx any) TraceNode {
	m, ok := rule.(map[string]any)
	if !ok {
		return TraceNode{Kind: "none", Result: true}
	}
	kind := strings.ToLower(strVal(m["kind"]))
	switch kind {
	case "group":
		logic := strings.ToUpper(strVal(m["logic"]))
		children := anySlice(m["rules"])
		var kids []TraceNode
		matchedCount := 0
		for _, ch := range children {
			t := evalRuleTrace(ch, ctx)
			if t.Result {
				matchedCount++
			}
			kids = append(kids, t)
		}
		res := false
		if len(children) == 0 {
			res = false
		} else if logic == "OR" {
			res = matchedCount > 0
		} else { // AND default
			res = matchedCount == len(children)
		}
		msg := ""
		if !res {
			msg = fmt.Sprintf("group %s failed: %d/%d matched", strings.ToUpper(logic), matchedCount, len(children))
		}
		return TraceNode{Kind: "group", Logic: strings.ToUpper(logic), Result: res, Message: msg, Children: kids}

	case "scalar":
		path := strVal(m["path"])
		op := strings.ToLower(strVal(m["op"]))
		vtype := strings.ToLower(strVal(m["valuetype"]))
		actual := deepGet(ctx, path)
		pass := false
		switch vtype {
		case "number":
			pass = evalNumber(op, toFloat(actual), toFloat(m["value"]), toFloat(m["value2"]))
		case "boolean":
			pass = evalBool(op, toBool(actual), toBool(m["value"]))
		case "date":
			pass = evalDate(op, toTime(actual), toTime(m["value"]), toTime(m["value2"]))
		default:
			pass = evalString(op, toString(actual), m["value"]) // string/unknown
		}
		msg := ""
		if !pass {
			if vtype == "number" && op == "between" {
				msg = fmt.Sprintf("value %v not between %v and %v", toString(actual), toString(m["value"]), toString(m["value2"]))
			} else if vtype == "number" || vtype == "date" {
				msg = fmt.Sprintf("%s %v %s %v is false", path, toString(actual), op, toString(m["value"]))
			} else {
				msg = fmt.Sprintf("%s '%s' %s '%s' is false", path, toString(actual), op, toString(m["value"]))
			}
		}
		return TraceNode{Kind: "scalar", Path: path, Op: strings.ToLower(op), Value: m["value"], Value2: m["value2"], Actual: actual, Result: pass, Message: msg}

	case "array":
		path := strVal(m["path"])
		quant := strings.ToUpper(strVal(m["quantifier"]))
		var countPtr *int
		if c := toIntPtr(m["count"]); c != nil {
			countPtr = c
		}
		var idxPtr *int
		if i := toIntPtr(m["index"]); i != nil {
			idxPtr = i
		}
		s := anySlice(deepGet(ctx, path))
		size := len(s)
		kids := make([]TraceNode, 0, size)
		matches := 0
		matchedIdx := []int{}
		for i := 0; i < size; i++ {
			t := evalRuleTrace(m["predicate"], s[i])
			if t.Result {
				matches++
				matchedIdx = append(matchedIdx, i)
			}
			kids = append(kids, t)
		}
		res := false
		switch quant {
		case "ANY":
			res = matches > 0
		case "ALL":
			res = size == 0 || matches == size
		case "NONE":
			res = matches == 0
		case "COUNT_GTE":
			res = countPtr != nil && matches >= *countPtr
		case "COUNT_EQ":
			res = countPtr != nil && matches == *countPtr
		case "COUNT_LTE":
			res = countPtr != nil && matches <= *countPtr
		case "INDEX":
			if idxPtr != nil && *idxPtr >= 0 && *idxPtr < size {
				res = kids[*idxPtr].Result
			} else {
				res = false
			}
		}
		msg := ""
		if !res {
			switch quant {
			case "ANY":
				msg = fmt.Sprintf("no elements matched predicate (0/%d)", size)
			case "ALL":
				msg = fmt.Sprintf("only %d/%d elements matched predicate", matches, size)
			case "NONE":
				msg = fmt.Sprintf("%d elements matched but expected NONE", matches)
			case "COUNT_GTE":
				if countPtr != nil {
					msg = fmt.Sprintf("matched %d < required %d", matches, *countPtr)
				}
			case "COUNT_EQ":
				if countPtr != nil {
					msg = fmt.Sprintf("matched %d != required %d", matches, *countPtr)
				}
			case "COUNT_LTE":
				if countPtr != nil {
					msg = fmt.Sprintf("matched %d > allowed %d", matches, *countPtr)
				}
			case "INDEX":
				if idxPtr == nil || *idxPtr < 0 || *idxPtr >= size {
					msg = "index out of range"
				} else if !kids[*idxPtr].Result {
					msg = "predicate at index failed"
				}
			}
		}
		sz := size
		mt := matches
		return TraceNode{Kind: "array", Path: path, Quantifier: quant, Size: &sz, Matches: &mt, MatchedIdx: matchedIdx, Index: idxPtr, Result: res, Message: msg, Children: kids}
	}
	return TraceNode{Kind: kind, Result: true}
}

// ApplyRoleRulesNow loads the stored rules for roleID, evaluates them for all active users,
// and adds/removes the role accordingly. It returns the count of users actually changed.
func ApplyRoleRulesNow(ctx context.Context, roleID string) (int, error) {
	// if roleID <= 0 {
	// 	return 0, fmt.Errorf("invalid roleID")
	// }

	raw, _, err := database.GetRoleRulesJSON(roleID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return 0, ErrNotFound
		}
		return 0, err
	}

	// No rules stored → remove this role from everyone.
	if len(raw) == 0 {
		return database.RemoveRoleFromAllUsers(ctx, roleID)
	}

	var rules map[string]any
	if err := json.Unmarshal(raw, &rules); err != nil {
		return 0, fmt.Errorf("stored rules invalid JSON: %w", err)
	}

	users, err := database.ListActiveUsers(ctx)
	if err != nil {
		return 0, err
	}

	changed := 0
	for _, u := range users {
		u42, err := GetUser42(u.FtLogin)
		if err != nil {
			// Skip flaky users; don't fail the whole job.
			log.Printf("ApplyRoleRulesNow: GetUser42(%s) error: %v", u.FtLogin, err)
			continue
		}

		// Convert the typed struct to a generic map for the evaluator.
		var payload map[string]any
		if err := json.Unmarshal(mustMarshal(u42), &payload); err != nil {
			log.Printf("ApplyRoleRulesNow: marshal user payload %s error: %v", u.FtLogin, err)
			continue
		}

		shouldHave := evalRuleNode(rules, payload)
		c, err := database.EnsureUserRole(ctx, u.ID, roleID, shouldHave)
		if err != nil {
			return changed, err
		}
		if c {
			changed++
		}
	}

	return changed, nil
}

/* ================================
   Evaluator (mirrors front-end)
   ================================ */

func evalRuleNode(rule any, ctx any) bool {
	m, ok := rule.(map[string]any)
	if !ok {
		return true
	}
	switch strings.ToLower(strVal(m["kind"])) {
	case "group":
		logic := strings.ToUpper(strVal(m["logic"]))
		children := anySlice(m["rules"])

		if len(children) == 0 {
			return false
		}
		if logic == "OR" {
			for _, ch := range children {
				if evalRuleNode(ch, ctx) {
					return true
				}
			}
			return false
		}
		// AND (default)
		for _, ch := range children {
			if !evalRuleNode(ch, ctx) {
				return false
			}
		}
		return true

	case "scalar":
		path := strVal(m["path"])
		op := strings.ToLower(strVal(m["op"]))
		valueType := strings.ToLower(strVal(m["valuetype"]))
		v := deepGet(ctx, path)

		switch valueType {
		case "number":
			return evalNumber(op, toFloat(v), toFloat(m["value"]), toFloat(m["value2"]))
		case "boolean":
			return evalBool(op, toBool(v), toBool(m["value"]))
		case "date":
			return evalDate(op, toTime(v), toTime(m["value"]), toTime(m["value2"]))
		case "string", "unknown", "":
			return evalString(op, toString(v), m["value"])
		default:
			return evalString(op, toString(v), m["value"])
		}

	case "array":
		path := strVal(m["path"])
		quant := strings.ToUpper(strVal(m["quantifier"]))
		count := toIntPtr(m["count"])
		index := toIntPtr(m["index"])
		pred := m["predicate"]

		arr := deepGet(ctx, path)
		s := anySlice(arr)

		// Empty array semantics
		if len(s) == 0 {
			switch quant {
			case "ANY":
				return false
			case "NONE":
				return true
			case "ALL":
				return true
			case "COUNT_GTE", "COUNT_EQ", "COUNT_LTE":
				return compareCount(0, quant, count)
			case "INDEX":
				return false
			default:
				return false
			}
		}

		matches := 0
		childrenResults := make([]bool, len(s))
		for i, el := range s {
			ok := evalRuleNode(pred, el)
			childrenResults[i] = ok
			if ok {
				matches++
			}
		}

		switch quant {
		case "ANY":
			return matches > 0
		case "ALL":
			return matches == len(s)
		case "NONE":
			return matches == 0
		case "COUNT_GTE", "COUNT_EQ", "COUNT_LTE":
			return compareCount(matches, quant, count)
		case "INDEX":
			if index == nil || *index < 0 || *index >= len(childrenResults) {
				return false
			}
			return childrenResults[*index]
		default:
			return false
		}
	}
	return true
}

/* ================================
   Eval helpers
   ================================ */

func evalNumber(op string, a, b, c float64) bool {
	switch op {
	case "exists":
		return !mathIsNaN(a)
	case "notexists":
		return mathIsNaN(a)
	case "eq":
		return a == b
	case "neq":
		return a != b
	case "gt":
		return a > b
	case "gte":
		return a >= b
	case "lt":
		return a < b
	case "lte":
		return a <= b
	case "between":
		min := mathMin(b, c)
		max := mathMax(b, c)
		return a >= min && a <= max
	default:
		return false
	}
}

func evalBool(op string, a, b bool) bool {
	switch op {
	case "exists":
		// booleans "exist" once coerced; null checks happen before coercion in the UI,
		// here we treat missing as false unless op is exists/notexists handled elsewhere.
		return true
	case "notexists":
		return false
	case "eq":
		return a == b
	case "neq":
		return a != b
	default:
		return false
	}
}

func evalDate(op string, a, b, c *time.Time) bool {
	switch op {
	case "exists":
		return a != nil
	case "notexists":
		return a == nil
	}
	if a == nil || b == nil {
		return false
	}
	ta := a.UTC()
	tb := b.UTC()
	switch op {
	case "eq":
		return ta.Equal(tb)
	case "neq":
		return !ta.Equal(tb)
	case "gt", "after":
		return ta.After(tb)
	case "gte":
		return ta.After(tb) || ta.Equal(tb)
	case "lt", "before":
		return ta.Before(tb)
	case "lte":
		return ta.Before(tb) || ta.Equal(tb)
	case "between":
		if c == nil {
			return false
		}
		tc := c.UTC()
		start, end := tb, tc
		if end.Before(start) {
			start, end = end, start
		}
		return (ta.Equal(start) || ta.After(start)) && (ta.Equal(end) || ta.Before(end))
	default:
		return false
	}
}

func evalString(op, s string, value any) bool {
	switch op {
	case "exists":
		return s != ""
	case "notexists":
		return s == ""
	case "empty":
		return s == ""
	case "notempty":
		return s != ""
	case "eq":
		return s == toString(value)
	case "neq":
		return s != toString(value)
	case "contains":
		return strings.Contains(s, toString(value))
	case "startswith":
		return strings.HasPrefix(s, toString(value))
	case "endswith":
		return strings.HasSuffix(s, toString(value))
	case "regex":
		pat := toString(value)
		if pat == "" {
			return false
		}
		re, err := regexp.Compile(pat)
		if err != nil {
			return false
		}
		return re.MatchString(s)
	case "in":
		arr := anySlice(value)
		for _, v := range arr {
			if s == toString(v) {
				return true
			}
		}
		return false
	case "notin":
		arr := anySlice(value)
		for _, v := range arr {
			if s == toString(v) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func compareCount(matches int, quant string, count *int) bool {
	n := 0
	if count != nil {
		n = *count
	}
	switch quant {
	case "COUNT_GTE":
		return matches >= n
	case "COUNT_EQ":
		return matches == n
	case "COUNT_LTE":
		return matches <= n
	default:
		return false
	}
}

/* ================================
   Generic helpers
   ================================ */

func deepGet(obj any, path string) any {
	if path == "" {
		return obj
	}
	parts := strings.Split(path, ".")
	cur := obj
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		var exists bool
		cur, exists = m[p]
		if !exists {
			return nil
		}
	}
	return cur
}

func anySlice(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case []map[string]any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = x[i]
		}
		return out
	default:
		return nil
	}
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case time.Time:
		return t.Format(time.RFC3339)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func toFloat(v any) float64 {
	if v == nil {
		return math.NaN()
	}
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		if t == "" {
			return math.NaN()
		}
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return math.NaN()
		}
		return f
	case bool:
		if t {
			return 1
		}
		return 0
	default:
		return math.NaN()
	}
}

func toBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "true" || s == "1" || s == "yes" || s == "y"
	case float64:
		return t != 0
	case json.Number:
		f, _ := t.Float64()
		return f != 0
	default:
		return v != nil
	}
}

func toTime(v any) *time.Time {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case time.Time:
		return &t
	case *time.Time:
		return t
	case string:
		if ts, ok := tryParseDate(t); ok {
			return &ts
		}
	case float64:
		sec := int64(t)
		ts := time.Unix(sec, 0).UTC()
		return &ts
	}
	return nil
}

func tryParseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func toIntPtr(v any) *int {
	switch t := v.(type) {
	case nil:
		return nil
	case float64:
		x := int(t)
		return &x
	case json.Number:
		i, _ := strconv.Atoi(t.String())
		return &i
	case string:
		if t == "" {
			return nil
		}
		i, err := strconv.Atoi(t)
		if err != nil {
			return nil
		}
		return &i
	default:
		return nil
	}
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return toString(v)
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// tiny math helpers (avoid importing math just for NaN/Min/Max if you prefer)
func mathIsNaN(f float64) bool { return f != f }
func mathMin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// GetRoleRules loads the persisted rules JSON for a role.
// Returns (nil, nil, ErrNotFound) if the role row doesn't exist.
func GetRoleRules(roleID string) ([]byte, *time.Time, error) {
	bytes, updatedAt, err := database.GetRoleRulesJSON(roleID)
	if err != nil {
		// database layer returns sql.ErrNoRows when role doesn't exist
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	return bytes, updatedAt, nil
}
