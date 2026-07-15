package version

import "fmt"

var (
	Version = "1.0.0-dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func Current() Info {
	return Info{Version: Version, Commit: Commit, Date: Date}
}

func String() string {
	return fmt.Sprintf("areaflow %s commit=%s built=%s", Version, Commit, Date)
}
