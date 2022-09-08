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

func (s *PostsService) CreatePost() error {
	return fmt.Errorf("NOT IMPLEMENTED")
}

func (s *PostsService) UpdatePost() error {
	return fmt.Errorf("NOT IMPLEMENTED")
}

func (s *PostsService) DeletePost() error {
	return fmt.Errorf("NOT IMPLEMENTED")
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

func (s *PostsService) GetPosts(offset int, limit int) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}

func (s *PostsService) GetPostsByIds(ids []int) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}
