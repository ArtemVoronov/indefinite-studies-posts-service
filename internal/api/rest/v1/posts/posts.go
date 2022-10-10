package posts

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetPosts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	idsStr := c.DefaultQuery("ids", "")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	var postsList []entities.Post
	if idsStr != "" {
		ids, castErr := convertIdsQueryParam(idsStr)
		if castErr != nil {
			c.JSON(http.StatusInternalServerError, "Unable to get posts by ids")
			log.Error("Error during casting 'ids' query param", castErr.Error())
			return
		}
		postsList, err = services.Instance().Posts().GetPostsByIds(ids, offset, limit)
	} else {
		postsList, err = services.Instance().Posts().GetPosts(offset, limit)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to get posts")
		log.Error("Unable to get posts", err.Error())
		return
	}

	result := &PostListDTO{
		Data:   convertPosts(postsList),
		Count:  len(postsList),
		Offset: offset,
		Limit:  limit,
	}

	c.JSON(http.StatusOK, result)
}

func GetPost(c *gin.Context) {
	postUuid := c.Param("uuid")

	if postUuid == "" {
		c.JSON(http.StatusBadRequest, "Missed 'uuid' param")
		return
	}

	post, err := services.Instance().Posts().GetPost(postUuid)
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
	var postDTO PostCreateDTO

	if err := c.ShouldBindJSON(&postDTO); err != nil {
		validation.SendError(c, err)
		return
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Error("unable to create uuid for post", err.Error())
		return
	}
	params := toCreatePostParams(&postDTO)
	params.Uuid = uuid.String()

	postId, err := services.Instance().Posts().CreatePost(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Error("Unable to create post", err.Error())
		return
	}

	log.Info(fmt.Sprintf("Created post. Id: %v. Uuid: %v", postId, uuid.String()))

	post, err := services.Instance().Posts().GetPost(uuid.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Error("Unable to get post after creation", err.Error())
		return
	}

	err = services.Instance().Feed().CreatePost(toFeedPostDTO(&post))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Error("Unable to create post at feed service", err.Error())
		return
	}

	c.JSON(http.StatusCreated, uuid.String())
}

func UpdatePost(c *gin.Context) {
	var postDTO PostEditDTO
	if err := c.ShouldBindJSON(&postDTO); err != nil {
		validation.SendError(c, err)
		return
	}

	if postDTO.State != nil {
		if *postDTO.State == entities.POST_STATE_DELETED {
			c.JSON(http.StatusBadRequest, api.DELETE_VIA_PUT_REQUEST_IS_FODBIDDEN)
			return
		}

		possibleStates := entities.GetPossiblePostStates()
		if !utils.Contains(possibleStates, *postDTO.State) {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update post. Wrong 'State' value. Possible values: %v", possibleStates))
			return
		}
	}

	err := services.Instance().Posts().UpdatePost(toUpdatePostParams(&postDTO))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update post")
			log.Error("Unable to update post", err.Error())
		}
		return
	}

	post, err := services.Instance().Posts().GetPost(postDTO.Uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update post")
		log.Error("Unable to get post after updating", err.Error())
		return
	}

	err = services.Instance().Feed().UpdatePost(toFeedPostDTO(&post))
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

	err := services.Instance().Posts().DeletePost(post.Uuid)

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

	c.JSON(http.StatusOK, api.DONE)
}

func convertPosts(input []entities.Post) []PostDTO {
	if input == nil {
		return make([]PostDTO, 0)
	}
	var result []PostDTO
	for _, p := range input {
		result = append(result, convertPost(p))
	}
	return result
}

func convertPost(input entities.Post) PostDTO {
	return PostDTO{Uuid: input.Uuid, Text: input.Text, PreviewText: input.PreviewText, Topic: input.Topic, AuthorId: input.AuthorId, State: input.State}
}

func toFeedPostDTO(post *entities.Post) *feed.FeedPostDTO {
	return &feed.FeedPostDTO{
		Uuid:           post.Uuid,
		AuthorId:       int32(post.AuthorId),
		Text:           post.Text,
		PreviewText:    post.PreviewText,
		Topic:          post.Topic,
		State:          post.State,
		CreateDate:     post.CreateDate,
		LastUpdateDate: post.LastUpdateDate,
	}
}

func toUpdatePostParams(post *PostEditDTO) *queries.UpdatePostParams {
	return &queries.UpdatePostParams{
		Uuid:        post.Uuid,
		AuthorId:    post.AuthorId,
		Text:        post.Text,
		PreviewText: post.PreviewText,
		Topic:       post.Topic,
		State:       post.State,
	}
}

func toCreatePostParams(post *PostCreateDTO) *queries.CreatePostParams {
	return &queries.CreatePostParams{
		AuthorId:    post.AuthorId,
		Text:        post.Text,
		PreviewText: post.PreviewText,
		Topic:       post.Topic,
	}
}

func convertIdsQueryParam(idsQueryParam string) ([]int, error) {
	re := regexp.MustCompile(",")
	tokens := re.Split(idsQueryParam, -1)
	result := make([]int, 0, len(tokens))
	for _, token := range tokens {
		postId, parseErr := strconv.Atoi(token)
		if parseErr != nil {
			return nil, parseErr
		}
		result = append(result, postId)
	}
	return result, nil
}
