package posts

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/api"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/posts"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
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
	post, err := services.Instance().Posts().GetPost(in.GetUuid())
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

	// if len(in.GetIds()) > 0 {
	// 	postsList, err = services.Instance().Posts().GetPostsByIds(utils.Int32SliceToIntSlice(in.GetIds()), int(in.Offset), int(in.Limit))
	// } else {
	postsList, err = services.Instance().Posts().GetPosts(int(in.Offset), int(in.Limit))
	// }

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
	comment, err := services.Instance().Posts().GetComment(in.GetPostUuid(), int(in.GetId()))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf(api.PAGE_NOT_FOUND)
		} else {
			return nil, err
		}
	}
	return toGetCommentReply(comment, in.GetPostUuid()), nil
}

func (s *PostsServiceServer) GetComments(ctx context.Context, in *posts.GetCommentsRequest) (*posts.GetCommentsReply, error) {
	var commentsList []entities.Comment
	var err error

	commentsList, err = services.Instance().Posts().GetComments(in.GetPostUuid(), int(in.Offset), int(in.Limit))

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf(api.PAGE_NOT_FOUND)
		} else {
			return nil, err
		}
	}

	result := &posts.GetCommentsReply{
		Offset:   in.Offset,
		Limit:    in.Limit,
		Count:    int32(len(commentsList)),
		Comments: toGetCommentReplies(commentsList, in.GetPostUuid()),
	}

	return result, nil
}

func (s *PostsServiceServer) GetCommentsStream(stream posts.PostsService_GetCommentsStreamServer) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}

func toGetPostReply(post entities.Post) *posts.GetPostReply {
	return &posts.GetPostReply{
		Uuid:           post.Uuid,
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

func toGetCommentReply(comment entities.Comment, postUuid string) *posts.GetCommentReply {
	return &posts.GetCommentReply{
		Id:              int32(comment.Id),
		Uuid:            comment.Uuid,
		AuthorId:        int32(comment.AuthorId),
		PostUuid:        postUuid,
		LinkedCommentId: utils.IntPtrToInt32(comment.LinkedCommentId),
		Text:            comment.Text,
		State:           comment.State,
		CreateDate:      timestamppb.New(comment.CreateDate),
		LastUpdateDate:  timestamppb.New(comment.LastUpdateDate),
	}
}

func toGetCommentReplies(input []entities.Comment, postUuid string) []*posts.GetCommentReply {
	replies := []*posts.GetCommentReply{}
	for _, p := range input {
		reply := toGetCommentReply(p, postUuid)
		replies = append(replies, reply)
	}
	return replies
}
