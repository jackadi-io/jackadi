package types

type PluginChanges struct {
	Name     string
	FileName string
	New      bool
	Updated  bool
	Deleted  bool
}

type PluginUpdateResponse struct {
	Changes []PluginChanges
	Error   error
}
