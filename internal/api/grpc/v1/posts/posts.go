package posts

import (
	"context"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/posts"
	"google.golang.org/grpc"
)

type PostsServiceServer struct {
	posts.UnimplementedPostsServiceServer
}

func RegisterServiceServer(s *grpc.Server) {
	posts.RegisterPostsServiceServer(s, &PostsServiceServer{})
}

func (s *PostsServiceServer) GetPost(ctx context.Context, in *posts.GetPostRequest) (*posts.GetPostReply, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED")
}

func (s *PostsServiceServer) GetPosts(ctx context.Context, in *posts.GetPostsRequest) (*posts.GetPostsReply, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED")
}

func (s *PostsServiceServer) GetPostsStream(stream posts.PostsService_GetPostsStreamServer) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}

func (s *PostsServiceServer) GetComment(ctx context.Context, in *posts.GetCommentRequest) (*posts.GetCommentReply, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED")
}

func (s *PostsServiceServer) GetComments(ctx context.Context, in *posts.GetCommentsRequest) (*posts.GetCommentsReply, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED")
}

func (s *PostsServiceServer) GetCommentsStream(stream posts.PostsService_GetCommentsStreamServer) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}
