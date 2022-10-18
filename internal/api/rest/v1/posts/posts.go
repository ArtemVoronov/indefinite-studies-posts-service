package posts

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
	utilsEntities "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetPosts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	shardStr := c.Query("shard")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	shard, err := strconv.Atoi(shardStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Missed 'shard' parameter or wrong value")
		log.Error(fmt.Sprintf("Missed 'shard' parameter or wrong value: %v", shardStr), err.Error())
		return
	}

	list, err := services.Instance().Posts().GetPostsWithTags(offset, limit, shard)

	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to get posts")
		log.Error("Unable to get posts", err.Error())
		return
	}

	result := &PostListDTO{
		Data:        convertPosts(list),
		Count:       len(list),
		Offset:      offset,
		Limit:       limit,
		ShardsCount: services.Instance().Posts().ShardsNum,
	}

	c.JSON(http.StatusOK, result)
}

func GetPost(c *gin.Context) {
	postUuid := c.Param("uuid")

	if postUuid == "" {
		c.JSON(http.StatusBadRequest, "Missed 'uuid' param")
		return
	}

	post, err := services.Instance().Posts().GetPostWithTags(postUuid)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to get post")
			log.Error("Unable to get post", err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, convertPost(post))
}

func CreatePost(c *gin.Context) {
	var dto PostCreateDTO

	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Error("unable to create uuid for post", err.Error())
		return
	}

	postUuid := uuid.String()

	postId, err := services.Instance().Posts().CreatePost(postUuid, dto.AuthorUuid, dto.Text, dto.PreviewText, dto.Topic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Error("Unable to create post", err.Error())
		return
	}

	log.Info(fmt.Sprintf("Created post. Id: %v. Uuid: %v", postId, postUuid))

	err = services.Instance().Posts().AssignTagsToPost(postUuid, dto.TagIds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Error("Unable to assign tags to post", err.Error())
		return
	}

	log.Info(fmt.Sprintf("Assigned tags to post. TagIds: %v. Post UUID: %v", dto.TagIds, postUuid))

	post, err := services.Instance().Posts().GetPostWithTags(postUuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Error("Unable to get post after creation", err.Error())
		return
	}

	err = services.Instance().Feed().CreatePost(ToFeedPostDTO(&post))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Error("Unable to create post at feed service", err.Error())
		return
	}

	c.JSON(http.StatusCreated, postUuid)
}

func UpdatePost(c *gin.Context) {
	var dto PostEditDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	if dto.State != nil {
		if *dto.State == utilsEntities.POST_STATE_DELETED {
			c.JSON(http.StatusBadRequest, api.DELETE_VIA_PUT_REQUEST_IS_FODBIDDEN)
			return
		}

		possibleStates := utilsEntities.GetPossiblePostStates()
		if !utils.Contains(possibleStates, *dto.State) {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update post. Wrong 'State' value. Possible values: %v", possibleStates))
			return
		}
	}

	err := services.Instance().Posts().UpdatePost(dto.Uuid, dto.AuthorUuid, dto.Text, dto.PreviewText, dto.Topic, dto.State)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update post")
			log.Error("Unable to update post", err.Error())
		}
		return
	}

	log.Info(fmt.Sprintf("Updated post: %v", dto))

	if dto.TagIds != nil {
		err = services.Instance().Posts().RemoveAllTagsFromPost(dto.Uuid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, "Unable to update post")
			log.Error("Unable to delete all tags from post: "+dto.Uuid, err.Error())
			return
		}
		err = services.Instance().Posts().AssignTagsToPost(dto.Uuid, *dto.TagIds)
		if err != nil {
			c.JSON(http.StatusInternalServerError, "Unable to update post")
			log.Error("Unable to assign tags to the post: "+dto.Uuid, err.Error())
			return
		}
		log.Info(fmt.Sprintf("Assigned tags to post. TagIds: %v. Post UUID: %v", *dto.TagIds, dto.Uuid))
	}

	post, err := services.Instance().Posts().GetPostWithTags(dto.Uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update post")
		log.Error("Unable to get post after updating", err.Error())
		return
	}

	err = services.Instance().Feed().UpdatePost(ToFeedPostDTO(&post))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update post")
		log.Error("Unable to update post at feed service", err.Error())
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func DeletePost(c *gin.Context) {
	var post PostDeleteDTO
	if err := c.ShouldBindJSON(&post); err != nil {
		validation.SendError(c, err)
		return
	}

	err := services.Instance().Posts().RemoveAllTagsFromPost(post.Uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to delete post")
		log.Error("Unable to remove tags from posts", err.Error())
		return
	}

	err = services.Instance().Posts().DeletePost(post.Uuid)

	if err != nil {
		if err == sql.ErrNoRows {
			errFeed := services.Instance().Feed().DeletePost(post.Uuid)
			if errFeed != nil {
				c.JSON(http.StatusInternalServerError, "Unable to delete post")
				log.Error("Unable to delete post at feed service", errFeed.Error())
				return
			}
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to delete post")
			log.Error("Unable to delete post", err.Error())
		}
		return
	}

	err = services.Instance().Feed().DeletePost(post.Uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to delete post")
		log.Error("Unable to delete post at feed service", err.Error())
		return
	}

	log.Info(fmt.Sprintf("Deleted post. Uuid: %v", post.Uuid))

	c.JSON(http.StatusOK, api.DONE)
}

func convertPosts(input []entities.PostWithTags) []PostDTO {
	if input == nil {
		return make([]PostDTO, 0)
	}
	var result []PostDTO
	for _, p := range input {
		result = append(result, convertPost(p))
	}
	return result
}

func convertPost(input entities.PostWithTags) PostDTO {
	return PostDTO{
		Uuid:        input.Uuid,
		Text:        input.Text,
		PreviewText: input.PreviewText,
		Topic:       input.Topic,
		AuthorUuid:  input.AuthorUuid,
		State:       input.State,
		TagIds:      input.TagIds,
	}
}

func ToFeedPostDTO(post *entities.PostWithTags) *feed.FeedPostDTO {
	return &feed.FeedPostDTO{
		Uuid:           post.Uuid,
		AuthorUuid:     post.AuthorUuid,
		Text:           post.Text,
		PreviewText:    post.PreviewText,
		Topic:          post.Topic,
		State:          post.State,
		CreateDate:     post.CreateDate,
		LastUpdateDate: post.LastUpdateDate,
		TagIds:         post.TagIds,
	}
}

func ToFeedTagDTO(tagId int, name string) *feed.FeedTagDTO {
	return &feed.FeedTagDTO{
		Id:   tagId,
		Name: name,
	}
}
