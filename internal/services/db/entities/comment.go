package entities

import "time"

type Comment struct {
	Id                int
	Uuid              string
	AuthorUuid        string
	PostId            int
	LinkedCommentUuid string
	Text              string
	State             string
	CreateDate        time.Time
	LastUpdateDate    time.Time
}
