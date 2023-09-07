package ssg

import (
	"github.com/kemingy/isite/pkg/types"
)

type StaticSiteGenerator interface {
	Generate(issues []types.Issue, outputDir string) error
}

func NewGenerator(engine, title, theme, themeRepo, baseUrl string, feed bool) StaticSiteGenerator {
	switch engine {
	case "zola":
		return NewZola(title, baseUrl, theme, themeRepo, feed)
	default:
		return nil
	}
}
