package comments

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
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
)

type CommentDTO struct {
	Id              int
	AuthorId        int
	PostId          int
	LinkedCommentId *int
	Text            string
	State           string
}

type CommentListDTO struct {
	Count  int
	Offset int
	Limit  int
	Data   []CommentDTO
}

type CommentEditDTO struct {
	Id    int     `json:"Id" binding:"required"`
	Text  *string `json:"Text,omitempty"`
	State *string `json:"State,omitempty"`
}

type CommentCreateDTO struct {
	AuthorId        int    `json:"AuthorId" binding:"required"`
	PostId          int    `json:"PostId" binding:"required"`
	Text            string `json:"Text" binding:"required"`
	LinkedCommentId *int   `json:"LinkedCommentId,omitempty"`
}

type CommentDeleteDTO struct {
	Id     int `json:"Id" binding:"required"`
	PostId int `json:"PostId" binding:"required"`
}

func GetComments(c *gin.Context) {
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
		comments, err := queries.GetComments(tx, ctx, postId, limit, offset)
		return comments, err
	})()

	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to get comments")
		log.Printf("Unable to get to comments : %s", err)
		return
	}

	comments, ok := data.([]entities.Comment)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to get comments")
		log.Printf("Unable to get to comments : %s", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	result := &CommentListDTO{Data: convertComments(comments), Count: len(comments), Offset: offset, Limit: limit}
	c.JSON(http.StatusOK, result)
}

func CreateComment(c *gin.Context) {
	var commentDTO CommentCreateDTO

	if err := c.ShouldBindJSON(&commentDTO); err != nil {
		validation.SendError(c, err)
		return
	}

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		result, err := queries.CreateComment(tx, ctx, toCreateCommentParams(&commentDTO))
		return result, err
	})()

	if err != nil || data == -1 {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Printf("Unable to create comment : %s", err)
		return
	}

	commentId, ok := data.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Printf("Unable to cast to 'int' : %s", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	data, err = services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		post, err := queries.GetComment(tx, ctx, commentId)
		return post, err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update comment")
			log.Printf("Unable to update comment: %s", err)
		}
		return
	}

	comment, ok := data.(entities.Comment)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to update comment")
		log.Printf("Unable to cast to 'entities.Comment': %s", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	errFeed := services.Instance().Feed().UpdateComment(toFeedCommentDTO(&comment))
	if errFeed != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update comment")
		log.Printf("Unable to update comment: %s", errFeed)
		return
	}

	c.JSON(http.StatusCreated, data)
}

func UpdateComment(c *gin.Context) {
	var commentDTO CommentEditDTO
	if err := c.ShouldBindJSON(&commentDTO); err != nil {
		validation.SendError(c, err)
		return
	}

	if commentDTO.State != nil {
		if *commentDTO.State == entities.COMMENT_STATE_DELETED {
			c.JSON(http.StatusBadRequest, api.DELETE_VIA_PUT_REQUEST_IS_FODBIDDEN)
			return
		}

		possibleStates := entities.GetPossibleCommentStates()
		if !utils.Contains(possibleStates, *commentDTO.State) {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update comment. Wrong 'State' value. Possible values: %v", possibleStates))
			return
		}
	}

	err := services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.UpdateComment(tx, ctx, toUpdateCommentParams(&commentDTO))
		return err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update comment")
			log.Printf("Unable to update comment : %s", err)
		}
		return
	}

	data, err := services.Instance().DB().Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		post, err := queries.GetComment(tx, ctx, commentDTO.Id)
		return post, err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update comment")
			log.Printf("Unable to update comment: %s", err)
		}
		return
	}

	comment, ok := data.(entities.Comment)
	if !ok {
		c.JSON(http.StatusInternalServerError, "Unable to update comment")
		log.Printf("Unable to cast to 'entities.Comment': %s", api.ERROR_ASSERT_RESULT_TYPE)
		return
	}

	errFeed := services.Instance().Feed().UpdateComment(toFeedCommentDTO(&comment))
	if errFeed != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update comment")
		log.Printf("Unable to update comment: %s", errFeed)
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func DeleteComment(c *gin.Context) {
	var commentDTO CommentDeleteDTO
	if err := c.ShouldBindJSON(&commentDTO); err != nil {
		validation.SendError(c, err)
		return
	}

	err := services.Instance().DB().TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeleteComment(tx, ctx, commentDTO.Id)
		return err
	})()

	if err != nil {
		if err == sql.ErrNoRows {
			errFeed := services.Instance().Feed().DeleteComment(int32(commentDTO.Id))
			if errFeed != nil {
				c.JSON(http.StatusInternalServerError, "Unable to delete post")
				log.Printf("Unable to delete post: %s", errFeed)
				return
			}
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to delete comment")
			log.Printf("Unable to delete comment: %s", err)
		}
		return
	}

	errFeed := services.Instance().Feed().DeleteComment(int32(commentDTO.Id))
	if errFeed != nil {
		c.JSON(http.StatusInternalServerError, "Unable to delete post")
		log.Printf("Unable to delete post: %s", errFeed)
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func convertComments(comments []entities.Comment) []CommentDTO {
	if comments == nil {
		return make([]CommentDTO, 0)
	}
	var result []CommentDTO
	for _, comment := range comments {
		result = append(result, convertComment(comment))
	}
	return result
}

func convertComment(comment entities.Comment) CommentDTO {
	return CommentDTO{Id: comment.Id, AuthorId: comment.AuthorId, PostId: comment.PostId, LinkedCommentId: comment.LinkedCommentId, Text: comment.Text, State: comment.State}
}

func toUpdateCommentParams(comment *CommentEditDTO) *queries.UpdateCommentParams {
	return &queries.UpdateCommentParams{
		Id:    comment.Id,
		Text:  comment.Text,
		State: comment.State,
	}
}

func toCreateCommentParams(comment *CommentCreateDTO) *queries.CreateCommentParams {
	return &queries.CreateCommentParams{
		AuthorId:        comment.AuthorId,
		PostId:          comment.PostId,
		LinkedCommentId: comment.LinkedCommentId,
		Text:            comment.Text,
	}
}

func toFeedCommentDTO(comment *entities.Comment) *feed.FeedCommentDTO {
	result := &feed.FeedCommentDTO{
		Id:             int32(comment.Id),
		AuthorId:       int32(comment.AuthorId),
		PostId:         int32(comment.PostId),
		Text:           comment.Text,
		State:          comment.State,
		CreateDate:     comment.CreateDate,
		LastUpdateDate: comment.LastUpdateDate,
	}

	if comment.LinkedCommentId != nil {
		result.LinkedCommentId = int32(*comment.LinkedCommentId)
	}
	return result
}
