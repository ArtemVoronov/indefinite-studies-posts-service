package queries

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	utilsEntities "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
)

type CreateCommentParams struct {
	Uuid              interface{}
	AuthorUuid        interface{}
	PostId            interface{}
	Text              interface{}
	LinkedCommentUuid interface{}
}

type UpdateCommentParams struct {
	Id                interface{}
	AuthorUuid        interface{}
	PostId            interface{}
	Text              interface{}
	LinkedCommentUuid interface{}
	State             interface{}
}

const (
	GET_COMMENTS_QUERY = `SELECT 
		id, uuid, author_uuid, text, linked_comment_uuid, state, create_date, last_update_date 
	FROM comments 
	WHERE state != $4 and post_id = $1
	LIMIT $2 OFFSET $3`

	GET_COMMENT_QUERY = `SELECT 
		id, uuid, author_uuid, post_id, text, linked_comment_uuid, state, create_date, last_update_date 
	FROM comments 
	WHERE id = $1 and state != $2`

	CREATE_COMMENT_QUERY = `INSERT INTO comments
		(uuid, author_uuid, post_id, text, linked_comment_uuid, state, create_date, last_update_date) 
		VALUES($1, $2, $3, $4, $5, $6, $7, $8) 
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

func GetComments(tx *sql.Tx, ctx context.Context, postId int, limit int, offset int) ([]entities.Comment, error) {
	var comments []entities.Comment
	var (
		id                int
		uuid              string
		authorUuid        string
		linkedCommentUuid string
		text              string
		state             string
		createDate        time.Time
		lastUpdateDate    time.Time
	)

	rows, err := tx.QueryContext(ctx, GET_COMMENTS_QUERY, postId, limit, offset, utilsEntities.COMMENT_STATE_DELETED)
	if err != nil {
		return comments, fmt.Errorf("error at loading comments from db, case after Query: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &uuid, &authorUuid, &text, &linkedCommentUuid, &state, &createDate, &lastUpdateDate)
		if err != nil {
			return comments, fmt.Errorf("error at loading comments from db, case iterating and using rows.Scan: %s", err)
		}
		comments = append(comments, entities.Comment{Id: id, Uuid: uuid, AuthorUuid: authorUuid, PostId: postId, LinkedCommentUuid: linkedCommentUuid, Text: text, State: state, CreateDate: createDate, LastUpdateDate: lastUpdateDate})
	}
	err = rows.Err()
	if err != nil {
		return comments, fmt.Errorf("error at loading comments from db, case after iterating: %s", err)
	}

	return comments, nil
}

func GetComment(tx *sql.Tx, ctx context.Context, id int) (entities.Comment, error) {
	var comment entities.Comment

	err := tx.QueryRowContext(ctx, GET_COMMENT_QUERY, id, utilsEntities.COMMENT_STATE_DELETED).
		Scan(&comment.Id, &comment.Uuid, &comment.AuthorUuid, &comment.PostId, &comment.Text, &comment.LinkedCommentUuid, &comment.State, &comment.CreateDate, &comment.LastUpdateDate)
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
		params.Uuid, params.AuthorUuid, params.PostId, params.Text, params.LinkedCommentUuid, utilsEntities.COMMENT_STATE_NEW, createDate, lastUpdateDate).
		Scan(&lastInsertId) // scan will release the connection
	if err != nil {
		return -1, fmt.Errorf("error at inserting comment (PostId: '%v', AuthorUuid: '%v') into db, case after QueryRow.Scan: %s", params.PostId, params.AuthorUuid, err)
	}

	return lastInsertId, nil
}

func UpdateComment(tx *sql.Tx, ctx context.Context, params *UpdateCommentParams) error {
	lastUpdateDate := time.Now()

	stmt, err := tx.PrepareContext(ctx, UPDATE_COMMENT_QUERY)
	if err != nil {
		return fmt.Errorf("error at updating comment, case after preparing statement: %s", err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, params.Id, params.Text, params.State, lastUpdateDate, utilsEntities.COMMENT_STATE_DELETED)
	if err != nil {
		return fmt.Errorf("error at updating comment (Id: %v, AuthorUuid: '%v', PostId: '%v'), case after executing statement: %s", params.Id, params.AuthorUuid, params.PostId, err)
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at updating comment (Id: %v, AuthorUuid: '%v', PostId: '%v'), case after counting affected rows: %s", params.Id, params.AuthorUuid, params.PostId, err)
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
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, id, utilsEntities.COMMENT_STATE_DELETED)
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
