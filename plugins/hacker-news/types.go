package main

type HackerNewsAPIResponse struct {
	ID          int    `json:"id"`
	Deleted     bool   `json:"deleted"`
	Type        string `json:"type"`
	By          string `json:"by"`
	Time        int    `json:"time"`
	Text        string `json:"text"`
	Dead        bool   `json:"dead"`
	Parent      int    `json:"parent"`
	Poll        int    `json:"poll"`
	Kids        []int  `json:"kids"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
	Title       string `json:"title"`
	Parts       []int  `json:"parts"`
	Descendants int    `json:"descendants"`
}

type HitsAlgolia struct {
	Tags        []string `json:"_tags"`
	Author      string   `json:"author"`
	CreatedAt   string   `json:"created_at"`
	CreatedAtI  int      `json:"created_at_i"`
	NumComments int      `json:"num_comments"`
	ObjectID    string   `json:"objectID"`
	Points      int      `json:"points"`
	StoryID     int      `json:"story_id"`
	StoryText   string   `json:"story_text"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	UpdatedAt   string   `json:"updated_at"`

	// Comments related
	CommentText string `json:"comment_text"`
	ParentID    int    `json:"parent_id"`
	StoryTitle  string `json:"story_title"`
}

type HackerNewsAPIResponseAlgolia struct {
	Hits        []HitsAlgolia `json:"hits"`
	HitsPerPage int           `json:"hitsPerPage"`
	NbHits      int           `json:"nbHits"`
	Page        int           `json:"page"`
	NbPages     int           `json:"nbPages"`
}

type HackerNewsUserAPIResponse struct {
	ID        string `json:"id"`
	Created   int    `json:"created"`
	Karma     int    `json:"karma"`
	About     string `json:"about"`
	Submitted []int  `json:"submitted"`
}
