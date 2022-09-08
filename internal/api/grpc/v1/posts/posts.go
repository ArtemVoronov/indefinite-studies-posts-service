package posts

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/posts"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PostsServiceServer struct {
	posts.UnimplementedPostsServiceServer
}

func RegisterServiceServer(s *grpc.Server) {
	posts.RegisterPostsServiceServer(s, &PostsServiceServer{})
}

func (s *PostsServiceServer) GetPost(ctx context.Context, in *posts.GetPostRequest) (*posts.GetPostReply, error) {
	post, err := services.Instance().Posts().GetPost(int(in.GetId()))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf(api.PAGE_NOT_FOUND)
		} else {
			return nil, err
		}
	}
	return toGetPostReply(post), nil
}

func (s *PostsServiceServer) GetPosts(ctx context.Context, in *posts.GetPostsRequest) (*posts.GetPostsReply, error) {
	var postsList []entities.Post
	var err error

	if len(in.GetIds()) > 0 {
		postsList, err = services.Instance().Posts().GetPostsByIds(toInt(in.GetIds()), int(in.Offset), int(in.Limit))
	} else {
		postsList, err = services.Instance().Posts().GetPosts(int(in.Offset), int(in.Limit))
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf(api.PAGE_NOT_FOUND)
		} else {
			return nil, err
		}
	}

	result := &posts.GetPostsReply{
		Offset: in.Offset,
		Limit:  in.Limit,
		Count:  int32(len(postsList)),
		Posts:  toGetPostReplies(postsList),
	}

	return result, nil
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

func toGetPostReply(post entities.Post) *posts.GetPostReply {
	return &posts.GetPostReply{
		Id:             int32(post.Id),
		AuthorId:       int32(post.AuthorId),
		Text:           post.Text,
		PreviewText:    post.PreviewText,
		Topic:          post.Topic,
		State:          post.State,
		CreateDate:     timestamppb.New(post.CreateDate),
		LastUpdateDate: timestamppb.New(post.LastUpdateDate),
	}
}

func toGetPostReplies(input []entities.Post) []*posts.GetPostReply {
	replies := []*posts.GetPostReply{}
	for _, p := range input {
		reply := toGetPostReply(p)
		replies = append(replies, reply)
	}
	return replies
}

func toGetCommentReply(comment entities.Comment) *posts.GetCommentReply {
	return &posts.GetCommentReply{
		Id:              int32(comment.Id),
		AuthorId:        int32(comment.AuthorId),
		PostId:          int32(comment.PostId),
		LinkedCommentId: toLinkedCommentId(comment.LinkedCommentId),
		Text:            comment.Text,
		State:           comment.State,
		CreateDate:      timestamppb.New(comment.CreateDate),
		LastUpdateDate:  timestamppb.New(comment.LastUpdateDate),
	}
}

func toLinkedCommentId(val *int) int32 {
	if val == nil {
		return 0
	}
	result := int32(*val)
	return result
}

func toInt(input []int32) []int {
	result := make([]int, len(input))
	for i := range result {
		result[i] = int(input[i])
	}
	return result
}
