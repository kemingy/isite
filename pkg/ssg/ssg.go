package ssg

import (
	"bytes"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldmarkHTML "github.com/yuin/goldmark/renderer/html"

	"github.com/kemingy/isite/pkg/models"
)

const (
	engineZola         = "zola"
	engineAstro        = "astro"
	engineHugo         = "hugo"
	templateTOMLEscape = "toml_escape"
)

var commentMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(goldmarkHTML.WithUnsafe()),
)

var commentHTMLPolicy = newCommentHTMLPolicy()

func newCommentHTMLPolicy() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	policy.RequireNoFollowOnLinks(true)
	policy.AddTargetBlankToFullyQualifiedLinks(true)
	policy.AllowStandardURLs()
	policy.AllowAttrs("style").OnElements("p", "div", "h1", "h2", "h3", "h4", "h5", "h6")
	policy.AllowStyles("text-align").MatchingEnum("left", "center", "right", "justify").OnElements(
		"p", "div", "h1", "h2", "h3", "h4", "h5", "h6",
	)
	return policy
}

func renderAndSanitizeCommentBody(body string) string {
	var rendered bytes.Buffer
	if err := commentMarkdown.Convert([]byte(body), &rendered); err != nil {
		return ""
	}
	return string(commentHTMLPolicy.SanitizeBytes(rendered.Bytes()))
}

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
