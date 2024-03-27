package posts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/shard"
)

type PostsService struct {
	clientPostsShards []*db.PostgreSQLService
	clientTagsShard   *db.PostgreSQLService
	ShardsNum         int
	shardService      *shard.ShardService
}

func CreatePostsService(clientPostsShards []*db.PostgreSQLService, clientTagsShard *db.PostgreSQLService) *PostsService {
	return &PostsService{
		clientPostsShards: clientPostsShards,
		clientTagsShard:   clientTagsShard,
		ShardsNum:         len(clientPostsShards),
		shardService:      shard.CreateShardService(len(clientPostsShards)),
	}
}

func (s *PostsService) Shutdown() error {
	result := []error{}
	l := len(s.clientPostsShards)
	for i := 0; i < l; i++ {
		err := s.clientPostsShards[i].Shutdown()
		if err != nil {
			result = append(result, err)
		}
	}
	if len(result) > 0 {
		return errors.Join(result...)
	}
	return nil
}

func (s *PostsService) getClientPostsShard(postUuid string) *db.PostgreSQLService {
	bucketIndex := s.shardService.GetBucketIndex(postUuid)
	bucket := s.shardService.GetBucketByIndex(bucketIndex)
	log.Info(fmt.Sprintf("bucket: %v\tbucketIndex: %v", bucket, bucketIndex))
	return s.clientPostsShards[bucket]
}

func (s *PostsService) CreatePost(postUuid string, authorUuid string, text string, previewText string, topic string) (int, error) {
	var postId int = -1
	data, err := s.getClientPostsShard(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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
	return s.getClientPostsShard(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
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
	return s.getClientPostsShard(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeletePost(tx, ctx, postUuid)
		return err
	})()
}

func (s *PostsService) GetPost(postUuid string) (entities.Post, error) {
	var result entities.Post

	data, err := s.getClientPostsShard(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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
	var postWithTagIds entities.PostWithTagIds
	var tags []entities.Tag

	data, err := s.getClientPostsShard(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		post, err := queries.GetPostWithTagIds(tx, ctx, postUuid)
		return post, err
	})()

	if err != nil {
		return result, err
	}

	postWithTagIds, ok := data.(entities.PostWithTagIds)
	if !ok {
		return result, fmt.Errorf("unable to convert data into entities.PostWithTagIds")
	}

	data, err = s.clientTagsShard.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		post, err := queries.GetTagsByIds(tx, ctx, postWithTagIds.TagIds)
		return post, err
	})()

	if err != nil {
		return result, err
	}

	tags, ok = data.([]entities.Tag)
	if !ok {
		return result, fmt.Errorf("unable to convert data into []entities.Tag")
	}

	return entities.PostWithTags{Post: postWithTagIds.Post, Tags: tags, TagIds: postWithTagIds.TagIds}, nil
}

func (s *PostsService) GetPosts(offset int, limit int, shard int) ([]entities.Post, error) {
	if shard > s.ShardsNum || shard < 0 {
		return nil, fmt.Errorf("unexpected shard number: %v", shard)
	}
	data, err := s.clientPostsShards[shard].Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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

func (s *PostsService) CreateComment(postUuid string, authorUuid string, text string, linkedCommentId *int) (int, error) {
	var commentId int = -1
	data, err := s.getClientPostsShard(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		params := &queries.CreateCommentParams{
			AuthorUuid:      authorUuid,
			PostUuid:        postUuid,
			LinkedCommentId: linkedCommentId,
			Text:            text,
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
	return s.getClientPostsShard(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
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
	return s.getClientPostsShard(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeleteComment(tx, ctx, commentId)
		return err
	})()
}

func (s *PostsService) GetComment(postUuid string, commentId int) (entities.Comment, error) {
	var result entities.Comment

	data, err := s.getClientPostsShard(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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
	data, err := s.getClientPostsShard(postUuid).Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		comments, err := queries.GetComments(tx, ctx, postUuid, limit, offset)
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
	data, err := s.clientTagsShard.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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

	data, err := s.clientTagsShard.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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
	var result int = -1
	data, err := s.clientTagsShard.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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
	return s.clientTagsShard.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.UpdateTag(tx, ctx, id, name)
		return err
	})()
}

func (s *PostsService) DeleteTag(id int) error {
	return fmt.Errorf("not implemented")
}

func (s *PostsService) AssignTagToPost(postUuid string, tagId int) error {
	return s.getClientPostsShard(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
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
	return s.getClientPostsShard(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		post, err := queries.GetPost(tx, ctx, postUuid)
		if err != nil {
			return err
		}
		return queries.RemoveTagFromPost(tx, ctx, post.Id, tagId)
	})()
}

func (s *PostsService) RemoveAllTagsFromPost(postUuid string) error {
	return s.getClientPostsShard(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		post, err := queries.GetPost(tx, ctx, postUuid)
		if err != nil {
			return err
		}
		return queries.RemoveAllTagsFromPost(tx, ctx, post.Id)
	})()
}

func (s *PostsService) AssignTagsToPost(postUuid string, tagIds []int) error {
	return s.getClientPostsShard(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
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
	return s.getClientPostsShard(postUuid).TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
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
