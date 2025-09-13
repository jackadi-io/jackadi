package serializer

import (
	jsoniter "github.com/json-iterator/go"
)

// reserialize to is a trick to be able to convert easily any kind of structure to a structpb.Value, and to enable custom struct tag.
//
// jsoniter is faster, keep the same de.ser behavior than JSON used by structpb.Value.
// but it is also useful to better handle struct tags:
// - does not use "json" struct tag as users are not expecting to have the struct tag reused when returning result to Jackadi.
// - enables the user to control the output field name with "jackadi" struct tag.
var JSON = jsoniter.Config{
	TagKey:    "jackadi",
	UseNumber: true,
}.Froze()
