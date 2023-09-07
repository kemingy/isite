package types

type User struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	URL       string `json:"html_url"`
	AvatarURL string `json:"avatar_url"`
}

type Reactions struct {
	URL        string `json:"url"`
	TotalCount int    `json:"total_count"`
	ThumbUp    int    `json:"+1"`
	ThumbDown  int    `json:"-1"`
	Laugh      int    `json:"laugh"`
	Hooray     int    `json:"hooray"`
	Confused   int    `json:"confused"`
	Heart      int    `json:"heart"`
	Rocket     int    `json:"rocket"`
	Eyes       int    `json:"eyes"`
}

type Label struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Default     bool   `json:"default"`
}

type Issue struct {
	ID        int       `json:"id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"html_url"`
	Body      string    `json:"body"`
	User      User      `json:"user"`
	State     string    `json:"state"`
	Locked    bool      `json:"locked"`
	Labels    []Label   `json:"labels"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	ClosedAt  string    `json:"closed_at"`
	Comments  int       `json:"comments"`
	Reactions Reactions `json:"reactions"`
}
