package utils

import (
	"encoding/json"
	"os"
)

func LoadRawSpec() (map[string]any, error) {
	data, err := os.ReadFile("./docs/swagger.json")
	if err != nil {
		return nil, err
	}
	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	return spec, nil
}

func PruneTags(spec map[string]any) {
	used := map[string]struct{}{}
	for _, rawOp := range spec["paths"].(map[string]any) {
		for _, op := range rawOp.(map[string]any) {
			if opMap, ok := op.(map[string]any); ok {
				if tags, ok := opMap["tags"].([]any); ok {
					for _, t := range tags {
						if ts, ok := t.(string); ok {
							used[ts] = struct{}{}
						}
					}
				}
			}
		}
	}
	if rawTags, ok := spec["tags"].([]any); ok {
		out := make([]any, 0, len(rawTags))
		for _, t := range rawTags {
			if tagObj, ok := t.(map[string]any); ok {
				if name, ok := tagObj["name"].(string); ok {
					if _, keep := used[name]; keep {
						out = append(out, tagObj)
					}
				}
			}
		}
		spec["tags"] = out
	}
}

// filterSpec returns a new spec containing only the paths for which keep(path)==true
func FilterSpec(raw map[string]any, keep func(string) bool) map[string]any {
	out := make(map[string]any, len(raw))
	// copy top‐level entries except "paths"
	for k, v := range raw {
		if k != "paths" {
			out[k] = v
		}
	}
	// filter paths
	paths := raw["paths"].(map[string]any)
	filtered := make(map[string]any, len(paths))
	for p, op := range paths {
		if keep(p) {
			filtered[p] = op
		}
	}
	out["paths"] = filtered
	return out
}
