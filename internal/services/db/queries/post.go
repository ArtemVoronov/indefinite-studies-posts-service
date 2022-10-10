package queries

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	"github.com/lib/pq"
)

type CreatePostParams struct {
	Uuid        interface{}
	AuthorId    interface{}
	Text        interface{}
	PreviewText interface{}
	Topic       interface{}
}

type UpdatePostParams struct {
	Uuid        interface{}
	AuthorId    interface{}
	Text        interface{}
	PreviewText interface{}
	Topic       interface{}
	State       interface{}
}

const (
	GET_POSTS_QUERY = `SELECT 
		id, uuid, author_id, text, preview_text, topic, state, create_date, last_update_date 
	FROM posts 
	WHERE state != $3 
	LIMIT $1 OFFSET $2`

	GET_POSTS_BY_IDS_QUERY = `SELECT 
		id, uuid, author_id, text, preview_text, topic, state, create_date, last_update_date 
	FROM posts 
	WHERE state != $4 AND id = ANY($1)
	LIMIT $2 OFFSET $3`

	GET_POSTS_BY_UUIDS_QUERY = `SELECT 
		id, uuid, author_id, text, preview_text, topic, state, create_date, last_update_date 
	FROM posts 
	WHERE state != $4 AND uuid = ANY($1)
	LIMIT $2 OFFSET $3`

	GET_POST_QUERY = `SELECT 
		id, uuid, author_id, text, preview_text, topic, state, create_date, last_update_date 
	FROM posts 
	WHERE id = $1 and state != $2`

	GET_POST_QUERY_BY_UUID = `SELECT 
		id, uuid, author_id, text, preview_text, topic, state, create_date, last_update_date 
	FROM posts 
	WHERE uuid = $1 and state != $2`

	CREATE_POST_QUERY = `INSERT INTO posts
		(uuid, author_id, text, preview_text, topic, state, create_date, last_update_date) 
		VALUES($1, $2, $3, $4, $5, $6, $7, $8) 
	RETURNING id`

	UPDATE_POST_QUERY = `UPDATE posts
	SET author_id = COALESCE($2, author_id),
		text = COALESCE($3, text),
		preview_text = COALESCE($4, preview_text),
		topic = COALESCE($5, topic),
		state = COALESCE($6, state),
		last_update_date = $7
	WHERE id = $1 and state != $8`

	UPDATE_POST_QUERY_BY_UUID = `UPDATE posts
	SET author_id = COALESCE($2, author_id),
		text = COALESCE($3, text),
		preview_text = COALESCE($4, preview_text),
		topic = COALESCE($5, topic),
		state = COALESCE($6, state),
		last_update_date = $7
	WHERE uuid = $1 and state != $8`

	DELETE_POST_QUERY = `UPDATE posts 
	SET state = $2 
	WHERE id = $1 and state != $2`

	DELETE_POST_QUERY_BY_UUID = `UPDATE posts 
	SET state = $2 
	WHERE uuid = $1 and state != $2`
)

func GetPosts(tx *sql.Tx, ctx context.Context, limit int, offset int) ([]entities.Post, error) {
	var posts []entities.Post
	var (
		id             int
		uuid           string
		authorId       int
		text           string
		previewText    string
		topic          string
		state          string
		createDate     time.Time
		lastUpdateDate time.Time
	)

	rows, err := tx.QueryContext(ctx, GET_POSTS_QUERY, limit, offset, entities.POST_STATE_DELETED)
	if err != nil {
		return posts, fmt.Errorf("error at loading posts, case after Query: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &uuid, &authorId, &text, &previewText, &topic, &state, &createDate, &lastUpdateDate)
		if err != nil {
			return posts, fmt.Errorf("error at loading posts, case iterating and using rows.Scan: %s", err)
		}
		posts = append(posts, entities.Post{Id: id, AuthorId: authorId, Uuid: uuid, Text: text, PreviewText: previewText, Topic: topic, State: state, CreateDate: createDate, LastUpdateDate: lastUpdateDate})
	}
	err = rows.Err()
	if err != nil {
		return posts, fmt.Errorf("error at loading posts, case after iterating: %s", err)
	}

	return posts, nil
}

func GetPostsByIds(tx *sql.Tx, ctx context.Context, ids []int, limit int, offset int) ([]entities.Post, error) {
	var posts []entities.Post
	var (
		id             int
		uuid           string
		authorId       int
		text           string
		previewText    string
		topic          string
		state          string
		createDate     time.Time
		lastUpdateDate time.Time
	)
	rows, err := tx.QueryContext(ctx, GET_POSTS_BY_IDS_QUERY, pq.Array(ids), limit, offset, entities.POST_STATE_DELETED)
	if err != nil {
		return posts, fmt.Errorf("error at loading posts by ids, case after Query: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &authorId, &text, &previewText, &topic, &state, &createDate, &lastUpdateDate)
		if err != nil {
			return posts, fmt.Errorf("error at loading posts by ids, case iterating and using rows.Scan: %s", err)
		}
		posts = append(posts, entities.Post{Id: id, Uuid: uuid, AuthorId: authorId, Text: text, PreviewText: previewText, Topic: topic, State: state, CreateDate: createDate, LastUpdateDate: lastUpdateDate})
	}
	err = rows.Err()
	if err != nil {
		return posts, fmt.Errorf("error at loading posts by ids, case after iterating: %s", err)
	}

	return posts, nil
}

func GetPost(tx *sql.Tx, ctx context.Context, uuid string) (entities.Post, error) {
	var post entities.Post

	err := tx.QueryRowContext(ctx, GET_POST_QUERY_BY_UUID, uuid, entities.POST_STATE_DELETED).
		Scan(&post.Id, &post.Uuid, &post.AuthorId, &post.Text, &post.PreviewText, &post.Topic, &post.State, &post.CreateDate, &post.LastUpdateDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return post, err
		} else {
			return post, fmt.Errorf("error at loading post by uuid '%v' from db, case after QueryRow.Scan: %s", uuid, err)
		}
	}

	return post, nil
}

func CreatePost(tx *sql.Tx, ctx context.Context, params *CreatePostParams) (int, error) {
	lastInsertId := -1

	createDate := time.Now()
	lastUpdateDate := time.Now()

	err := tx.QueryRowContext(ctx, CREATE_POST_QUERY,
		params.Uuid, params.AuthorId, params.Text, params.PreviewText, params.Topic, entities.POST_STATE_NEW, createDate, lastUpdateDate).
		Scan(&lastInsertId) // scan will release the connection
	if err != nil {
		return -1, fmt.Errorf("error at inserting post (Topic: '%v', AuthorId: '%v') into db, case after QueryRow.Scan: %s", params.Topic, params.AuthorId, err)
	}

	return lastInsertId, nil
}

func UpdatePost(tx *sql.Tx, ctx context.Context, params *UpdatePostParams) error {
	lastUpdateDate := time.Now()

	stmt, err := tx.PrepareContext(ctx, UPDATE_POST_QUERY_BY_UUID)
	if err != nil {
		return fmt.Errorf("error at updating post, case after preparing statement: %s", err)
	}
	res, err := stmt.ExecContext(ctx, params.Uuid, params.AuthorId, params.Text, params.PreviewText, params.Topic, params.State, lastUpdateDate, entities.POST_STATE_DELETED)
	if err != nil {
		return fmt.Errorf("error at updating post (Uuid: %v, AuthorId: '%v'), case after executing statement: %s", params.Uuid, params.AuthorId, err)
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at updating post (Uuid: %v, AuthorId: '%v'), case after counting affected rows: %s", params.Uuid, params.AuthorId, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeletePost(tx *sql.Tx, ctx context.Context, uuid string) error {
	stmt, err := tx.PrepareContext(ctx, DELETE_POST_QUERY_BY_UUID)
	if err != nil {
		return fmt.Errorf("error at deleting post, case after preparing statement: %s", err)
	}
	res, err := stmt.ExecContext(ctx, uuid, entities.POST_STATE_DELETED)
	if err != nil {
		return fmt.Errorf("error at deleting post by Uuid '%v', case after executing statement: %s", uuid, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting post by Uuid '%v', case after counting affected rows: %s", uuid, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}
