package ssg

import (
	"github.com/kemingy/isite/pkg"
)

type StaticSiteGenerator interface {
	Generate(issues []pkg.Issue) error
}
