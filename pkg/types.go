package pkg

type User struct {
	Login     string `json:"login"`
	Id        int    `json:"id"`
	Url       string `json:"html_url"`
	AvatarUrl string `json:"avatar_url"`
}

type Reactions struct {
	Url        string `json:"url"`
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
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Default     bool   `json:"default"`
}

type Issue struct {
	Id        int       `json:"id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Url       string    `json:"html_url"`
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
