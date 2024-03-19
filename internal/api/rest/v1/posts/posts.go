package posts

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/tags"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api/validation"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	utilsEntities "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const NewPostsTopic = "new_posts"
const UpdatedPostsStatesTopic = "updated_posts_states"
const UpdatedPostsTagsTopic = "updated_posts_tags"
const DeletedPostsTopic = "deleted_posts"

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

func GetPostPreview(c *gin.Context) {
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
			c.JSON(http.StatusInternalServerError, "Unable to get post preview")
			log.Error("Unable to get post preview", err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, convertPostPreview(post))
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

	sendPostToKafkaQueue(post, NewPostsTopic)

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
		if err != nil && err != sql.ErrNoRows {
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
		log.Error("Unable to get post after updating "+dto.Uuid, err.Error())
		return
	}

	queueTopicsToNotify := make([]string, 0, 2)

	if dto.State != nil {
		queueTopicsToNotify = append(queueTopicsToNotify, UpdatedPostsStatesTopic)
	}

	if dto.TagIds != nil {
		queueTopicsToNotify = append(queueTopicsToNotify, UpdatedPostsTagsTopic)
	}

	if len(queueTopicsToNotify) != 0 {
		sendPostToKafkaQueue(post, queueTopicsToNotify...)
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
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, "Unable to delete post")
		log.Error("Unable to remove tags from posts", err.Error())
		return
	}

	err = services.Instance().Posts().DeletePost(post.Uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, api.PAGE_NOT_FOUND)
		} else {
			c.JSON(http.StatusInternalServerError, "Unable to delete post")
			log.Error("Unable to delete post", err.Error())
		}
		return
	}

	sendMessageToKafkaQueue(DeletedPostsTopic, post.Uuid)

	log.Info(fmt.Sprintf("Deleted post. Uuid: %v", post.Uuid))

	c.JSON(http.StatusOK, api.DONE)
}

func sendPostToKafkaQueue(post entities.PostWithTags, queueTopics ...string) {
	postWithTagsForQueue := entities.PostWithTagsForQueue{
		PostUuid:   post.Post.Uuid,
		CreateDate: post.Post.CreateDate,
		State:      post.Post.State,
		TagIds:     post.TagIds,
	}
	postJSON, err := json.Marshal(postWithTagsForQueue)
	if err != nil {
		// TODO: create some daemon that catch unpublished posts
		log.Error(fmt.Sprintf("Unable to convert post with uuid '%v' to JSON", post.Post.Uuid), err.Error())
	}
	for _, queueTopic := range queueTopics {
		sendMessageToKafkaQueue(queueTopic, string(postJSON))
	}
}

func sendMessageToKafkaQueue(queueTopic string, message string) {
	err := services.Instance().KafkaProducer().CreateMessage(queueTopic, message)
	if err != nil {
		// TODO: create some daemon that catch unpublished posts
		log.Error(fmt.Sprintf("Unable to put message '%v' into queue '%v'", message, queueTopic), err.Error())
	}
}

func convertPost(input entities.PostWithTags) PostDTO {
	return PostDTO{
		Uuid:        input.Post.Uuid,
		Text:        input.Post.Text,
		PreviewText: input.Post.PreviewText,
		Topic:       input.Post.Topic,
		AuthorUuid:  input.Post.AuthorUuid,
		State:       input.Post.State,
		Tags:        tags.ConvertTags(input.Tags),
		CreateDate:  input.Post.CreateDate,
	}
}

func convertPostPreview(input entities.PostWithTags) PostDTO {
	return PostDTO{
		Uuid:        input.Post.Uuid,
		Text:        "",
		PreviewText: input.Post.PreviewText,
		Topic:       input.Post.Topic,
		AuthorUuid:  input.Post.AuthorUuid,
		State:       input.Post.State,
		Tags:        tags.ConvertTags(input.Tags),
		CreateDate:  input.Post.CreateDate,
	}
}
