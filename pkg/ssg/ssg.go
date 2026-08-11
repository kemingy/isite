package ssg

import (
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/kemingy/isite/pkg/models"
)

type StaticSiteGenerator interface {
	Generate(issues []models.Issue, outputDir string) error
}

func validateOutputDir(outputDir string) error {
	if strings.TrimSpace(outputDir) == "" {
		return errors.New("output directory must not be empty")
	}
	return nil
}

func NewGenerator(cmd *models.Command, meta *models.Repository) StaticSiteGenerator {
	switch cmd.Engine {
	case "zola":
		return NewZola(cmd, meta)
	case "astro":
		return NewAstro(cmd, meta)
	default:
		return nil
	}
}
