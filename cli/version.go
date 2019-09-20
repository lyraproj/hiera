package cli

import "fmt"

var (
	// BuildTag set at build time, empty if not a tagged version
	BuildTag string
	// BuildTime set at build time
	BuildTime string
	// BuildSHA set at build time
	BuildSHA string
)

type version struct {
	buildTag  string
	buildTime string
	buildSHA  string
}

// Get the structured version
func getVersion() *version {
	tag := BuildTag
	if len(tag) == 0 {
		tag = "dirty"
	}

	return &version{
		buildTag:  tag,
		buildTime: BuildTime,
		buildSHA:  BuildSHA,
	}
}

// String returns a simplified version string consisting of <Git SHA>-<Git Tag>
func (v *version) String() string {
	return fmt.Sprintf("%s-%s", v.buildSHA, v.buildTag)
}
