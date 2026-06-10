package version

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func Full() string {
	return Version + " (" + GitCommit + ", " + BuildTime + ")"
}
