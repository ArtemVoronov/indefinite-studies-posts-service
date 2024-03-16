package entities

import (
	"time"
)

type Post struct {
	Id             int
	Uuid           string
	AuthorUuid     string
	Text           string
	PreviewText    string
	Topic          string
	State          string
	CreateDate     time.Time
	LastUpdateDate time.Time
}

type PostWithTags struct {
	Post
	TagIds []int
}

type PostWithTagsForQueue struct {
	PostUuid string
	TagIds   []int
}
