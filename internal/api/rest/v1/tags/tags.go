package tags

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/posts"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/gin-gonic/gin"
)

func GetTags(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	var list []entities.Tag
	list, err = services.Instance().Posts().GetTags(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to get tags")
		log.Error("Unable to get tags", err.Error())
		return
	}

	result := &TagListDTO{
		Data:   convertTags(list),
		Count:  len(list),
		Offset: offset,
		Limit:  limit,
	}

	c.JSON(http.StatusOK, result)
}

func GetTag(c *gin.Context) {
	tagIdStr := c.Param("id")

	if tagIdStr == "" {
		c.JSON(http.StatusBadRequest, "Missed 'id' query param")
		return
	}

	var tagId int
	var parseErr error
	if tagId, parseErr = strconv.Atoi(tagIdStr); parseErr != nil {
		c.JSON(http.StatusBadRequest, api.ERROR_ID_WRONG_FORMAT)
		return
	}

	tag, err := services.Instance().Posts().GetTag(tagId)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to get tag")
			log.Error("Unable to get tag", err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, convertTag(tag))
}

func CreateTag(c *gin.Context) {
	var dto TagCreateDTO

	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	tagId, err := services.Instance().Posts().CreateTag(dto.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create tag")
		log.Error("Unable to create tag", err.Error())
		return
	}

	err = services.Instance().Feed().UpdateTag(posts.ToFeedTagDTO(tagId, dto.Name))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update tag")
		log.Error("Unable to update tag at feed service", err.Error())
		return
	}

	log.Info(fmt.Sprintf("Created tag. Id: %v", tagId))

	c.JSON(http.StatusCreated, tagId)
}

func UpdateTag(c *gin.Context) {
	var dto TagEditDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	err := services.Instance().Posts().UpdateTag(dto.Id, dto.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update tag")
			log.Error("Unable to update tag", err.Error())
		}
		return
	}

	err = services.Instance().Feed().UpdateTag(posts.ToFeedTagDTO(dto.Id, dto.Name))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update tag")
		log.Error("Unable to update tag at feed service", err.Error())
		return
	}

	log.Info(fmt.Sprintf("Updated tag. Id: %v. New name: %v", dto.Id, dto.Name))

	c.JSON(http.StatusOK, api.DONE)
}

func AssignTag(c *gin.Context) {
	var dto PostTagConnectionDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	err := services.Instance().Posts().AssignTagToPost(dto.PostUuid, dto.TagId)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to assign tag to post")
			log.Error("Unable to assign tag to post", err.Error())
		}
		return
	}

	log.Info(fmt.Sprintf("Assigned tag to post. TagId: %v. Post UUID: %v", dto.TagId, dto.PostUuid))

	post, err := services.Instance().Posts().GetPostWithTags(dto.PostUuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to assign tag to post")
		log.Error("Unable to get post after tag assigning", err.Error())
		return
	}

	err = services.Instance().Feed().UpdatePost(posts.ToFeedPostDTO(&post))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to assign tag to post")
		log.Error("Unable to update post at feed service after tag assigning", err.Error())
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func RemoveTag(c *gin.Context) {
	var dto PostTagConnectionDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	err := services.Instance().Posts().RemoveTagToPost(dto.PostUuid, dto.TagId)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to remove tag from post")
			log.Error("Unable to remove tag from post", err.Error())
		}
		return
	}

	log.Info(fmt.Sprintf("Removed tag from post. TagId: %v. Post UUID: %v", dto.TagId, dto.PostUuid))

	post, err := services.Instance().Posts().GetPostWithTags(dto.PostUuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to remove tag from post")
		log.Error("Unable to get post after tag removing", err.Error())
		return
	}

	err = services.Instance().Feed().UpdatePost(posts.ToFeedPostDTO(&post))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to remove tag from post")
		log.Error("Unable to update post at feed service after tag removing", err.Error())
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func convertTags(input []entities.Tag) []TagDTO {
	if input == nil {
		return make([]TagDTO, 0)
	}
	var result []TagDTO
	for _, p := range input {
		result = append(result, convertTag(p))
	}
	return result
}

func convertTag(input entities.Tag) TagDTO {
	return TagDTO{Id: input.Id, Name: input.Name}
}
