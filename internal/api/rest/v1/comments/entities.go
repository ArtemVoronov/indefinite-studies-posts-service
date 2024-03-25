package comments

import "time"

type CommentDTO struct {
	Id              int
	AuthorUuid      string
	PostUuid        string
	LinkedCommentId *int `json:"LinkedCommentId,omitempty"`
	Text            string
	State           string
	CreateDate      time.Time
	LastUpdateDate  time.Time
}

type CommentListDTO struct {
	Count  int
	Offset int
	Limit  int
	Data   []CommentDTO
}

type CommentEditDTO struct {
	CommentId  int     `json:"CommentId" binding:"required"`
	PostUuid   string  `json:"PostUuid" binding:"required"`
	Text       *string `json:"Text,omitempty"`
	State      *string `json:"State,omitempty"`
	AuthorUuid string  `json:"AuthorUuid" binding:"required"`
}

type CommentCreateDTO struct {
	AuthorUuid      string `json:"AuthorUuid" binding:"required"`
	PostUuid        string `json:"PostUuid" binding:"required"`
	Text            string `json:"Text" binding:"required"`
	LinkedCommentId *int   `json:"LinkedCommentId,omitempty"`
}

type CommentDeleteDTO struct {
	CommentId int    `json:"CommentId" binding:"required"`
	PostUuid  string `json:"PostId" binding:"required"`
}
