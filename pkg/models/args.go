package models

type Command struct {
	// filter
	Creator string
	State   string
	Label   []string
	// config
	Engine    string
	Output    string
	Title     string
	Theme     string
	ThemeRepo string
	BaseURL   string
	Feed      bool
	Katex     bool
}
