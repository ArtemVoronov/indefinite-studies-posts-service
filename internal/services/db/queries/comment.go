package queries

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
)

type CreateCommentParams struct {
	AuthorId        interface{}
	PostId          interface{}
	Text            interface{}
	LinkedCommentId interface{}
}

type UpdateCommentParams struct {
	Id              interface{}
	AuthorId        interface{}
	PostId          interface{}
	Text            interface{}
	LinkedCommentId interface{}
	State           interface{}
}

const (
	GET_COMMENTS_QUERY = `SELECT 
		id, author_id, text, linked_comment_id, state, create_date, last_update_date 
	FROM comments 
	WHERE state != $4 and post_id = $1
	LIMIT $2 OFFSET $3`

	GET_COMMENT_QUERY = `SELECT 
		id, author_id, post_id, text, linked_comment_id, state, create_date, last_update_date 
	FROM comments 
	WHERE id = $1 and state != $2`

	CREATE_COMMENT_QUERY = `INSERT INTO comments
		(author_id, post_id, text, linked_comment_id, state, create_date, last_update_date) 
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

func toIntPtr(val sql.NullInt32) *int {
	if val.Valid {
		result := int(val.Int32)
		return &result
	} else {
		return nil
	}
}

func GetComments(tx *sql.Tx, ctx context.Context, postId int, limit int, offset int) ([]entities.Comment, error) {
	var comments []entities.Comment
	var (
		id              int
		authorId        int
		linkedCommentId sql.NullInt32
		text            string
		state           string
		createDate      time.Time
		lastUpdateDate  time.Time
	)

	rows, err := tx.QueryContext(ctx, GET_COMMENTS_QUERY, postId, limit, offset, entities.COMMENT_STATE_DELETED)
	if err != nil {
		return comments, fmt.Errorf("error at loading comments from db, case after Query: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &authorId, &text, &linkedCommentId, &state, &createDate, &lastUpdateDate)
		if err != nil {
			return comments, fmt.Errorf("error at loading comments from db, case iterating and using rows.Scan: %s", err)
		}
		comments = append(comments, entities.Comment{Id: id, AuthorId: authorId, PostId: postId, LinkedCommentId: toIntPtr(linkedCommentId), Text: text, State: state, CreateDate: createDate, LastUpdateDate: lastUpdateDate})
	}
	err = rows.Err()
	if err != nil {
		return comments, fmt.Errorf("error at loading comments from db, case after iterating: %s", err)
	}

	return comments, nil
}

func GetComment(tx *sql.Tx, ctx context.Context, id int) (entities.Comment, error) {
	var comment entities.Comment

	err := tx.QueryRowContext(ctx, GET_COMMENT_QUERY, id, entities.COMMENT_STATE_DELETED).
		Scan(&comment.Id, &comment.AuthorId, &comment.PostId, &comment.Text, &comment.LinkedCommentId, &comment.State, &comment.CreateDate, &comment.LastUpdateDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return comment, err
		} else {
			return comment, fmt.Errorf("error at loading comment by id '%v' from db, case after QueryRow.Scan: %s", id, err)
		}
	}

	return comment, nil
}

func CreateComment(tx *sql.Tx, ctx context.Context, params *CreateCommentParams) (int, error) {
	lastInsertId := -1

	createDate := time.Now()
	lastUpdateDate := time.Now()

	err := tx.QueryRowContext(ctx, CREATE_COMMENT_QUERY,
		params.AuthorId, params.PostId, params.Text, params.LinkedCommentId, entities.COMMENT_STATE_NEW, createDate, lastUpdateDate).
		Scan(&lastInsertId) // scan will release the connection
	if err != nil {
		return -1, fmt.Errorf("error at inserting comment (PostId: '%v', AuthorId: '%v') into db, case after QueryRow.Scan: %s", params.PostId, params.AuthorId, err)
	}

	return lastInsertId, nil
}

func UpdateComment(tx *sql.Tx, ctx context.Context, params *UpdateCommentParams) error {
	lastUpdateDate := time.Now()

	stmt, err := tx.PrepareContext(ctx, UPDATE_COMMENT_QUERY)
	if err != nil {
		return fmt.Errorf("error at updating comment, case after preparing statement: %s", err)
	}
	res, err := stmt.ExecContext(ctx, params.Id, params.Text, params.State, lastUpdateDate, entities.COMMENT_STATE_DELETED)
	if err != nil {
		return fmt.Errorf("error at updating comment (Id: %v, AuthorId: '%v', PostId: '%v'), case after executing statement: %s", params.Id, params.AuthorId, params.PostId, err)
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at updating comment (Id: %v, AuthorId: '%v', PostId: '%v'), case after counting affected rows: %s", params.Id, params.AuthorId, params.PostId, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeleteComment(tx *sql.Tx, ctx context.Context, id int) error {
	stmt, err := tx.PrepareContext(ctx, DELETE_COMMENT_QUERY)
	if err != nil {
		return fmt.Errorf("error at deleting comment, case after preparing statement: %s", err)
	}
	res, err := stmt.ExecContext(ctx, id, entities.COMMENT_STATE_DELETED)
	if err != nil {
		return fmt.Errorf("error at deleting comment by id '%d', case after executing statement: %s", id, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting comment by id '%d', case after counting affected rows: %s", id, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}
