package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
	utilsEntities "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db/entities"
	"github.com/lib/pq"
)

type CreatePostParams struct {
	Uuid        interface{}
	AuthorUuid  interface{}
	Text        interface{}
	PreviewText interface{}
	Topic       interface{}
}

type UpdatePostParams struct {
	Uuid        interface{}
	AuthorUuid  interface{}
	Text        interface{}
	PreviewText interface{}
	Topic       interface{}
	State       interface{}
}

// TODO: add memory safe pagination without direct offset, use sorting by id and where criteria

const (
	GET_POSTS_QUERY = `SELECT 
		id, uuid, author_uuid, text, preview_text, topic, state, create_date, last_update_date 
	FROM posts 
	WHERE state != $3 
	LIMIT $1 OFFSET $2`

	GET_POSTS_WITH_TAGS_QUERY = `SELECT 
		posts.id, posts.uuid, posts.author_uuid, posts.text, posts.preview_text, posts.topic, posts.state, posts.create_date, posts.last_update_date, 
		array_agg(posts_and_tags.tag_id) as tags
	FROM posts 
	LEFT OUTER JOIN posts_and_tags ON posts.id = posts_and_tags.post_id
	WHERE state != $3 
	GROUP BY posts.id, posts.uuid, posts.author_uuid, posts.text, posts.preview_text, posts.topic, posts.state, posts.create_date, posts.last_update_date
	ORDER BY posts.id ASC
	LIMIT $1
	OFFSET $2
	`

	GET_POSTS_BY_IDS_QUERY = `SELECT 
		id, uuid, author_uuid, text, preview_text, topic, state, create_date, last_update_date 
	FROM posts 
	WHERE state != $4 AND id = ANY($1)
	LIMIT $2 OFFSET $3`

	GET_POSTS_BY_UUIDS_QUERY = `SELECT 
		id, uuid, author_uuid, text, preview_text, topic, state, create_date, last_update_date 
	FROM posts 
	WHERE state != $4 AND uuid = ANY($1)
	LIMIT $2 OFFSET $3`

	GET_POST_QUERY = `SELECT 
		id, uuid, author_uuid, text, preview_text, topic, state, create_date, last_update_date 
	FROM posts 
	WHERE id = $1 and state != $2`

	GET_POST_QUERY_BY_UUID = `SELECT 
		id, uuid, author_uuid, text, preview_text, topic, state, create_date, last_update_date 
	FROM posts 
	WHERE uuid = $1 and state != $2`

	GET_POST_WITH_TAGS_BY_UUID_QUERY = `SELECT 
		posts.id, posts.uuid, posts.author_uuid, posts.text, posts.preview_text, posts.topic, posts.state, posts.create_date, posts.last_update_date, 
		array_agg(posts_and_tags.tag_id) as tags 
	FROM posts 
	LEFT OUTER JOIN posts_and_tags ON posts.id = posts_and_tags.post_id
	WHERE uuid = $1 and state != $2
	GROUP BY posts.id, posts.uuid, posts.author_uuid, posts.text, posts.preview_text, posts.topic, posts.state, posts.create_date, posts.last_update_date`

	CREATE_POST_QUERY = `INSERT INTO posts
		(uuid, author_uuid, text, preview_text, topic, state, create_date, last_update_date) 
		VALUES($1, $2, $3, $4, $5, $6, $7, $8) 
	RETURNING id`

	UPDATE_POST_QUERY = `UPDATE posts
	SET author_uuid = COALESCE($2, author_uuid),
		text = COALESCE($3, text),
		preview_text = COALESCE($4, preview_text),
		topic = COALESCE($5, topic),
		state = COALESCE($6, state),
		last_update_date = $7
	WHERE id = $1 and state != $8`

	UPDATE_POST_QUERY_BY_UUID = `UPDATE posts
	SET author_uuid = COALESCE($2, author_uuid),
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
		authorUuid     string
		text           string
		previewText    string
		topic          string
		state          string
		createDate     time.Time
		lastUpdateDate time.Time
	)

	rows, err := tx.QueryContext(ctx, GET_POSTS_QUERY, limit, offset, utilsEntities.POST_STATE_DELETED)
	if err != nil {
		return posts, fmt.Errorf("error at loading posts, case after Query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &uuid, &authorUuid, &text, &previewText, &topic, &state, &createDate, &lastUpdateDate)
		if err != nil {
			return posts, fmt.Errorf("error at loading posts, case iterating and using rows.Scan: %w", err)
		}
		posts = append(posts, entities.Post{Id: id, AuthorUuid: authorUuid, Uuid: uuid, Text: text, PreviewText: previewText, Topic: topic, State: state, CreateDate: createDate, LastUpdateDate: lastUpdateDate})
	}
	err = rows.Err()
	if err != nil {
		return posts, fmt.Errorf("error at loading posts, case after iterating: %w", err)
	}

	return posts, nil
}

func GetPostsByIds(tx *sql.Tx, ctx context.Context, ids []int, limit int, offset int) ([]entities.Post, error) {
	var posts []entities.Post
	var (
		id             int
		uuid           string
		authorUuid     string
		text           string
		previewText    string
		topic          string
		state          string
		createDate     time.Time
		lastUpdateDate time.Time
	)
	rows, err := tx.QueryContext(ctx, GET_POSTS_BY_IDS_QUERY, pq.Array(ids), limit, offset, utilsEntities.POST_STATE_DELETED)
	if err != nil {
		return posts, fmt.Errorf("error at loading posts by ids, case after Query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &authorUuid, &text, &previewText, &topic, &state, &createDate, &lastUpdateDate)
		if err != nil {
			return posts, fmt.Errorf("error at loading posts by ids, case iterating and using rows.Scan: %w", err)
		}
		posts = append(posts, entities.Post{Id: id, Uuid: uuid, AuthorUuid: authorUuid, Text: text, PreviewText: previewText, Topic: topic, State: state, CreateDate: createDate, LastUpdateDate: lastUpdateDate})
	}
	err = rows.Err()
	if err != nil {
		return posts, fmt.Errorf("error at loading posts by ids, case after iterating: %w", err)
	}

	return posts, nil
}

func GetPost(tx *sql.Tx, ctx context.Context, uuid string) (entities.Post, error) {
	var post entities.Post

	err := tx.QueryRowContext(ctx, GET_POST_QUERY_BY_UUID, uuid, utilsEntities.POST_STATE_DELETED).
		Scan(&post.Id, &post.Uuid, &post.AuthorUuid, &post.Text, &post.PreviewText, &post.Topic, &post.State, &post.CreateDate, &post.LastUpdateDate)
	if errors.Is(err, sql.ErrNoRows) {
		return post, err
	} else if err != nil {
		return post, fmt.Errorf("error at loading post by uuid '%v' from db, case after QueryRow.Scan: %w", uuid, err)
	}

	return post, nil
}

func GetPostWithTagIds(tx *sql.Tx, ctx context.Context, uuid string) (entities.PostWithTagIds, error) {
	var result entities.PostWithTagIds
	post, err := GetPost(tx, ctx, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, err
		} else {
			return result, fmt.Errorf("error at loading post by uuid '%v' from db, case after QueryRow.Scan: %w", uuid, err)
		}
	}
	tagIds, err := GetTagIdsByPostId(tx, ctx, post.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, err
		} else {
			return result, fmt.Errorf("error at loading post tags. Post uuid '%v'. Error: %w", uuid, err)
		}
	}
	result.Post = post
	result.TagIds = tagIds
	return result, nil
}

func CreatePost(tx *sql.Tx, ctx context.Context, params *CreatePostParams) (int, error) {
	lastInsertId := -1

	createDate := time.Now()
	lastUpdateDate := time.Now()

	err := tx.QueryRowContext(ctx, CREATE_POST_QUERY,
		params.Uuid, params.AuthorUuid, params.Text, params.PreviewText, params.Topic, utilsEntities.POST_STATE_NEW, createDate, lastUpdateDate).
		Scan(&lastInsertId) // scan will release the connection
	if err != nil {
		return -1, fmt.Errorf("error at inserting post (Topic: '%v', AuthorUuid: '%v') into db, case after QueryRow.Scan: %w", params.Topic, params.AuthorUuid, err)
	}

	return lastInsertId, nil
}

func UpdatePost(tx *sql.Tx, ctx context.Context, params *UpdatePostParams) error {
	lastUpdateDate := time.Now()

	stmt, err := tx.PrepareContext(ctx, UPDATE_POST_QUERY_BY_UUID)
	if err != nil {
		return fmt.Errorf("error at updating post, case after preparing statement: %w", err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, params.Uuid, params.AuthorUuid, params.Text, params.PreviewText, params.Topic, params.State, lastUpdateDate, utilsEntities.POST_STATE_DELETED)
	if err != nil {
		return fmt.Errorf("error at updating post (Uuid: %v, AuthorUuid: '%v'), case after executing statement: %w", params.Uuid, params.AuthorUuid, err)
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at updating post (Uuid: %v, AuthorUuid: '%v'), case after counting affected rows: %w", params.Uuid, params.AuthorUuid, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeletePost(tx *sql.Tx, ctx context.Context, uuid string) error {
	stmt, err := tx.PrepareContext(ctx, DELETE_POST_QUERY_BY_UUID)
	if err != nil {
		return fmt.Errorf("error at deleting post, case after preparing statement: %w", err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, uuid, utilsEntities.POST_STATE_DELETED)
	if err != nil {
		return fmt.Errorf("error at deleting post by Uuid '%v', case after executing statement: %w", uuid, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting post by Uuid '%v', case after counting affected rows: %w", uuid, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}
