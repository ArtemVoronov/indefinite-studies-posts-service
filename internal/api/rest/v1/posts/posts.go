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
	getPost(c, false)
}

func GetPostPreview(c *gin.Context) {
	getPost(c, true)
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
		if post.Post.State == utilsEntities.POST_STATE_PUBLISHED {
			// then update preview and post at cache
			postJSON, err := json.Marshal(convertPost(post))
			if err != nil {
				// TODO: create some daemon that catch unpublished posts
				log.Error(fmt.Sprintf("Unable to convert post with uuid '%v' to JSON", post.Post.Uuid), err.Error())
			}

			err = services.PutToCache(buildCacheKey(post.Post.Uuid, false), string(postJSON))
			if err != nil {
				log.Error("Unable to put post into the cache", err.Error())
			}

			postJSON, err = json.Marshal(convertPostPreview(post))
			if err != nil {
				// TODO: create some daemon that catch unpublished posts
				log.Error(fmt.Sprintf("Unable to convert post with uuid '%v' to JSON", post.Post.Uuid), err.Error())
			}

			err = services.PutToCache(buildCacheKey(post.Post.Uuid, true), string(postJSON))
			if err != nil {
				log.Error("Unable to put post into the cache", err.Error())
			}
		}
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

	services.SendMessageToKafkaQueue(DeletedPostsTopic, post.Uuid)

	log.Info(fmt.Sprintf("Deleted post. Uuid: %v", post.Uuid))

	c.JSON(http.StatusOK, api.DONE)
}

func getPost(c *gin.Context, isPreview bool) {
	postUuid := c.Param("uuid")

	if postUuid == "" {
		c.JSON(http.StatusBadRequest, "Missed 'uuid' param")
		return
	}

	cached, err := getPostFromCache(postUuid, isPreview)
	if err != nil {
		log.Error("Unable to read cache", err.Error())
	}
	if cached != nil {
		c.JSON(http.StatusOK, cached)
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

	var convertedPost PostDTO

	if isPreview {
		convertedPost = convertPostPreview(post)
	} else {
		convertedPost = convertPost(post)
	}

	if convertedPost.State == utilsEntities.POST_STATE_PUBLISHED {
		postJSON, err := json.Marshal(convertedPost)
		if err != nil {
			// TODO: create some daemon that catch unpublished posts
			log.Error(fmt.Sprintf("Unable to convert post with uuid '%v' to JSON", post.Post.Uuid), err.Error())
		}

		result := string(postJSON)

		err = services.PutToCache(buildCacheKey(post.Post.Uuid, isPreview), result)
		if err != nil {
			log.Error("Unable to put post into the cache", err.Error())
		}
	}

	c.JSON(http.StatusOK, convertedPost)
}

func IsPostPublished(postUuid string) (bool, error) {
	cached, err := getPostFromCache(postUuid, false)
	if err != nil {
		log.Error("Unable to read cache", err.Error())
	}
	if cached != nil {
		return cached.State == utilsEntities.POST_STATE_PUBLISHED, nil
	}
	post, err := services.Instance().Posts().GetPost(postUuid)
	if err != nil {
		log.Error("Unable to get post", err.Error())
		return false, err
	}
	return post.State == utilsEntities.POST_STATE_PUBLISHED, nil
}

func getPostFromCache(postUuid string, isPreview bool) (*PostDTO, error) {
	cached, err := services.GetFromCache(buildCacheKey(postUuid, isPreview))
	if err != nil {
		return nil, err
	}
	if len(cached) > 0 {
		result, err := toPost(cached)
		if err != nil {
			return nil, err
		} else {
			return result, nil
		}
	}
	return nil, nil
}

func buildCacheKey(postUuid string, isPreview bool) string {
	if isPreview {
		return fmt.Sprintf("post_preview_%v", postUuid)
	}
	return fmt.Sprintf("post_%v", postUuid)
}

func toPost(jsonStr string) (*PostDTO, error) {
	var result *PostDTO
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return result, fmt.Errorf("unable to unmarshal post: %w", err)
	}
	return result, nil
}

func sendPostToKafkaQueue(post entities.PostWithTags, queueTopics ...string) {
	postWithTagsForQueue := entities.PostWithTagsForQueue{
		PostUuid:   post.Post.Uuid,
		AuthorUuid: post.Post.AuthorUuid,
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
		services.SendMessageToKafkaQueue(queueTopic, string(postJSON))
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
