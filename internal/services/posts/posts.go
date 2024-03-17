package posts

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/shard"
)

type PostsService struct {
	clientShards []*db.PostgreSQLService
	ShardsNum    int
	shardService *shard.ShardService
}

func CreatePostsService(clients []*db.PostgreSQLService) *PostsService {
	return &PostsService{
		clientShards: clients,
		ShardsNum:    len(clients),
		shardService: shard.CreateShardService(len(clients)),
	}
}

func (s *PostsService) Shutdown() error {
	result := []error{}
	l := len(s.clientShards)
	for i := 0; i < l; i++ {
		err := s.clientShards[i].Shutdown()
		if err != nil {
			result = append(result, err)
		}
	}
	if len(result) > 0 {
		return fmt.Errorf("errors during shutdown: %v", result)
	}
	return nil
}

func (s *PostsService) client(postUuid string) *db.PostgreSQLService {
	bucketIndex := s.shardService.GetBucketIndex(postUuid)
	bucket := s.shardService.GetBucketByIndex(bucketIndex)
	log.Info(fmt.Sprintf("bucket: %v\tbucketIndex: %v", bucket, bucketIndex))
	return s.clientShards[bucket]
}

func (s *PostsService) CreatePost(postUuid string, authorUuid string, text string, previewText string, topic string) (int, error) {
	var postId int = -1
	data, err := s.client(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		params := &queries.CreatePostParams{
			Uuid:        postUuid,
			AuthorUuid:  authorUuid,
			Text:        text,
			PreviewText: previewText,
			Topic:       topic,
		}
		result, err := queries.CreatePost(tx, ctx, params)
		return result, err
	})()

	if err != nil || data == -1 {
		return postId, err
	}

	postId, ok := data.(int)
	if !ok {
		return postId, fmt.Errorf("unable to convert result into int")
	}
	return postId, nil
}

func (s *PostsService) UpdatePost(postUuid string, authorUuid *string, text *string, previewText *string, topic *string, state *string) error {
	return s.client(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		params := &queries.UpdatePostParams{
			Uuid:        postUuid,
			AuthorUuid:  authorUuid,
			Text:        text,
			PreviewText: previewText,
			Topic:       topic,
			State:       state,
		}
		err := queries.UpdatePost(tx, ctx, params)
		return err
	})()
}

func (s *PostsService) DeletePost(postUuid string) error {
	return s.client(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeletePost(tx, ctx, postUuid)
		return err
	})()
}

func (s *PostsService) GetPost(postUuid string) (entities.Post, error) {
	var result entities.Post

	data, err := s.client(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		post, err := queries.GetPost(tx, ctx, postUuid)
		return post, err
	})()
	if err != nil {
		return result, err
	}

	result, ok := data.(entities.Post)
	if !ok {
		return result, fmt.Errorf("unable to convert result into entities.Post")
	}

	return result, nil
}

func (s *PostsService) GetPostWithTags(postUuid string) (entities.PostWithTags, error) {
	var result entities.PostWithTags

	data, err := s.client(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		post, err := queries.GetPostWithTags(tx, ctx, postUuid)
		return post, err
	})()

	if err != nil {
		return result, err
	}

	result, ok := data.(entities.PostWithTags)
	if !ok {
		return result, fmt.Errorf("unable to convert result into entities.PostWithTags")
	}

	return result, nil
}

func (s *PostsService) GetPosts(offset int, limit int, shard int) ([]entities.Post, error) {
	if shard > s.ShardsNum || shard < 0 {
		return nil, fmt.Errorf("unexpected shard number: %v", shard)
	}
	data, err := s.clientShards[shard].Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		posts, err := queries.GetPosts(tx, ctx, limit, offset)
		return posts, err
	})()
	if err != nil {
		return nil, err
	}

	posts, ok := data.([]entities.Post)
	if !ok {
		return nil, fmt.Errorf("unable to convert result into []entities.Post")
	}
	return posts, nil
}

func (s *PostsService) CreateComment(postUuid string, commentUuid string, authorUuid string, text string, linkedCommentUuid string) (int, error) {
	var commentId int = -1
	data, err := s.client(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		post, err := queries.GetPost(tx, ctx, postUuid)
		if err != nil {
			return nil, err
		}
		params := &queries.CreateCommentParams{
			Uuid:              commentUuid,
			AuthorUuid:        authorUuid,
			PostId:            post.Id,
			LinkedCommentUuid: linkedCommentUuid,
			Text:              text,
		}

		result, err := queries.CreateComment(tx, ctx, params)
		return result, err
	})()

	if err != nil || data == -1 {
		return commentId, err
	}

	commentId, ok := data.(int)
	if !ok {
		return commentId, fmt.Errorf("unable to convert result into int")
	}
	return commentId, nil
}

func (s *PostsService) UpdateComment(postUuid string, commentId int, text *string, state *string) error {
	return s.client(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		params := &queries.UpdateCommentParams{
			Id:    commentId,
			Text:  text,
			State: state,
		}
		err := queries.UpdateComment(tx, ctx, params)
		return err
	})()
}

func (s *PostsService) DeleteComment(postUuid string, commentId int) error {
	return s.client(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeleteComment(tx, ctx, commentId)
		return err
	})()
}

func (s *PostsService) GetComment(postUuid string, commentId int) (entities.Comment, error) {
	var result entities.Comment

	data, err := s.client(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		comment, err := queries.GetComment(tx, ctx, commentId)
		return comment, err
	})()
	if err != nil {
		return result, err
	}

	result, ok := data.(entities.Comment)
	if !ok {
		return result, fmt.Errorf("unable to convert result into entities.Comment")
	}

	return result, nil
}

func (s *PostsService) GetComments(postUuid string, offset int, limit int) ([]entities.Comment, error) {
	data, err := s.client(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		post, err := queries.GetPost(tx, ctx, postUuid)
		if err != nil {
			return nil, err
		}

		comments, err := queries.GetComments(tx, ctx, post.Id, limit, offset)
		return comments, err
	})()
	if err != nil {
		return nil, err
	}

	comments, ok := data.([]entities.Comment)
	if !ok {
		return nil, fmt.Errorf("unable to convert result into []entities.Comment")
	}
	return comments, nil
}

func (s *PostsService) GetTags(offset int, limit int) ([]entities.Tag, error) {
	data, err := s.clientShards[0].Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		comments, err := queries.GetTags(tx, ctx, limit, offset)
		return comments, err
	})()
	if err != nil {
		return nil, err
	}

	comments, ok := data.([]entities.Tag)
	if !ok {
		return nil, fmt.Errorf("unable to convert result into []entities.Tag")
	}
	return comments, nil
}

func (s *PostsService) GetTag(id int) (entities.Tag, error) {
	var result entities.Tag

	data, err := s.clientShards[0].Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		comment, err := queries.GetTag(tx, ctx, id)
		return comment, err
	})()
	if err != nil {
		return result, err
	}

	result, ok := data.(entities.Tag)
	if !ok {
		return result, fmt.Errorf("unable to convert result into entities.Tag")
	}

	return result, nil
}

func (s *PostsService) CreateTag(name string) (int, error) {
	idsMap := make(map[int]int)
	for shard := range s.clientShards {
		id, err := s.createTag(name, shard)
		if err != nil {
			if err.Error() != queries.ErrorTagDuplicateKey.Error() {
				return -1, err
			} else {
				log.Info(fmt.Sprintf("Duplicate tag name during create. Shard ID: %v. Name: %v", shard, name))
			}
		}
		idsMap[id] += 1
	}

	if len(idsMap) != 1 {
		return -1, fmt.Errorf("tag was created with different id in some shard")
	}

	for k := range idsMap {
		return k, nil
	}

	return -1, fmt.Errorf("unable to create tags in all shards")
}

func (s *PostsService) createTag(name string, shard int) (int, error) {
	var result int = -1
	data, err := s.clientShards[shard].Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		result, err := queries.CreateTag(tx, ctx, name)
		return result, err
	})()

	if err != nil || data == -1 {
		return result, err
	}

	result, ok := data.(int)
	if !ok {
		return result, fmt.Errorf("unable to convert result into int")
	}
	return result, nil
}

func (s *PostsService) UpdateTag(id int, name string) error {
	result := []error{}
	for shard := range s.clientShards {
		err := s.updateTag(id, name, shard)
		if err != nil {
			if err.Error() != queries.ErrorTagDuplicateKey.Error() {
				result = append(result, err)
			} else {
				log.Info(fmt.Sprintf("Duplicate tag name during update. Shard ID: %v. Name: %v. ID: %v", shard, name, id))
			}
		}
	}

	if len(result) != 0 {
		return fmt.Errorf("unable to update tags in all shards")
	}

	return nil
}

func (s *PostsService) updateTag(id int, name string, shard int) error {
	return s.clientShards[shard].TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.UpdateTag(tx, ctx, id, name)
		return err
	})()
}

func (s *PostsService) DeleteTag(id int) error {
	return fmt.Errorf("not implemented")
}

func (s *PostsService) AssignTagToPost(postUuid string, tagId int) error {
	return s.client(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		post, err := queries.GetPost(tx, ctx, postUuid)
		if err != nil {
			return err
		}
		err = queries.AssignTagToPost(tx, ctx, post.Id, tagId)
		if err != nil && err != queries.ErrorPostTagDuplicateKey {
			return err
		}
		return nil
	})()
}

func (s *PostsService) RemoveTagFromPost(postUuid string, tagId int) error {
	return s.client(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		post, err := queries.GetPost(tx, ctx, postUuid)
		if err != nil {
			return err
		}
		return queries.RemoveTagFromPost(tx, ctx, post.Id, tagId)
	})()
}

func (s *PostsService) RemoveAllTagsFromPost(postUuid string) error {
	return s.client(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		post, err := queries.GetPost(tx, ctx, postUuid)
		if err != nil {
			return err
		}
		return queries.RemoveAllTagsFromPost(tx, ctx, post.Id)
	})()
}

func (s *PostsService) AssignTagsToPost(postUuid string, tagIds []int) error {
	return s.client(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		post, err := queries.GetPost(tx, ctx, postUuid)
		if err != nil {
			return err
		}
		for _, tagId := range tagIds {
			err = queries.AssignTagToPost(tx, ctx, post.Id, tagId)
			if err != nil && err != queries.ErrorPostTagDuplicateKey {
				return err
			}
		}
		return nil
	})()
}

func (s *PostsService) RemoveTagsFromPost(postUuid string, tagIds []int) error {
	return s.client(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		post, err := queries.GetPost(tx, ctx, postUuid)
		if err != nil {
			return err
		}
		for _, tagId := range tagIds {
			err = queries.RemoveTagFromPost(tx, ctx, post.Id, tagId)
			if err != nil {
				return err
			}
		}
		return nil
	})()
}
