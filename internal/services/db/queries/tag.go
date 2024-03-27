package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
)

var ErrorTagDuplicateKey = errors.New("pq: duplicate key value violates unique constraint \"tags_name_unique\"")
var ErrorPostTagDuplicateKey = errors.New("pq: duplicate key value violates unique constraint \"PK_posts_and_tags\"")

const (
	GET_TAGS_QUERY = `SELECT id, name FROM tags LIMIT $1 OFFSET $2`

	GET_TAG_QUERY = `SELECT id, name FROM tags WHERE id = $1`

	CREATE_TAG_QUERY = `INSERT INTO tags (name) VALUES($1) RETURNING id`

	UPDATE_TAG_QUERY = `UPDATE tags SET name = $2 WHERE id = $1`

	DELETE_TAG_QUERY = `DELETE FROM tags WHERE id = $1`

	ASSIGN_TAG_TO_POST_QUERY = `INSERT INTO posts_and_tags (post_id, tag_id) VALUES($1, $2)`

	REMOVE_TAG_FROM_POST_QUERY = `DELETE FROM posts_and_tags WHERE post_id = $1 and tag_id = $2`

	REMOVE_ALL_TAGS_FROM_POST_QUERY = `DELETE FROM posts_and_tags WHERE post_id = $1`

	GET_TAGS_BY_POST_ID_QUERY = `SELECT tags.id, tags.name 
    FROM (SELECT tag_id FROM posts_and_tags WHERE post_id = $1) as chosen_tags
    INNER JOIN tags ON chosen_tags.tag_id = tags.id;
    `

	GET_TAG_IDS_BY_POST_ID_QUERY = `SELECT tag_id 
    FROM posts_and_tags 
    WHERE post_id = $1;
    `

	GET_TAGS_BY_IDS_QUERY = `SELECT tags.id, tags.name 
    FROM tags 
    WHERE tags.id = ANY($1::int[]);
    `
)

func GetTags(tx *sql.Tx, ctx context.Context, limit int, offset int) ([]entities.Tag, error) {
	var result []entities.Tag = make([]entities.Tag, 0)
	var (
		id   int
		name string
	)

	rows, err := tx.QueryContext(ctx, GET_TAGS_QUERY, limit, offset)
	if err != nil {
		return result, fmt.Errorf("error at loading tags from db, case after Query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &name)
		if err != nil {
			return result, fmt.Errorf("error at loading tags from db, case iterating and using rows.Scan: %w", err)
		}
		result = append(result, entities.Tag{Id: id, Name: name})
	}
	err = rows.Err()
	if err != nil {
		return result, fmt.Errorf("error at loading tags from db, case after iterating: %w", err)
	}

	return result, nil
}

func GetTag(tx *sql.Tx, ctx context.Context, id int) (entities.Tag, error) {
	var result entities.Tag

	err := tx.QueryRowContext(ctx, GET_TAG_QUERY, id).
		Scan(&result.Id, &result.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return result, err
	} else if err != nil {
		return result, fmt.Errorf("error at loading tag by id '%v' from db, case after QueryRow.Scan: %w", id, err)
	}

	return result, nil
}

func CreateTag(tx *sql.Tx, ctx context.Context, name string) (int, error) {
	lastInsertId := -1

	err := tx.QueryRowContext(ctx, CREATE_TAG_QUERY, name).Scan(&lastInsertId) // scan will release the connection
	if err != nil {
		if err.Error() == ErrorTagDuplicateKey.Error() {
			return -1, ErrorTagDuplicateKey
		}
		return -1, fmt.Errorf("error at inserting tag (Name: '%v') into db, case after QueryRow.Scan: %w", name, err)
	}

	return lastInsertId, nil
}

func UpdateTag(tx *sql.Tx, ctx context.Context, id int, newName string) error {
	stmt, err := tx.PrepareContext(ctx, UPDATE_TAG_QUERY)
	if err != nil {
		return fmt.Errorf("error at updating tag, case after preparing statement: %w", err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, id, newName)
	if err != nil {
		if err.Error() == ErrorTagDuplicateKey.Error() {
			return ErrorTagDuplicateKey
		}
		return fmt.Errorf("error at updating tag (Id: %v, NewName: '%v'), case after executing statement: %w", id, newName, err)
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at updating tag (Id: %v, NewName: '%v', case after counting affected rows: %w", id, newName, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeleteTag(tx *sql.Tx, ctx context.Context, id int) error {
	stmt, err := tx.PrepareContext(ctx, DELETE_TAG_QUERY)
	if err != nil {
		return fmt.Errorf("error at deleting tag, case after preparing statement: %w", err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return fmt.Errorf("error at deleting tag by id '%v', case after executing statement: %w", id, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting tag by id '%v', case after counting affected rows: %w", id, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func AssignTagToPost(tx *sql.Tx, ctx context.Context, postId int, tagId int) error {
	stmt, err := tx.PrepareContext(ctx, ASSIGN_TAG_TO_POST_QUERY)
	if err != nil {
		return fmt.Errorf("error at inserting to posts_and_tags (PostId: '%v', TagId: '%v'), case after preparing statement: %w", postId, tagId, err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, postId, tagId)
	if err != nil {
		return fmt.Errorf("error at inserting to posts_and_tags (PostId: '%v', TagId: '%v'), case after ExecContext: %w", postId, tagId, err)
	}

	return nil
}

func RemoveTagFromPost(tx *sql.Tx, ctx context.Context, postId int, tagId int) error {
	stmt, err := tx.PrepareContext(ctx, REMOVE_TAG_FROM_POST_QUERY)
	if err != nil {
		return fmt.Errorf("error at deleting from posts_and_tags (PostId: '%v', TagId: '%v'), case after preparing statement: %w", postId, tagId, err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, postId, tagId)
	if err != nil {
		return fmt.Errorf("error at deleting from posts_and_tags (PostId: '%v', TagId: '%v'), case after executing statement: %w", postId, tagId, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting from posts_and_tags (PostId: '%v', TagId: '%v'), case after counting affected rows: %w", postId, tagId, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func RemoveAllTagsFromPost(tx *sql.Tx, ctx context.Context, postId int) error {
	stmt, err := tx.PrepareContext(ctx, REMOVE_ALL_TAGS_FROM_POST_QUERY)
	if err != nil {
		return fmt.Errorf("error at deleting from posts_and_tags (PostId: '%v'), case after preparing statement: %w", postId, err)
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, postId)
	if err != nil {
		return fmt.Errorf("error at deleting from posts_and_tags (PostId: '%v'), case after executing statement: %w", postId, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting from posts_and_tags (PostId: '%v'), case after counting affected rows: %w", postId, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func GetTagIdsByPostId(tx *sql.Tx, ctx context.Context, postId int) ([]int, error) {
	var result []int
	var (
		id int
	)

	rows, err := tx.QueryContext(ctx, GET_TAG_IDS_BY_POST_ID_QUERY, postId)
	if err != nil {
		return result, fmt.Errorf("error at loading tags from db, case after Query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id)
		if err != nil {
			return result, fmt.Errorf("error at loading tags from db, case iterating and using rows.Scan: %w", err)
		}
		result = append(result, id)
	}
	err = rows.Err()
	if err != nil {
		return result, fmt.Errorf("error at loading tags from db, case after iterating: %w", err)
	}

	return result, nil
}

func GetTagsByIds(tx *sql.Tx, ctx context.Context, tagIds []int) ([]entities.Tag, error) {
	tagsConverted := make([]string, len(tagIds), len(tagIds))
	for i, tagId := range tagIds {
		tagsConverted[i] = fmt.Sprintf("%v", tagId)
	}
	tagsStr := "{" + strings.Join(tagsConverted, ",") + "}"

	var result []entities.Tag = make([]entities.Tag, 0)
	var (
		id   int
		name string
	)

	rows, err := tx.QueryContext(ctx, GET_TAGS_BY_IDS_QUERY, tagsStr)
	if err != nil {
		return result, fmt.Errorf("error at loading tags from db, case after Query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &name)
		if err != nil {
			return result, fmt.Errorf("error at loading tags from db, case iterating and using rows.Scan: %w", err)
		}
		result = append(result, entities.Tag{Id: id, Name: name})
	}
	err = rows.Err()
	if err != nil {
		return result, fmt.Errorf("error at loading tags from db, case after iterating: %w", err)
	}

	return result, nil
}
