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
	Tags []Tag
}

func (post PostWithTags) TagIds() []int {
	result := make([]int, 0, len(post.Tags))
	for k := range post.Tags {
		result = append(result, k)
	}
	return result
}

type PostWithTagsForQueue struct {
	PostUuid string
	TagIds   []int
}
