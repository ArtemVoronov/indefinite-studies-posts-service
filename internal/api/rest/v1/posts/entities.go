package posts

type PostDTO struct {
	Uuid        string
	AuthorUuid  string
	Text        string
	PreviewText string
	Topic       string
	State       string
	TagIds      []int
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
	AuthorUuid  *string `json:"AuthorUuid,omitempty"`
	Text        *string `json:"Text,omitempty"`
	PreviewText *string `json:"PreviewText,omitempty"`
	Topic       *string `json:"Topic,omitempty"`
	State       *string `json:"State,omitempty"`
}

type PostCreateDTO struct {
	AuthorUuid  string `json:"AuthorUuid" binding:"required"`
	Text        string `json:"text" binding:"required"`
	PreviewText string `json:"previewText" binding:"required"`
	Topic       string `json:"topic" binding:"required"`
	TagId       int    `json:"tagId" binding:"required"`
}

type PostDeleteDTO struct {
	Uuid string `json:"Uuid" binding:"required"`
}
