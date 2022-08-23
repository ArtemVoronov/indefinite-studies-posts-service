package entities

import "time"

type Post struct {
	Id             int
	AuthorId       int
	Text           string
	Topic          string
	State          string
	CreateDate     time.Time
	LastUpdateDate time.Time
}

const (
	POST_STATE_NEW     string = "NEW"
	POST_STATE_BLOCKED string = "BLOCKED"
	POST_STATE_DELETED string = "DELETED"
)

func GetPossiblePostStates() []string {
	return []string{POST_STATE_NEW, POST_STATE_BLOCKED, POST_STATE_DELETED}
}
