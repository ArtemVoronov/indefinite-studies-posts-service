package posts

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
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
			log.Printf("Error during casting 'ids' query param: %s", castErr)
			return
		}
		postsList, err = services.Instance().Posts().GetPostsByIds(ids, offset, limit)
	} else {
		postsList, err = services.Instance().Posts().GetPosts(offset, limit)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to get posts")
		log.Printf("Unable to get posts: %s", err)
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
	postIdStr := c.Param("id")

	if postIdStr == "" {
		c.JSON(http.StatusBadRequest, "Missed 'id' query param")
		return
	}

	var postId int
	var parseErr error
	if postId, parseErr = strconv.Atoi(postIdStr); parseErr != nil {
		c.JSON(http.StatusBadRequest, api.ERROR_ID_WRONG_FORMAT)
		return
	}

	post, err := services.Instance().Posts().GetPost(postId)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to get post")
			log.Printf("Unable to get post: %s", err)
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

	postId, err := services.Instance().Posts().CreatePost(toCreatePostParams(&postDTO))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Printf("Unable to create post: %s", err)
		return
	}

	post, err := services.Instance().Posts().GetPost(postId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Printf("Unable to get post after create: %s", err)
		return
	}

	errFeed := services.Instance().Feed().CreatePost(toFeedPostDTO(&post))
	if errFeed != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Printf("Unable to send post to feed: %s", errFeed)
		return
	}

	c.JSON(http.StatusCreated, postId)
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
			log.Printf("Unable to update post: %s", err)
		}
		return
	}

	post, err := services.Instance().Posts().GetPost(*postDTO.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update post")
		log.Printf("Unable to get post after update: %s", err)
		return
	}

	errFeed := services.Instance().Feed().UpdatePost(toFeedPostDTO(&post))
	if errFeed != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update post")
		log.Printf("Unable to send post to feed: %s", errFeed)
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

	err := services.Instance().Posts().DeletePost(post.Id)

	if err != nil {
		if err == sql.ErrNoRows {
			errFeed := services.Instance().Feed().DeletePost(int32(post.Id))
			if errFeed != nil {
				c.JSON(http.StatusInternalServerError, "Unable to delete post")
				log.Printf("Unable to delete post: %s", errFeed)
				return
			}
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to delete post")
			log.Printf("Unable to delete post: %s", err)
		}
		return
	}

	errFeed := services.Instance().Feed().DeletePost(int32(post.Id))
	if errFeed != nil {
		c.JSON(http.StatusInternalServerError, "Unable to delete post")
		log.Printf("Unable to delete post: %s", errFeed)
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
	return PostDTO{Id: input.Id, Text: input.Text, PreviewText: input.PreviewText, Topic: input.Topic, AuthorId: input.AuthorId, State: input.State}
}

func toFeedPostDTO(post *entities.Post) *feed.FeedPostDTO {
	return &feed.FeedPostDTO{
		Id:             int32(post.Id),
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
		Id:          post.Id,
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
