package version

import (
	"encoding/json"
	"runtime"
)

var (
	Package = "github.com/kemingy/isite"

	version   = "0.0.0+unknown"
	buildDate = "1970-01-01T00:00:00Z"
	gitCommit = ""
	gitTag    = ""
)

type Version struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	GitCommit string `json:"git_commit"`
	GitTag    string `json:"git_tag"`
	GoVersion string `json:"go_version"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

func (v *Version) PrettyString() string {
	str, _ := json.MarshalIndent(v, "", "\t")
	return string(str)
}

func GetVersion() string {
	if gitTag != "" {
		return gitTag
	}
	return version
}

func GetVersionInfo() Version {
	return Version{
		Version:   GetVersion(),
		BuildDate: buildDate,
		GitCommit: gitCommit,
		GitTag:    gitTag,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}
