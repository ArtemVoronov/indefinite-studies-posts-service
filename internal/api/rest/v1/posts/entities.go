package posts

type PostDTO struct {
	Id          int
	AuthorId    int
	Text        string
	PreviewText string
	Topic       string
	State       string
}

type PostListDTO struct {
	Count  int
	Offset int
	Limit  int
	Data   []PostDTO
}

type PostEditDTO struct {
	Id          *int    `json:"Id" binding:"required"`
	AuthorId    *int    `json:"AuthorId,omitempty"`
	Text        *string `json:"Text,omitempty"`
	PreviewText *string `json:"PreviewText,omitempty"`
	Topic       *string `json:"Topic,omitempty"`
	State       *string `json:"State,omitempty"`
}

type PostCreateDTO struct {
	AuthorId    int    `json:"authorId" binding:"required"`
	Text        string `json:"text" binding:"required"`
	PreviewText string `json:"PreviewText" binding:"required"`
	Topic       string `json:"topic" binding:"required"`
}

type PostDeleteDTO struct {
	Id int `json:"Id" binding:"required"`
}
