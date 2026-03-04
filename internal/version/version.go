package version

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func BuildInfo() map[string]string {
	return map[string]string{
		"version":    Version,
		"git_commit": GitCommit,
		"build_date": BuildDate,
	}
}
