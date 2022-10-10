package comments

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetComments(c *gin.Context) {
	postUuid := c.Param("uuid")

	if postUuid == "" {
		c.JSON(http.StatusBadRequest, "Missed 'uuid' parameter")
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

	comments, err := services.Instance().Posts().GetComments(postUuid, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to get comments")
		log.Error("Unable to get comments", err.Error())
		return
	}

	result := &CommentListDTO{
		Data:   convertComments(comments, postUuid),
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

	uuid, err := uuid.NewRandom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Error("unable to create uuid for comment", err.Error())
		return
	}

	params := toCreateCommentParams(&commentDTO)
	params.Uuid = uuid.String()

	commentId, err := services.Instance().Posts().CreateComment(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Error("Unable to create comment", err.Error())
		return
	}

	log.Info(fmt.Sprintf("Created comment. Id: %v. Uuid: %v", commentId, uuid.String()))

	comment, err := services.Instance().Posts().GetComment(uuid.String(), commentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Error("Unable to get comment after creation", err.Error())
		return
	}

	err = services.Instance().Feed().CreateComment(toFeedCommentDTO(&comment, uuid.String()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Error("Unable to create comment at feed servuce", err.Error())
		return
	}

	c.JSON(http.StatusCreated, params.Uuid)
}

func UpdateComment(c *gin.Context) {
	var commentDTO CommentEditDTO
	if err := c.ShouldBindJSON(&commentDTO); err != nil {
		validation.SendError(c, err)
		return
	}

	if !app.IsSameUser(c, commentDTO.AuthorId) && !app.HasOwnerRole(c) {
		c.JSON(http.StatusForbidden, "Forbidden")
		log.Info(fmt.Sprintf("Forbidden to update comment. Author ID: %v", commentDTO.AuthorId))
		return
	}

	if commentDTO.State != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Info(fmt.Sprintf("Forbidden to update comment state. Author ID: %v", commentDTO.AuthorId))
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
			log.Error("Unable to update comment", err.Error())
		}
		return
	}

	comment, err := services.Instance().Posts().GetComment(commentDTO.PostUuid, commentDTO.CommentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update comment")
		log.Error("Unable to get comment after updating", err.Error())
		return
	}

	err = services.Instance().Feed().UpdateComment(toFeedCommentDTO(&comment, commentDTO.PostUuid))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update comment")
		log.Error("Unable to update comment in feed service", err.Error())
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

	err := services.Instance().Posts().DeleteComment(commentDTO.PostUuid, commentDTO.CommentId)

	if err != nil {
		if err == sql.ErrNoRows {
			errFeed := services.Instance().Feed().DeleteComment(commentDTO.PostUuid, commentDTO.CommentUuid)
			if errFeed != nil {
				c.JSON(http.StatusInternalServerError, "Unable to delete comment")
				log.Error("Unable to delete comment from feed service", errFeed.Error())
				return
			}
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to delete comment")
			log.Error("Unable to delete comment", err.Error())
		}
		return
	}

	err = services.Instance().Feed().DeleteComment(commentDTO.PostUuid, commentDTO.CommentUuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to delete comment")
		log.Error("Unable to delete comment from feed service", err.Error())
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func convertComments(comments []entities.Comment, postUuid string) []CommentDTO {
	if comments == nil {
		return make([]CommentDTO, 0)
	}
	var result []CommentDTO
	for _, comment := range comments {
		result = append(result, convertComment(comment, postUuid))
	}
	return result
}

func convertComment(comment entities.Comment, postUuid string) CommentDTO {
	return CommentDTO{Id: comment.Id, AuthorId: comment.AuthorId, PostUuid: postUuid, LinkedCommentId: comment.LinkedCommentId, Text: comment.Text, State: comment.State}
}

func toUpdateCommentParams(comment *CommentEditDTO) *queries.UpdateCommentParams {
	return &queries.UpdateCommentParams{
		Id:    comment.CommentId,
		Text:  comment.Text,
		State: comment.State,
	}
}

func toCreateCommentParams(comment *CommentCreateDTO) *queries.CreateCommentParams {
	return &queries.CreateCommentParams{
		AuthorId:        comment.AuthorId,
		PostId:          comment.PostUuid,
		LinkedCommentId: comment.LinkedCommentId,
		Text:            comment.Text,
	}
}

func toFeedCommentDTO(comment *entities.Comment, postUuid string) *feed.FeedCommentDTO {
	result := &feed.FeedCommentDTO{
		Uuid:           comment.Uuid,
		AuthorId:       int32(comment.AuthorId),
		PostUuid:       postUuid,
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
