package queries

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
)

func GetPosts(tx *sql.Tx, ctx context.Context, limit int, offset int) ([]entities.Post, error) {
	var post []entities.Post
	var (
		id             int
		authorId       int
		text           string
		topic          string
		state          string
		createDate     time.Time
		lastUpdateDate time.Time
	)

	rows, err := tx.QueryContext(ctx, "SELECT id, text, topic, author_id, state, create_date, last_update_date FROM posts WHERE state != $3 LIMIT $1 OFFSET $2 ", limit, offset, entities.POST_STATE_DELETED)
	if err != nil {
		return post, fmt.Errorf("error at loading post from db, case after Query: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &text, &topic, &authorId, &state, &createDate, &lastUpdateDate)
		if err != nil {
			return post, fmt.Errorf("error at loading posts from db, case iterating and using rows.Scan: %s", err)
		}
		post = append(post, entities.Post{Id: id, Text: text, Topic: topic, AuthorId: authorId, State: state, CreateDate: createDate, LastUpdateDate: lastUpdateDate})
	}
	err = rows.Err()
	if err != nil {
		return post, fmt.Errorf("error at loading posts from db, case after iterating: %s", err)
	}

	return post, nil
}

func GetPost(tx *sql.Tx, ctx context.Context, id int) (entities.Post, error) {
	var post entities.Post

	err := tx.QueryRowContext(ctx, "SELECT id, text, topic, author_id, state, create_date, last_update_date FROM posts WHERE id = $1 and state != $2 ", id, entities.POST_STATE_DELETED).
		Scan(&post.Id, &post.Text, &post.Topic, &post.AuthorId, &post.State, &post.CreateDate, &post.LastUpdateDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return post, err
		} else {
			return post, fmt.Errorf("error at loading post by id '%d' from db, case after QueryRow.Scan: %s", id, err)
		}
	}

	return post, nil
}

func CreatePost(tx *sql.Tx, ctx context.Context, text string, topic string, authorId int, state string) (int, error) {
	lastInsertId := -1

	createDate := time.Now()
	lastUpdateDate := time.Now()

	err := tx.QueryRowContext(ctx, "INSERT INTO posts(text, topic, author_id, state, create_date, last_update_date) VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		text, topic, authorId, state, createDate, lastUpdateDate).
		Scan(&lastInsertId) // scan will release the connection
	if err != nil {
		return -1, fmt.Errorf("error at inserting post (Topic: '%s', AuthorId: '%d') into db, case after QueryRow.Scan: %s", topic, authorId, err)
	}

	return lastInsertId, nil
}

func UpdatePost(tx *sql.Tx, ctx context.Context, id int, text string, topic string, authorId int, state string) error {
	lastUpdateDate := time.Now()
	stmt, err := tx.PrepareContext(ctx, "UPDATE posts SET text = $2, topic = $3, author_id = $4, state = $5, last_update_date = $6 WHERE id = $1 and state != $7")
	if err != nil {
		return fmt.Errorf("error at updating post, case after preparing statement: %s", err)
	}
	res, err := stmt.ExecContext(ctx, id, text, topic, authorId, state, lastUpdateDate, entities.POST_STATE_DELETED)
	if err != nil {
		return fmt.Errorf("error at updating post (Id: %d, Topic: '%s', AuthorId: '%d', State: '%s'), case after executing statement: %s", id, topic, authorId, state, err)
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at updating post (Id: %d, Topic: '%s', AuthorId: '%d', State: '%s'), case after counting affected rows: %s", id, topic, authorId, state, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeletePost(tx *sql.Tx, ctx context.Context, id int) error {
	stmt, err := tx.PrepareContext(ctx, "UPDATE posts SET state = $2 WHERE id = $1 and state != $2")
	if err != nil {
		return fmt.Errorf("error at deleting post, case after preparing statement: %s", err)
	}
	res, err := stmt.ExecContext(ctx, id, entities.POST_STATE_DELETED)
	if err != nil {
		return fmt.Errorf("error at deleting post by id '%d', case after executing statement: %s", id, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting post by id '%d', case after counting affected rows: %s", id, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}
