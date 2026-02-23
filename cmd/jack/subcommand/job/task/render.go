package task

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/jackadi-io/jackadi/cmd/jack/option"
	"github.com/jackadi-io/jackadi/cmd/jack/style"
	"github.com/jackadi-io/jackadi/internal/proto"
	"github.com/jackadi-io/jackadi/internal/serializer"
)

func printTaskResult(responses *proto.FwdResponse) {
	var in strings.Builder
	allResponses := responses.GetResponses()
	keys := maps.Keys(allResponses)
	if option.GetSortOutput() {
		keys = slices.Values(slices.Sorted(keys))
	}

	for id := range keys {
		res, ok := allResponses[id]
		if !ok || res == nil {
			continue
		}

		in.WriteString(style.Title(id))
		if res.InternalError > 0 {
			in.WriteString(style.InlineBlockTitle("id") + fmt.Sprintf("%d", res.GetId()))
			in.WriteString(style.InlineBlockTitle("groupID") + fmt.Sprintf("%d", res.GetGroupID()))
			in.WriteString(style.InlineBlockTitle("internal error") + style.ErrorStyle.Render(res.GetInternalError().String()))
			if res.GetModuleError() != "" {
				in.WriteString(style.Block(res.GetModuleError()))
			}
			in.WriteString("\n")
			continue
		}

		if res.GetOutput() != nil {
			in.WriteString(style.BlockTitle("output"))
			var parsed any
			if err := serializer.JSON.Unmarshal(res.GetOutput(), &parsed); err != nil {
				in.WriteString(style.Block(string(res.GetOutput())))
			}
			out, _ := yaml.MarshalWithOptions(parsed, yaml.UseLiteralStyleIfMultiline(true))
			in.WriteString(style.Block(string(out)))
		} else {
			in.WriteString(style.InlineBlockTitle("output"))
			in.WriteString(style.Emph("empty"))
		}

		if res.GetError() != "" {
			in.WriteString(style.BlockTitle("err"))
			in.WriteString(style.Block(res.GetError()))
		}

		if res.GetRetcode() > 0 {
			in.WriteString(style.InlineBlockTitle("retcode"))
			in.WriteString(fmt.Sprintf("%d", res.GetRetcode()))
		}

		in.WriteString("\n")
	}

	style.PrettyPrint(in.String())
}
