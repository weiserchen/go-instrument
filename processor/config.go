package processor

var (
	DefaultTraceConfig TraceConfig = TraceConfig{
		App:           "app",
		Overwrite:     false,
		DefaultSelect: true,
		SkipGenerated: false,
	}
)

type TraceConfig struct {
	App           string
	Overwrite     bool
	DefaultSelect bool
	SkipGenerated bool
}

type LicenseConfig struct {
	License string
}
