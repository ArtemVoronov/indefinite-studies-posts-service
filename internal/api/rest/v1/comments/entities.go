package comments

type CommentDTO struct {
	Id                int
	Uuid              string
	AuthorUuid        string
	PostUuid          string
	LinkedCommentUuid string
	Text              string
	State             string
}

type CommentListDTO struct {
	Count  int
	Offset int
	Limit  int
	Data   []CommentDTO
}

type CommentEditDTO struct {
	CommentId   int     `json:"CommentId" binding:"required"`
	CommentUuid string  `json:"CommentUuid" binding:"required"`
	PostUuid    string  `json:"PostUuid" binding:"required"`
	Text        *string `json:"Text,omitempty"`
	State       *string `json:"State,omitempty"`
	AuthorUuid  string  `json:"AuthorUuid" binding:"required"`
}

type CommentCreateDTO struct {
	AuthorUuid        string `json:"AuthorUuid" binding:"required"`
	PostUuid          string `json:"PostUuid" binding:"required"`
	Text              string `json:"Text" binding:"required"`
	LinkedCommentUuid string `json:"LinkedCommentUuid,omitempty"`
}

type CommentDeleteDTO struct {
	CommentId   int    `json:"CommentId" binding:"required"`
	CommentUuid string `json:"CommentUuid" binding:"required"`
	PostUuid    string `json:"PostId" binding:"required"`
}
