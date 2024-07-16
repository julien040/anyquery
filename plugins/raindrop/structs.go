package main

type RaindropListItemResponse struct {
	Result       bool   `json:"result"`
	Items        []Item `json:"items"`
	Count        int64  `json:"count"`
	CollectionID int64  `json:"collectionId"`
}

type Item struct {
	ID           int64       `json:"_id"`
	Link         string      `json:"link"`
	Title        string      `json:"title"`
	Excerpt      string      `json:"excerpt"`
	Note         string      `json:"note"`
	Type         Type        `json:"type"`
	User         User        `json:"user"`
	Cover        string      `json:"cover"`
	Media        []Media     `json:"media"`
	Tags         []string    `json:"tags"`
	Important    *bool       `json:"important,omitempty"`
	Reminder     *Reminder   `json:"reminder,omitempty"`
	Removed      bool        `json:"removed"`
	Created      string      `json:"created"`
	Collection   Collection  `json:"collection"`
	Highlights   []Highlight `json:"highlights"`
	LastUpdate   string      `json:"lastUpdate"`
	Domain       string      `json:"domain"`
	CreatorRef   CreatorRef  `json:"creatorRef"`
	Sort         int64       `json:"sort"`
	CollectionID int64       `json:"collectionId"`
}

type Collection struct {
	Ref CollectionRef `json:"$ref"`
	ID  int64         `json:"$id"`
	OID int64         `json:"oid"`
}

type CreatorRef struct {
	ID     int64  `json:"_id"`
	Avatar string `json:"avatar"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

type Highlight struct {
	Text       string  `json:"text"`
	Note       string  `json:"note"`
	Color      *string `json:"color,omitempty"`
	Created    string  `json:"created"`
	LastUpdate string  `json:"lastUpdate"`
	CreatorRef int64   `json:"creatorRef"`
	ID         string  `json:"_id"`
}

type Media struct {
	Link string `json:"link"`
	Type Type   `json:"type"`
}

type Reminder struct {
	Date string `json:"date"`
}

type User struct {
	Ref UserRef `json:"$ref"`
	ID  int64   `json:"$id"`
}

type CollectionRef string

const (
	Collections CollectionRef = "collections"
)

type Type string

const (
	Article Type = "article"
	Image   Type = "image"
	Link    Type = "link"
)

type UserRef string

const (
	Users UserRef = "users"
)

type CreateItem struct {
	PleaseParse struct{}   `json:"pleaseParse"`
	Created     string     `json:"created,omitempty"`
	LastUpdate  string     `json:"lastUpdate,omitempty"`
	Order       int64      `json:"order"`
	Important   bool       `json:"important,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	Cover       string     `json:"cover,omitempty"`
	Collection  Collection `json:"collection,omitempty"`
	Type        Type       `json:"type,omitempty"`
	Excerpt     string     `json:"excerpt,omitempty"`
	Title       string     `json:"title,omitempty"`
	Link        string     `json:"link"`
	Reminder    Reminder   `json:"reminder,omitempty"`
}

type MultipleCreateItemRequest struct {
	Items []CreateItem `json:"items"`
}

type MultipleCreateItemResponse struct {
	Result bool `json:"result"`
}

type MultipleDeleteItemRequest struct {
	IDs []int64 `json:"ids"`
}

type MultipleDeleteItemResponse struct {
	Result   bool  `json:"result"`
	Modified int64 `json:"modified"`
}
