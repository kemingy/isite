package ssg

import (
	"github.com/kemingy/isite/pkg/models"
)

type StaticSiteGenerator interface {
	Generate(issues []models.Issue, outputDir string) error
}

func NewGenerator(cmd *models.Command, meta *models.Repository) StaticSiteGenerator {
	switch cmd.Engine {
	case "zola":
		return NewZola(cmd, meta)
	default:
		return nil
	}
}
