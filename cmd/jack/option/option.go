package option

var JSONFormat *bool
var SortOutput *bool

func GetJSONFormat() bool {
	if JSONFormat == nil {
		return false
	}
	return *JSONFormat
}

func GetSortOutput() bool {
	if SortOutput == nil {
		return true // Default to true for backward compatibility
	}
	return *SortOutput
}
