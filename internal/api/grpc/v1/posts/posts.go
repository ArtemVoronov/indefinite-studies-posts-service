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

// TODO: remove unused posts calls
type PostsServiceServer struct {
	posts.UnimplementedPostsServiceServer
}

func RegisterServiceServer(s *grpc.Server) {
	posts.RegisterPostsServiceServer(s, &PostsServiceServer{})
}

func (s *PostsServiceServer) GetPost(ctx context.Context, in *posts.GetPostRequest) (*posts.GetPostReply, error) {
	post, err := services.Instance().Posts().GetPostWithTags(in.GetUuid())
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
	return nil, fmt.Errorf("NOT IMPLEMENTED")
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

	commentsList, err = services.Instance().Posts().GetComments(in.GetPostUuid(), int(in.GetOffset()), int(in.GetLimit()))

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

func (s *PostsServiceServer) GetTag(ctx context.Context, in *posts.GetTagRequest) (*posts.GetTagReply, error) {
	tag, err := services.Instance().Posts().GetTag(int(in.GetId()))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf(api.PAGE_NOT_FOUND)
		} else {
			return nil, err
		}
	}
	return toGetTagReply(tag), nil
}

func (s *PostsServiceServer) GetTags(ctx context.Context, in *posts.GetTagsRequest) (*posts.GetTagsReply, error) {
	var tagsList []entities.Tag
	var err error

	tagsList, err = services.Instance().Posts().GetTags(int(in.GetOffset()), int(in.GetLimit()))

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf(api.PAGE_NOT_FOUND)
		} else {
			return nil, err
		}
	}

	result := &posts.GetTagsReply{
		Offset: in.Offset,
		Limit:  in.Limit,
		Count:  int32(len(tagsList)),
		Tags:   toGetTagReplies(tagsList),
	}

	return result, nil
}

func toGetPostReply(post entities.PostWithTags) *posts.GetPostReply {
	return &posts.GetPostReply{
		Uuid:           post.Post.Uuid,
		AuthorUuid:     post.Post.AuthorUuid,
		Text:           post.Post.Text,
		PreviewText:    post.Post.PreviewText,
		Topic:          post.Post.Topic,
		State:          post.Post.State,
		CreateDate:     timestamppb.New(post.Post.CreateDate),
		LastUpdateDate: timestamppb.New(post.Post.LastUpdateDate),
		TagIds:         utils.ToInt32(post.TagIds),
	}
}

func toGetPostReplies(input []entities.PostWithTags) []*posts.GetPostReply {
	replies := []*posts.GetPostReply{}
	for _, p := range input {
		reply := toGetPostReply(p)
		replies = append(replies, reply)
	}
	return replies
}

func toGetCommentReply(comment entities.Comment, postUuid string) *posts.GetCommentReply {
	return &posts.GetCommentReply{
		Id:                int32(comment.Id),
		Uuid:              comment.Uuid,
		AuthorUuid:        comment.AuthorUuid,
		PostUuid:          postUuid,
		LinkedCommentUuid: comment.LinkedCommentUuid,
		Text:              comment.Text,
		State:             comment.State,
		CreateDate:        timestamppb.New(comment.CreateDate),
		LastUpdateDate:    timestamppb.New(comment.LastUpdateDate),
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

func toGetTagReply(tag entities.Tag) *posts.GetTagReply {
	return &posts.GetTagReply{
		Id:   int32(tag.Id),
		Name: tag.Name,
	}
}

func toGetTagReplies(input []entities.Tag) []*posts.GetTagReply {
	replies := []*posts.GetTagReply{}
	for _, p := range input {
		reply := toGetTagReply(p)
		replies = append(replies, reply)
	}
	return replies
}
