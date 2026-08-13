package ssg

import (
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/kemingy/isite/pkg/models"
)

const (
	engineZola  = "zola"
	engineAstro = "astro"
	engineHugo  = "hugo"
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
	case engineZola:
		return NewZola(cmd, meta)
	case engineAstro:
		return NewAstro(cmd, meta)
	case engineHugo:
		return NewHugo(cmd, meta)
	default:
		return nil
	}
}
