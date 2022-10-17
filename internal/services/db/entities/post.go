package entities

import (
	"time"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
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

func GetPossiblePostStates() []string {
	return []string{entities.POST_STATE_NEW, entities.POST_STATE_ON_MODERATION, entities.POST_STATE_PUBLISHED, entities.POST_STATE_BLOCKED, entities.POST_STATE_DELETED}
}
