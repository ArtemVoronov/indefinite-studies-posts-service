package tags

type TagDTO struct {
	Id   int
	Name string
}

type TagListDTO struct {
	Count  int
	Offset int
	Limit  int
	Data   []TagDTO
}

type TagEditDTO struct {
	Id   int    `json:"Id" binding:"required"`
	Name string `json:"Name" binding:"required"`
}

type TagCreateDTO struct {
	Name string `json:"Name" binding:"required"`
}

type TagDeleteDTO struct {
	Id int `json:"Id" binding:"required"`
}

type PostTagConnectionDTO struct {
	PostUuid string `json:"PostUuid" binding:"required"`
	TagId    int    `json:"TagId" binding:"required"`
}
