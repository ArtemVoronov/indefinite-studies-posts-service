package tags

import (
	"database/sql"
	"encoding/json"
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

const AssignedTagsToPostsTopic = "assigned_tags_to_posts"
const DeletededTagsToPostsTopic = "deleted_tags_to_posts"

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
		Data:   ConvertTags(list),
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

	c.JSON(http.StatusOK, ConvertTag(tag))
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

	c.JSON(http.StatusOK, api.DONE)
}

func AssignTags(c *gin.Context) {
	var dto PostTagConnectionDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	err := services.Instance().Posts().AssignTagsToPost(dto.PostUuid, dto.TagIds)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to assign tag to post")
			log.Error("Unable to assign tag to post", err.Error())
		}
		return
	}

	log.Info(fmt.Sprintf("Assigned tags to post. TagIds: %v. Post UUID: %v", dto.TagIds, dto.PostUuid))

	postJSON, err := json.Marshal(entities.PostWithTagsForQueue{PostUuid: dto.PostUuid, TagIds: dto.TagIds})
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to assign tags to post")
		log.Error(fmt.Sprintf("Unable to convert post with uuid %v to JSON", dto.PostUuid), err.Error())
		return
	}

	err = services.Instance().KafkaProducer().CreateMessage(AssignedTagsToPostsTopic, string(postJSON))
	if err != nil {
		// TODO: create some daemon that catch unpublished posts
		log.Error(fmt.Sprintf("Unable to put post uuid %v into queue %v", dto.PostUuid, AssignedTagsToPostsTopic), err.Error())
	}

	c.JSON(http.StatusOK, api.DONE)
}

func RemoveTags(c *gin.Context) {
	var dto PostTagConnectionDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	err := services.Instance().Posts().RemoveTagsFromPost(dto.PostUuid, dto.TagIds)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to remove tag from post")
			log.Error("Unable to remove tag from post", err.Error())
		}
		return
	}

	log.Info(fmt.Sprintf("Removed tags from post. TagIds: %v. Post UUID: %v", dto.TagIds, dto.PostUuid))

	postJSON, err := json.Marshal(entities.PostWithTagsForQueue{PostUuid: dto.PostUuid, TagIds: dto.TagIds})
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to remove tag from post")
		log.Error(fmt.Sprintf("Unable to convert post with uuid %v to JSON", dto.PostUuid), err.Error())
		return
	}

	err = services.Instance().KafkaProducer().CreateMessage(DeletededTagsToPostsTopic, string(postJSON))
	if err != nil {
		// TODO: create some daemon that catch unpublished posts
		log.Error(fmt.Sprintf("Unable to put post uuid %v into queue %v", dto.PostUuid, DeletededTagsToPostsTopic), err.Error())
	}

	c.JSON(http.StatusOK, api.DONE)
}

func ConvertTags(input []entities.Tag) []TagDTO {
	if input == nil {
		return []TagDTO{}
	}
	var result []TagDTO
	for _, p := range input {
		result = append(result, ConvertTag(p))
	}
	return result
}

func ConvertTag(input entities.Tag) TagDTO {
	return TagDTO{Id: input.Id, Name: input.Name}
}
