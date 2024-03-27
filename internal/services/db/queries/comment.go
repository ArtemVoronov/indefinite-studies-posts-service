package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	utilsEntities "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
)

type CreateCommentParams struct {
	AuthorUuid      interface{}
	PostUuid        interface{}
	Text            interface{}
	LinkedCommentId interface{}
}

type UpdateCommentParams struct {
	Id              interface{}
	AuthorUuid      interface{}
	PostId          interface{}
	Text            interface{}
	LinkedCommentId interface{}
	State           interface{}
}

// TODO: add memory safe pagination without direct offset, use sorting by id and where criteria

const (
	GET_COMMENTS_QUERY = `SELECT 
		id, author_uuid, text, linked_comment_id, state, create_date, last_update_date 
	FROM comments 
	WHERE state != $4 and post_uuid = $1
	LIMIT $2 OFFSET $3`

	GET_COMMENT_QUERY = `SELECT 
		id, author_uuid, post_uuid, text, linked_comment_id, state, create_date, last_update_date 
	FROM comments 
	WHERE id = $1 and state != $2`

	CREATE_COMMENT_QUERY = `INSERT INTO comments
		(author_uuid, post_uuid, text, linked_comment_id, state, create_date, last_update_date) 
		VALUES($1, $2, $3, $4, $5, $6, $7) 
	RETURNING id`

	UPDATE_COMMENT_QUERY = `UPDATE comments
	SET text = COALESCE($2, text),
		state = COALESCE($3, state),
		last_update_date = $4
	WHERE id = $1 and state != $5`

	DELETE_COMMENT_QUERY = `UPDATE comments 
	SET state = $2 
	WHERE id = $1 and state != $2`
)

func GetComments(tx *sql.Tx, ctx context.Context, postUuid string, limit int, offset int) ([]entities.Comment, error) {
	var comments []entities.Comment
	// TODO: implement getting comments by post UUID with safe pagination (where id > 0 + limit)
	return comments, nil
}

func GetComment(tx *sql.Tx, ctx context.Context, id int) (entities.Comment, error) {
	var comment entities.Comment

	err := tx.QueryRowContext(ctx, GET_COMMENT_QUERY, id, utilsEntities.COMMENT_STATE_DELETED).
		Scan(&comment.Id, &comment.AuthorUuid, &comment.PostUuid, &comment.Text, &comment.LinkedCommentId, &comment.State, &comment.CreateDate, &comment.LastUpdateDate)
	if errors.Is(err, sql.ErrNoRows) {
		return comment, err
	} else if err != nil {
		return comment, fmt.Errorf("error at loading comment by id '%v' from db, case after QueryRow.Scan: %w", id, err)
	}

	return comment, nil
}

func CreateComment(tx *sql.Tx, ctx context.Context, params *CreateCommentParams) (int, error) {
	lastInsertId := -1

	createDate := time.Now()
	lastUpdateDate := time.Now()

	err := tx.QueryRowContext(ctx, CREATE_COMMENT_QUERY,
		params.AuthorUuid, params.PostUuid, params.Text, params.LinkedCommentId, utilsEntities.COMMENT_STATE_NEW, createDate, lastUpdateDate).
		Scan(&lastInsertId) // scan will release the connection
	if err != nil {
		return -1, fmt.Errorf("error at inserting comment (PostUuid: '%v', AuthorUuid: '%v') into db, case after QueryRow.Scan: %w", params.PostUuid, params.AuthorUuid, err)
	}

	return lastInsertId, nil
}

func UpdateComment(tx *sql.Tx, ctx context.Context, params *UpdateCommentParams) error {
	lastUpdateDate := time.Now()

	stmt, err := tx.PrepareContext(ctx, UPDATE_COMMENT_QUERY)
	if err != nil {
		return fmt.Errorf("error at updating comment, case after preparing statement: %w", err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, params.Id, params.Text, params.State, lastUpdateDate, utilsEntities.COMMENT_STATE_DELETED)
	if err != nil {
		return fmt.Errorf("error at updating comment (Id: %v, AuthorUuid: '%v', PostId: '%v'), case after executing statement: %w", params.Id, params.AuthorUuid, params.PostId, err)
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at updating comment (Id: %v, AuthorUuid: '%v', PostId: '%v'), case after counting affected rows: %w", params.Id, params.AuthorUuid, params.PostId, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeleteComment(tx *sql.Tx, ctx context.Context, id int) error {
	stmt, err := tx.PrepareContext(ctx, DELETE_COMMENT_QUERY)
	if err != nil {
		return fmt.Errorf("error at deleting comment, case after preparing statement: %w", err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, id, utilsEntities.COMMENT_STATE_DELETED)
	if err != nil {
		return fmt.Errorf("error at deleting comment by id '%v', case after executing statement: %w", id, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting comment by id '%v', case after counting affected rows: %w", id, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}
