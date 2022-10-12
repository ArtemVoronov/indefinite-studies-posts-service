package tags

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

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

	log.Info(fmt.Sprintf("Created tag. Id: %v", tagId))

	// TODO: send tag to feed builder

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

	log.Info(fmt.Sprintf("Updated tag. Id: %v. New name: %v", dto.Id, dto.Name))

	// TODO: send tag to feed builder

	c.JSON(http.StatusOK, api.DONE)
}

func AssignTag(c *gin.Context) {
	// TODO: store link in appropriate shard: post id <-> tag id
	// TODO: log info
	// TODO: send tag to feed builder
	c.JSON(http.StatusNotImplemented, "NOT IMPLEMENTED")
}

func RemoveTag(c *gin.Context) {
	// TODO: store link in appropriate shard: post id <-> tag id
	// TODO: log info
	// TODO: send tag to feed builder
	c.JSON(http.StatusNotImplemented, "NOT IMPLEMENTED")
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
