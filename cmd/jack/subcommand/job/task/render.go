package task

import (
	"fmt"
	"maps"
	"slices"

	"github.com/goccy/go-yaml"

	"github.com/jackadi-io/jackadi/cmd/jack/option"
	"github.com/jackadi-io/jackadi/cmd/jack/style"
	"github.com/jackadi-io/jackadi/internal/proto"
	"github.com/jackadi-io/jackadi/internal/serializer"
)

func printTaskResult(responses *proto.FwdResponse) {
	in := ""
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

		in += style.Title(id)
		if res.InternalError > 0 {
			in += style.InlineBlockTitle("id") + fmt.Sprintf("%d", res.GetId())
			in += style.InlineBlockTitle("groupID") + fmt.Sprintf("%d", res.GetGroupID())
			in += style.InlineBlockTitle("internal error") + style.ErrorStyle.Render(res.GetInternalError().String())
			if res.GetModuleError() != "" {
				in += style.Block(res.GetModuleError())
			}
			in += "\n"
			continue
		}

		if res.GetOutput() != nil {
			in += style.BlockTitle("output")
			var parsed any
			if err := serializer.JSON.Unmarshal(res.GetOutput(), &parsed); err != nil {
				in += style.Block(string(res.GetOutput()))
			}
			out, _ := yaml.MarshalWithOptions(parsed, yaml.UseLiteralStyleIfMultiline(true))
			in += style.Block(string(out))
		} else {
			in += style.InlineBlockTitle("output")
			in += style.Emph("empty")
		}

		if res.GetError() != "" {
			in += style.BlockTitle("err")
			in += style.Block(res.GetError())
		}

		if res.GetRetcode() > 0 {
			in += style.InlineBlockTitle("retcode")
			in += fmt.Sprintf("%d", res.GetRetcode())
		}

		in += "\n"
	}

	style.PrettyPrint(in)
}
