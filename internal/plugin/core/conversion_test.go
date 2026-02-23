package core

import (
	"reflect"
	"testing"
)

func TestConvertToType(t *testing.T) {
	tests := map[string]struct {
		value      string
		targetType reflect.Type
		want       any
	}{
		"int": {
			value:      "7",
			targetType: reflect.TypeFor[int](),
			want:       int(7),
		},
		"int8": {
			value:      "7",
			targetType: reflect.TypeFor[int8](),
			want:       int8(7),
		},
		"int16": {
			value:      "7",
			targetType: reflect.TypeFor[int16](),
			want:       int16(7),
		},
		"int32": {
			value:      "7",
			targetType: reflect.TypeFor[int32](),
			want:       int32(7),
		},
		"int64": {
			value:      "7",
			targetType: reflect.TypeFor[int64](),
			want:       int64(7),
		},
	}

	for name, test := range tests {
		result, err := StructpbValueToInput(test.value, test.targetType)
		if err != nil {
			t.Fatalf("[%s], unexpected error: %s", name, err)
		}
		if result != test.want {
			t.Fatalf("[%s], conversion failed, got: %s, want: %s", name, result, test.want)
		}
	}
}
