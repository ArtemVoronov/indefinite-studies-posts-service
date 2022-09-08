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

func (s *PostsService) DeletePost(postId int) error {
	return s.client.TxVoid(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) error {
		err := queries.DeletePost(tx, ctx, postId)
		return err
	})()
}

func (s *PostsService) GetPost(postId int) (entities.Post, error) {
	var result entities.Post

	data, err := s.client.Tx(func(tx *sql.Tx, ctx context.Context, cancel context.CancelFunc) (any, error) {
		post, err := queries.GetPost(tx, ctx, postId)
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

func (s *PostsService) GetPostsByIds(ids []int) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}
