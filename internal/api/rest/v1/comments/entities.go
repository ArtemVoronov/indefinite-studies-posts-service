package comments

type CommentDTO struct {
	Id              int
	Uuid            string
	AuthorId        int
	PostUuid        string
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
	CommentId   int     `json:"CommentId" binding:"required"`
	CommentUuid string  `json:"CommentUuid" binding:"required"`
	PostUuid    string  `json:"PostUuid" binding:"required"`
	Text        *string `json:"Text,omitempty"`
	State       *string `json:"State,omitempty"`
	AuthorId    int     `json:"AuthorId" binding:"required"`
}

type CommentCreateDTO struct {
	AuthorId        int    `json:"AuthorId" binding:"required"`
	PostUuid        string `json:"PostUuid" binding:"required"`
	Text            string `json:"Text" binding:"required"`
	LinkedCommentId *int   `json:"LinkedCommentId,omitempty"`
}

type CommentDeleteDTO struct {
	CommentId   int    `json:"CommentId" binding:"required"`
	CommentUuid string `json:"CommentUuid" binding:"required"`
	PostUuid    string `json:"PostId" binding:"required"`
}
