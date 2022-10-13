package comments

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
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
	var dto CommentCreateDTO

	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Error("unable to create uuid for comment", err.Error())
		return
	}
	commentUuid := uuid.String()

	commentId, err := services.Instance().Posts().CreateComment(dto.PostUuid, commentUuid, dto.AuthorUuid, dto.Text, dto.LinkedCommentUuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Error("Unable to create comment", err.Error())
		return
	}

	log.Info(fmt.Sprintf("Created comment. Id: %v. Uuid: %v", commentId, commentUuid))

	comment, err := services.Instance().Posts().GetComment(dto.PostUuid, commentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Error("Unable to get comment after creation", err.Error())
		return
	}

	err = services.Instance().Feed().CreateComment(toFeedCommentDTO(&comment, dto.PostUuid))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Error("Unable to create comment at feed servuce", err.Error())
		return
	}

	c.JSON(http.StatusCreated, commentUuid)
}

func UpdateComment(c *gin.Context) {
	var dto CommentEditDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	if !app.IsSameUser(c, dto.AuthorUuid) && !app.HasOwnerRole(c) {
		c.JSON(http.StatusForbidden, "Forbidden")
		log.Info(fmt.Sprintf("Forbidden to update comment. Author UUID: %v", dto.AuthorUuid))
		return
	}

	if dto.State != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			log.Info(fmt.Sprintf("Forbidden to update comment state. Author UUID: %v", dto.AuthorUuid))
			return
		}
		if *dto.State == entities.COMMENT_STATE_DELETED {
			c.JSON(http.StatusBadRequest, api.DELETE_VIA_PUT_REQUEST_IS_FODBIDDEN)
			return
		}

		possibleStates := entities.GetPossibleCommentStates()
		if !utils.Contains(possibleStates, *dto.State) {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update comment. Wrong 'State' value. Possible values: %v", possibleStates))
			return
		}
	}

	err := services.Instance().Posts().UpdateComment(dto.PostUuid, dto.CommentId, dto.Text, dto.State)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to update comment")
			log.Error("Unable to update comment", err.Error())
		}
		return
	}

	log.Info(fmt.Sprintf("Updated comment: %v", dto))

	comment, err := services.Instance().Posts().GetComment(dto.PostUuid, dto.CommentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update comment")
		log.Error("Unable to get comment after updating", err.Error())
		return
	}

	err = services.Instance().Feed().UpdateComment(toFeedCommentDTO(&comment, dto.PostUuid))
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to update comment")
		log.Error("Unable to update comment in feed service", err.Error())
		return
	}

	c.JSON(http.StatusOK, api.DONE)
}

func DeleteComment(c *gin.Context) {
	var dto CommentDeleteDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	err := services.Instance().Posts().DeleteComment(dto.PostUuid, dto.CommentId)

	if err != nil {
		if err == sql.ErrNoRows {
			errFeed := services.Instance().Feed().DeleteComment(dto.PostUuid, dto.CommentUuid)
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

	log.Info(fmt.Sprintf("Deleted comment. Post UUID: %v. Comment ID: %v", dto.PostUuid, dto.CommentId))

	err = services.Instance().Feed().DeleteComment(dto.PostUuid, dto.CommentUuid)
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
	return CommentDTO{Id: comment.Id, AuthorUuid: comment.AuthorUuid, PostUuid: postUuid, LinkedCommentUuid: comment.LinkedCommentUuid, Text: comment.Text, State: comment.State}
}

func toFeedCommentDTO(comment *entities.Comment, postUuid string) *feed.FeedCommentDTO {
	result := &feed.FeedCommentDTO{
		Id:                int32(comment.Id),
		Uuid:              comment.Uuid,
		AuthorUuid:        comment.AuthorUuid,
		PostUuid:          postUuid,
		LinkedCommentUuid: comment.LinkedCommentUuid,
		Text:              comment.Text,
		State:             comment.State,
		CreateDate:        comment.CreateDate,
		LastUpdateDate:    comment.LastUpdateDate,
	}
	return result
}
