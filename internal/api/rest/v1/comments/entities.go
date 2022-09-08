package comments

type CommentDTO struct {
	Id              int
	AuthorId        int
	PostId          int
	LinkedCommentId *int
	Text            string
	State           string
}

type CommentListDTO struct {
	Count  int
	Offset int
	Limit  int
	Data   []CommentDTO
}

type CommentEditDTO struct {
	Id    int     `json:"Id" binding:"required"`
	Text  *string `json:"Text,omitempty"`
	State *string `json:"State,omitempty"`
}

type CommentCreateDTO struct {
	AuthorId        int    `json:"AuthorId" binding:"required"`
	PostId          int    `json:"PostId" binding:"required"`
	Text            string `json:"Text" binding:"required"`
	LinkedCommentId *int   `json:"LinkedCommentId,omitempty"`
}

type CommentDeleteDTO struct {
	CommentId int `json:"CommentId" binding:"required"`
	PostId    int `json:"PostId" binding:"required"`
}
