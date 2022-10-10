package posts

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/queries"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
)

type PostsService struct {
	client *db.PostgreSQLService
}

func CreatePostsService(client *db.PostgreSQLService) *PostsService {
	return &PostsService{
		client: client,
	}
}

func (s *PostsService) Shutdown() error {
	return s.client.Shutdown()
}

func (s *PostsService) CreatePost(post *queries.CreatePostParams) (int, error) {
	var postId int = -1
	data, err := s.client.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		result, err := queries.CreatePost(tx, ctx, post)
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

func (s *PostsService) UpdatePost(post *queries.UpdatePostParams) error {
	return s.client.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.UpdatePost(tx, ctx, post)
		return err
	})()
}

func (s *PostsService) DeletePost(postUuid string) error {
	return s.client.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeletePost(tx, ctx, postUuid)
		return err
	})()
}

func (s *PostsService) GetPost(postUuid string) (entities.Post, error) {
	var result entities.Post

	data, err := s.client.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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

func (s *PostsService) GetPosts(offset int, limit int) ([]entities.Post, error) {
	// TODO: iterate through all shards
	data, err := s.client.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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

func (s *PostsService) GetPostsByIds(ids []int, offset int, limit int) ([]entities.Post, error) {
	data, err := s.client.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		posts, err := queries.GetPostsByIds(tx, ctx, ids, limit, offset)
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
func (s *PostsService) CreateComment(postUuid string, commentUuid string, AuthorId int, Text string, LinkedCommentId *int) (int, error) {
	// TODO: get shard by post uuid, get post id, convert to params using post id

	var commentId int = -1
	data, err := s.client.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		post, err := queries.GetPost(tx, ctx, postUuid)
		if err != nil {
			return nil, err
		}
		params := &queries.CreateCommentParams{
			Uuid:            commentUuid,
			AuthorId:        AuthorId,
			PostId:          post.Id,
			LinkedCommentId: LinkedCommentId,
			Text:            Text,
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
	// TODO: get shard by post uuid
	return s.client.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
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
	// TODO: get shard by post uuid, delete comment by id (not uuid)
	return s.client.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeleteComment(tx, ctx, commentId)
		return err
	})()
}

func (s *PostsService) GetComment(postUuid string, commentId int) (entities.Comment, error) {
	// TODO: get shard by post uuid, get comment by id (not uuid)
	var result entities.Comment

	data, err := s.client.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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
	// TODO: get shard by post uuid, fund post id, find comments by post id
	data, err := s.client.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
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
