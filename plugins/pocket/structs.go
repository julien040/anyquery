package main

type stateRetrieve string

var retrieveAll stateRetrieve = "all"
var retrieveUnread stateRetrieve = "unread"
var retrieveArchive stateRetrieve = "archive"

type sortRetrieve string

var sortNewest sortRetrieve = "newest"
var sortOldest sortRetrieve = "oldest"
var sortTitle sortRetrieve = "title"
var sortSite sortRetrieve = "site"

type detailType string

var detailTypeSimple detailType = "simple"
var detailTypeComplete detailType = "complete"

type retrieveRequest struct {
	ConsumerKey string        `json:"consumer_key"`
	AccessToken string        `json:"access_token"`
	Count       int           `json:"count,omitempty"`
	Offset      int           `json:"offset,omitempty"`
	State       stateRetrieve `json:"state,omitempty"`
	Sort        sortRetrieve  `json:"sort,omitempty"`
	Search      string        `json:"search,omitempty"`
	Domain      string        `json:"domain,omitempty"`
	DetailType  detailType    `json:"detail_type,omitempty"`
}

type retrieveResponseItem struct {
	ItemID                 string `json:"item_id"`
	ResolvedID             string `json:"resolved_id"`
	GivenURL               string `json:"given_url"`
	ResolvedURL            string `json:"resolved_url"`
	GivenTitle             string `json:"given_title"`
	ResolvedTitle          string `json:"resolved_title"`
	Favorite               string `json:"favorite"`
	Status                 string `json:"status"`
	TimeAdded              string `json:"time_added"`
	TimeUpdated            string `json:"time_updated"`
	TimeFavorited          string `json:"time_favorited"`
	TimeRead               string `json:"time_read"`
	Excerpt                string `json:"excerpt"`
	IsArticle              string `json:"is_article"`
	HasImage               string `json:"has_image"`
	HasVideo               string `json:"has_video"`
	WordCount              string `json:"word_count"`
	Lang                   string `json:"lang"`
	TimeToRead             int    `json:"time_to_read"`
	ListenDurationEstimate int    `json:"listen_duration_estimate"`
}

type retrieveResponse struct {
	Status int                             `json:"status"`
	List   map[string]retrieveResponseItem `json:"list"`
}

/* -------------------------------------------------------------------------- */
/*                                   Actions                                  */
/* -------------------------------------------------------------------------- */

type addAction struct {
	Action  string `json:"action"`
	Tags    string `json:"tags,omitempty"`
	Title   string `json:"title,omitempty"`
	URL     string `json:"url"`
	Time    string `json:"time,omitempty"`
	TweetID string `json:"ref_id,omitempty"`
}

type deleteAction struct {
	Action string `json:"action"`
	ItemID string `json:"item_id"`
}

type actionResponse struct {
	Status        int         `json:"status"`
	ActionResults interface{} `json:"action_results"`
	ActionErrors  interface{} `json:"action_errors"`
}
