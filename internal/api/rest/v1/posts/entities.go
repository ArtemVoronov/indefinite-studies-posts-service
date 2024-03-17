package posts

type PostDTO struct {
	Uuid        string
	AuthorUuid  string
	Text        string
	PreviewText string
	Topic       string
	State       string
	Tags        map[int]string
}

// TODO: need to add additional service and remove paramter ShardsCount, UI should not know about shards at all

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
	TagIds      *[]int  `json:"TagIds,omitempty"`
}

type PostCreateDTO struct {
	AuthorUuid  string `json:"AuthorUuid" binding:"required"`
	Text        string `json:"Text" binding:"required"`
	PreviewText string `json:"PreviewText" binding:"required"`
	Topic       string `json:"Topic" binding:"required"`
	TagIds      []int  `json:"TagIds" binding:"required"`
}

type PostDeleteDTO struct {
	Uuid string `json:"Uuid" binding:"required"`
}
