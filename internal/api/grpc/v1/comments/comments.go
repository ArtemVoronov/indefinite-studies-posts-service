package comments

import (
	"context"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/posts"
	"google.golang.org/grpc"
)

type CommentsServiceServer struct {
	posts.UnimplementedPostsServiceServer
}

func RegisterServiceServer(s *grpc.Server) {
	posts.RegisterPostsServiceServer(s, &CommentsServiceServer{})
}

func (s *CommentsServiceServer) GetComment(ctx context.Context, in *posts.GetCommentRequest) (*posts.GetCommentReply, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED")
}

func (s *CommentsServiceServer) GetComments(ctx context.Context, in *posts.GetCommentsRequest) (*posts.GetCommentsReply, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED")
}

func (s *CommentsServiceServer) GetCommentsStream(stream posts.PostsService_GetCommentsStreamServer) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}
