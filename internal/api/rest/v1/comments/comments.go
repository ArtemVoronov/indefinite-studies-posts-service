package comments

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/posts"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	utilsEntities "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
)

const NewCommentsTopic = "new_comments"
const UpdatedCommentsStatesTopic = "updated_comments_states"
const DeletedCommentsTopic = "deleted_comments"

// TODO: implement later
// func GetComments(c *gin.Context) {
// 	postUuid := c.Param("uuid")

// 	if postUuid == "" {
// 		c.JSON(http.StatusBadRequest, "Missed 'uuid' parameter")
// 		return
// 	}

// 	limitStr := c.DefaultQuery("limit", "50")
// 	offsetStr := c.DefaultQuery("offset", "0")

// 	limit, err := strconv.Atoi(limitStr)
// 	if err != nil {
// 		limit = 50
// 	}

// 	offset, err := strconv.Atoi(offsetStr)
// 	if err != nil {
// 		offset = 0
// 	}

// 	comments, err := services.Instance().Posts().GetComments(postUuid, offset, limit)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, "Unable to get comments")
// 		log.Error("Unable to get comments", err.Error())
// 		return
// 	}

// 	result := &CommentListDTO{
// 		Data:   convertComments(comments, postUuid),
// 		Count:  len(comments),
// 		Offset: offset,
// 		Limit:  limit,
// 	}

// 	c.JSON(http.StatusOK, result)
// }

func GetComment(c *gin.Context) {
	postUuid := c.Param("uuid")
	commentIdStr := c.Param("id")

	if postUuid == "" {
		c.JSON(http.StatusBadRequest, "Missed 'uuid' parameter")
		return
	}

	if commentIdStr == "" {
		c.JSON(http.StatusBadRequest, "Missed 'id' parameter")
		return
	}

	commentId, err := strconv.Atoi(commentIdStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to parse comment id")
		return
	}

	cached, err := getCommentFromCache(postUuid, commentIdStr)
	if err != nil {
		log.Error("Unable to read cache", err.Error())
	}
	if cached != nil {
		c.JSON(http.StatusOK, cached)
		return
	}

	comment, err := services.Instance().Posts().GetComment(postUuid, commentId)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to get comment")
			log.Error("Unable to get comment", err.Error())
		}
		return
	}

	convertedComment := convertComment(comment)

	commentJSON, err := json.Marshal(convertedComment)
	if err != nil {
		// TODO: create some daemon that catch unpublished posts
		log.Error(fmt.Sprintf("Unable to convert comment with post uuid '%v' and id '%v' to JSON", postUuid, commentIdStr), err.Error())
	}

	result := string(commentJSON)

	if convertedComment.State == utilsEntities.COMMENT_STATE_PUBLISHED {
		err = services.PutToCache(buildCacheKey(postUuid, commentIdStr), result)
		if err != nil {
			log.Error("Unable to put post into the cache", err.Error())
		}
	}

	c.JSON(http.StatusOK, convertedComment)
}

func CreateComment(c *gin.Context) {
	var dto CommentCreateDTO

	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	isPostPublished, err := posts.IsPostPublished(dto.PostUuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to verify post state")
		log.Error("Unable to verify post state", err.Error())
		return
	}

	if !isPostPublished {
		c.JSON(http.StatusBadRequest, "Unable to create post. Post is not published")
		return
	}

	commentId, err := services.Instance().Posts().CreateComment(dto.PostUuid, dto.AuthorUuid, dto.Text, dto.LinkedCommentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Error("Unable to create comment", err.Error())
		return
	}

	log.Info(fmt.Sprintf("Created comment. ID: %v. Post UUID: %v", commentId, dto.PostUuid))

	comment, err := services.Instance().Posts().GetComment(dto.PostUuid, commentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Unable to create comment")
		log.Error("Unable to get comment after creation", err.Error())
		return
	}

	sendCommentToKafkaQueue(comment, NewCommentsTopic)

	c.JSON(http.StatusCreated, commentId)
}

func UpdateComment(c *gin.Context) {
	var dto CommentEditDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		validation.SendError(c, err)
		return
	}

	comment, err := services.Instance().Posts().GetComment(dto.PostUuid, dto.CommentId)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to get comment")
			log.Error("Unable to get comment", err.Error())
		}
		return
	}

	isAllowToUpdateComment := (comment.State == utilsEntities.COMMENT_STATE_NEW && app.IsSameUser(c, comment.AuthorUuid)) || app.HasOwnerRole(c)

	if !isAllowToUpdateComment {
		c.JSON(http.StatusForbidden, "Forbidden")
		userUuidFromCtx, _ := c.Get(app.CTX_TOKEN_ID_KEY)
		log.Info(fmt.Sprintf("Forbidden to update comment. User UUID: %v", userUuidFromCtx))
		return
	}

	if dto.State != nil {
		if !app.HasOwnerRole(c) {
			c.JSON(http.StatusForbidden, "Forbidden")
			userUuidFromCtx, _ := c.Get(app.CTX_TOKEN_ID_KEY)
			log.Info(fmt.Sprintf("Forbidden to update comment state. User UUID: %v", userUuidFromCtx))
			return
		}
		if *dto.State == utilsEntities.COMMENT_STATE_DELETED {
			c.JSON(http.StatusBadRequest, api.DELETE_VIA_PUT_REQUEST_IS_FODBIDDEN)
			return
		}

		possibleStates := utilsEntities.GetPossibleCommentStates()
		if !utils.Contains(possibleStates, *dto.State) {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Unable to update comment. Wrong 'State' value. Possible values: %v", possibleStates))
			return
		}
	}

	err = services.Instance().Posts().UpdateComment(dto.PostUuid, dto.CommentId, dto.Text, dto.State)
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

	if dto.State != nil {
		comment, err := services.Instance().Posts().GetComment(dto.PostUuid, dto.CommentId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, "Unable to create comment")
			log.Error("Unable to get comment after creation", err.Error())
		} else {
			sendCommentToKafkaQueue(comment, UpdatedCommentsStatesTopic)
		}
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
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to delete comment")
			log.Error("Unable to delete comment", err.Error())
		}
		return
	}

	log.Info(fmt.Sprintf("Deleted comment. Post UUID: %v. Comment ID: %v", dto.PostUuid, dto.CommentId))

	sendDeletedCommentToKafkaQueue(dto.PostUuid, dto.CommentId, DeletedCommentsTopic)

	c.JSON(http.StatusOK, api.DONE)
}

func getCommentFromCache(postUuid string, commentId string) (*CommentDTO, error) {
	cached, err := services.GetFromCache(buildCacheKey(postUuid, commentId))
	if err != nil {
		return nil, err
	}
	if len(cached) > 0 {
		result, err := toComment(cached)
		if err != nil {
			return nil, err
		} else {
			return result, nil
		}
	}
	return nil, nil
}

func buildCacheKey(postUuid string, commentId string) string {
	return fmt.Sprintf("post_%v_comment_%v", postUuid, commentId)
}

func sendCommentToKafkaQueue(comment entities.Comment, queueTopics ...string) {
	commentForQueue := entities.CommentForQueue{
		PostUuid:   comment.PostUuid,
		CommentId:  comment.Id,
		CreateDate: comment.CreateDate,
		State:      comment.State,
	}
	commentJSON, err := json.Marshal(commentForQueue)
	if err != nil {
		// TODO: create some daemon that catch unpublished posts
		log.Error(fmt.Sprintf("Unable to convert comment '%v' to JSON", commentForQueue), err.Error())
	}
	for _, queueTopic := range queueTopics {
		services.SendMessageToKafkaQueue(queueTopic, string(commentJSON))
	}
}

func sendDeletedCommentToKafkaQueue(postUuid string, commentId int, queueTopics ...string) {
	commentForQueue := entities.DeletedCommentForQueue{
		PostUuid:  postUuid,
		CommentId: commentId,
	}
	commentJSON, err := json.Marshal(commentForQueue)
	if err != nil {
		// TODO: create some daemon that catch unpublished posts
		log.Error(fmt.Sprintf("Unable to convert comment '%v' to JSON", commentForQueue), err.Error())
	}
	for _, queueTopic := range queueTopics {
		services.SendMessageToKafkaQueue(queueTopic, string(commentJSON))
	}
}

func toComment(jsonStr string) (*CommentDTO, error) {
	var result *CommentDTO
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return result, fmt.Errorf("unable to unmarshal comment: %v", err)
	}
	return result, nil
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
	return CommentDTO{
		Id:              comment.Id,
		AuthorUuid:      comment.AuthorUuid,
		PostUuid:        comment.PostUuid,
		LinkedCommentId: comment.LinkedCommentId,
		Text:            comment.Text,
		State:           comment.State,
	}
}
