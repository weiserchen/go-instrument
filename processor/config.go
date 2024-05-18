package processor

type TraceConfig struct {
	App           string
	Overwrite     bool
	DefaultSelect bool
	SkipGenerated bool
}

type LicenseConfig struct {
	License string
}
