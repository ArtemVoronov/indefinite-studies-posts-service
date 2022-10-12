package posts

type PostDTO struct {
	Uuid        string
	AuthorId    int
	Text        string
	PreviewText string
	Topic       string
	State       string
	Tags        []string
}

type PostListDTO struct {
	Count       int
	Offset      int
	Limit       int
	ShardsCount int
	Data        []PostDTO
}

type PostEditDTO struct {
	Uuid        string  `json:"Uuid" binding:"required"`
	AuthorId    *int    `json:"AuthorId,omitempty"`
	Text        *string `json:"Text,omitempty"`
	PreviewText *string `json:"PreviewText,omitempty"`
	Topic       *string `json:"Topic,omitempty"`
	State       *string `json:"State,omitempty"`
}

type PostCreateDTO struct {
	AuthorId    int    `json:"authorId" binding:"required"`
	Text        string `json:"text" binding:"required"`
	PreviewText string `json:"previewText" binding:"required"`
	Topic       string `json:"topic" binding:"required"`
}

type PostDeleteDTO struct {
	Uuid string `json:"Uuid" binding:"required"`
}
