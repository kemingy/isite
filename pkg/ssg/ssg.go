package ssg

import (
	"github.com/kemingy/isite/pkg/types"
)

type StaticSiteGenerator interface {
	Generate(issues []types.Issue, outputDir string) error
}

func NewGenerator(engine, title, theme, themeRepo, baseURL string, feed bool) StaticSiteGenerator {
	switch engine {
	case "zola":
		return NewZola(title, baseURL, theme, themeRepo, feed)
	default:
		return nil
	}
}
