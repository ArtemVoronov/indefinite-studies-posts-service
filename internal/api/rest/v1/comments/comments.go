package comments

import (
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
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
)

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

	comments, err := services.Instance().Posts().GetComments(postId, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to get comments")
		log.Printf("Unable to get comments: %s", err)
		return
	}

	result := &CommentListDTO{
		Data:   convertComments(comments),
		Count:  len(comments),
		Offset: offset,
		Limit:  limit,
	}

	c.JSON(http.StatusOK, result)
}

func CreateComment(c *gin.Context) {
	var commentDTO CommentCreateDTO

	if err := c.ShouldBindJSON(&commentDTO); err != nil {
		validation.SendError(c, err)
		return
	}

	commentId, err := services.Instance().Posts().CreateComment(toCreateCommentParams(&commentDTO))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Printf("Unable to create comment: %s", err)
		return
	}

	comment, err := services.Instance().Posts().GetComment(commentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Printf("Unable to get comment after create: %s", err)
		return
	}

	errFeed := services.Instance().Feed().CreateComment(toFeedCommentDTO(&comment))
	if errFeed != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Printf("Unable to create comment: %s", errFeed)
		return
	}

	c.JSON(http.StatusCreated, commentId)
}

func UpdateComment(c *gin.Context) {
	var commentDTO CommentEditDTO
	if err := c.ShouldBindJSON(&commentDTO); err != nil {
		validation.SendError(c, err)
		return
	}

	if !app.IsSameUser(c, commentDTO.AuthorId) && !app.HasOwnerRole(c) {
		c.JSON(http.StatusForbidden, "Forbidden")
		log.Printf("Forbidden to update comment. Author ID: %v", commentDTO.AuthorId)
		return
	}

	if commentDTO.State != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Printf("Forbidden to update comment state. Author ID: %v", commentDTO.AuthorId)
			return
		}
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

	err := services.Instance().Posts().UpdateComment(toUpdateCommentParams(&commentDTO))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update comment")
			log.Printf("Unable to update comment: %s", err)
		}
		return
	}

	comment, err := services.Instance().Posts().GetComment(commentDTO.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update comment")
		log.Printf("Unable to get comment after update: %s", err)
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

	err := services.Instance().Posts().DeleteComment(commentDTO.CommentId)

	if err != nil {
		if err == sql.ErrNoRows {
			errFeed := services.Instance().Feed().DeleteComment(int32(commentDTO.PostId), int32(commentDTO.CommentId))
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

	errFeed := services.Instance().Feed().DeleteComment(int32(commentDTO.PostId), int32(commentDTO.CommentId))
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
