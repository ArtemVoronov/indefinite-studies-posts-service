package posts

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
)

type PostDTO struct {
	Id       int
	AuthorId int
	Text     string
	Topic    string
	State    string
}

type PostListDTO struct {
	Count  int
	Offset int
	Limit  int
	Data   []PostDTO
}

type PostEditDTO struct {
	Text     string `json:"text" binding:"required"`
	Topic    string `json:"topic" binding:"required"`
	AuthorId int    `json:"authorId" binding:"required"`
	State    string `json:"state" binding:"required"`
}

type PostCreateDTO struct {
	Text     string `json:"text" binding:"required"`
	Topic    string `json:"topic" binding:"required"`
	AuthorId int    `json:"authorId" binding:"required"`
	State    string `json:"state" binding:"required"`
}

func convertPosts(posts []entities.Post) []PostDTO {
	if posts == nil {
		return make([]PostDTO, 0)
	}
	var result []PostDTO
	for _, post := range posts {
		result = append(result, convertPost(post))
	}
	return result
}

func convertPost(post entities.Post) PostDTO {
	return PostDTO{Id: post.Id, Text: post.Text, Topic: post.Topic, AuthorId: post.AuthorId, State: post.State}
}

func GetPosts(c *gin.Context) {
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

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		posts, err := queries.GetPosts(tx, ctx, limit, offset)
		return posts, err
	})()

	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to get posts")
		log.Printf("Unable to get to posts : %s", err)
		return
	}

	posts, ok := data.([]entities.Post)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to get posts")
		log.Printf("Unable to get to posts : %s", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	result := &PostListDTO{Data: convertPosts(posts), Count: len(posts), Offset: offset, Limit: limit}
	c.JSON(http.StatusOK, result)
}

func GetPost(c *gin.Context) {
	postIdStr := c.Param("id")

	if postIdStr == "" {
		c.JSON(http.StatusBadRequest, "Missed ID")
		return
	}

	var postId int
	var parseErr error
	if postId, parseErr = strconv.Atoi(postIdStr); parseErr != nil {
		c.JSON(http.StatusBadRequest, api.ERROR_ID_WRONG_FORMAT)
		return
	}

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		post, err := queries.GetPost(tx, ctx, postId)
		return post, err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to get post")
			log.Printf("Unable to get to post : %s", err)
		}
		return
	}

	post, ok := data.(entities.Post)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to get post")
		log.Printf("Unable to get to post : %s", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	c.JSON(http.StatusOK, convertPost(post))
}

func CreatePost(c *gin.Context) {
	var post PostCreateDTO

	if err := c.ShouldBindJSON(&post); err != nil {
		validation.SendError(c, err)
		return
	}

	possiblePostStates := entities.GetPossiblePostStates()
	if !utils.Contains(possiblePostStates, post.State) {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to create post. Wrong 'State' value. Possible values: %v", possiblePostStates))
		return
	}

	if post.State == entities.POST_STATE_DELETED {
		c.JSON(http.StatusBadRequest, api.DELETE_VIA_POST_REQUEST_IS_FODBIDDEN)
		return
	}

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		result, err := queries.CreatePost(tx, ctx, post.Text, post.Topic, post.AuthorId, post.State)
		return result, err
	})()

	if err != nil || data == -1 {
		c.JSON(http.StatusInternalServerError, "Unable to create post")
		log.Printf("Unable to create post : %s", err)
		return
	}

	c.JSON(http.StatusCreated, data)
}

// TODO: add optional field updating (field is not reqired and missed -> do not update it)
func UpdatePost(c *gin.Context) {
	postIdStr := c.Param("id")

	if postIdStr == "" {
		c.JSON(http.StatusBadRequest, "Missed ID")
		return
	}

	var postId int
	var parseErr error
	if postId, parseErr = strconv.Atoi(postIdStr); parseErr != nil {
		c.JSON(http.StatusBadRequest, api.ERROR_ID_WRONG_FORMAT)
		return
	}

	var post PostEditDTO

	if err := c.ShouldBindJSON(&post); err != nil {
		validation.SendError(c, err)
		return
	}

	if post.State == entities.POST_STATE_DELETED {
		c.JSON(http.StatusBadRequest, api.DELETE_VIA_PUT_REQUEST_IS_FODBIDDEN)
		return
	}

	possiblePostStates := entities.GetPossiblePostStates()
	if !utils.Contains(possiblePostStates, post.State) {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update post. Wrong 'State' value. Possible values: %v", possiblePostStates))
		return
	}

	err := services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.UpdatePost(tx, ctx, postId, post.Text, post.Topic, post.AuthorId, post.State)
		return err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update post")
			log.Printf("Unable to update post : %s", err)
		}
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func DeletePost(c *gin.Context) {
	idStr := c.Param("id")

	if idStr == "" {
		c.JSON(http.StatusBadRequest, "Missed ID")
		return
	}

	var id int
	var parseErr error
	if id, parseErr = strconv.Atoi(idStr); parseErr != nil {
		c.JSON(http.StatusBadRequest, api.ERROR_ID_WRONG_FORMAT)
		return
	}

	err := services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeletePost(tx, ctx, id)
		return err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to delete post")
			log.Printf("Unable to delete post: %s", err)
		}
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}
