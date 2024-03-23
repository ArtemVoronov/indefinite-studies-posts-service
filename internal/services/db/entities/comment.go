package entities

import "time"

type Comment struct {
	Id              int
	AuthorUuid      string
	PostUuid        string
	LinkedCommentId *int
	Text            string
	State           string
	CreateDate      time.Time
	LastUpdateDate  time.Time
}

type CommentForQueue struct {
	PostUuid   string
	CommentId  int
	CreateDate time.Time
	State      string
}
