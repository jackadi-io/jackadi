package result

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/jackadi-io/jackadi/cmd/jack/style"
	"github.com/jackadi-io/jackadi/internal/serializer"
)

func prettySprint(in []byte) (string, error) {
	var parsed map[string]any
	// Needs serialize.JSON to have UseNumber enabled
	if err := serializer.JSON.Unmarshal(in, &parsed); err != nil {
		return "", err
	}

	keys := make([]string, 0, len(parsed))
	for k := range parsed {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	items := ""
	title := ""
	for _, k := range keys {
		if subMap, ok := parsed[k].(map[string]any); ok {
			subKeys := make([]string, 0, len(subMap))
			for k := range subMap {
				subKeys = append(subKeys, k)
			}

			sort.Strings(subKeys)

			for _, k2 := range subKeys {
				if k2 == "output" {
					subMap[k2] = outputToPretty(subMap, k2)
				}

				content := fmt.Sprintf("%v", subMap[k2])
				if strings.Contains(content, "\n") {
					items += style.BlockTitle(k2) + style.Block(content)
				} else {
					items += style.InlineBlockTitle(k2) + content
				}
			}
		} else {
			title += style.Title(fmt.Sprintf("%s: %v", k, parsed[k]))
		}
	}
	return title + style.SpacedBlock(items), nil
}

func outputToPretty(subMap map[string]any, k2 string) any {
	val, ok := subMap[k2].(string)
	if !ok {
		return subMap[k2]
	}

	decoded, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return subMap[k2]
	}

	var out any
	if err := serializer.JSON.Unmarshal(decoded, &out); err != nil {
		return subMap[k2]
	}

	prettyOut, err := yaml.MarshalWithOptions(out, yaml.UseLiteralStyleIfMultiline(true))
	if err != nil {
		return string(decoded)
	}

	return string(prettyOut)
}
