package ssg

import (
	"bytes"
	"strings"

	chromaHTML "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/cockroachdb/errors"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkHTML "github.com/yuin/goldmark/renderer/html"

	"github.com/kemingy/isite/pkg/models"
)

const (
	engineZola         = "zola"
	engineAstro        = "astro"
	engineHugo         = "hugo"
	templateTOMLEscape = "toml_escape"
)

var commentHTMLPolicy = newCommentHTMLPolicy()

var markdownRenderer = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		emoji.Emoji,
		highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
			highlighting.WithFormatOptions(chromaHTML.WithClasses(true), chromaHTML.Standalone(false)),
		),
	),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(goldmarkHTML.WithUnsafe()),
)

func newCommentHTMLPolicy() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	policy.RequireNoFollowOnLinks(true)
	policy.AddTargetBlankToFullyQualifiedLinks(true)
	policy.AllowStandardURLs()
	policy.AllowElements("blockquote", "pre", "code", "p", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li", "details", "summary", "sub")
	policy.AllowAttrs("class").OnElements("div", "pre", "code", "span")
	policy.AllowAttrs("style").OnElements("p", "div", "h1", "h2", "h3", "h4", "h5", "h6")
	policy.AllowStyles("text-align").MatchingEnum("left", "center", "right", "justify").OnElements(
		"p", "div", "h1", "h2", "h3", "h4", "h5", "h6",
	)
	return policy
}

func sanitizeMarkdownSource(body string) string {
	return string(commentHTMLPolicy.SanitizeBytes([]byte(body)))
}

func renderAndSanitizeMarkdown(body string) string {
	var rendered bytes.Buffer
	if err := markdownRenderer.Convert([]byte(body), &rendered); err != nil {
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
