package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/db/entities"
)

var ErrorTagDuplicateKey = errors.New("pq: duplicate key value violates unique constraint \"tags_name_unique\"")

const (
	GET_TAGS_QUERY = `SELECT id, name FROM tags LIMIT $1 OFFSET $2`

	GET_TAG_QUERY = `SELECT id, name FROM tags WHERE id = $1`

	CREATE_TAG_QUERY = `INSERT INTO tags (name) VALUES($1) RETURNING id`

	UPDATE_TAG_QUERY = `UPDATE tags SET name = $2 WHERE id = $1`

	DELETE_TAG_QUERY = `DELETE from tags WHERE id = $1`
)

func GetTags(tx *sql.Tx, ctx context.Context, limit int, offset int) ([]entities.Tag, error) {
	var result []entities.Tag
	var (
		id   int
		name string
	)

	rows, err := tx.QueryContext(ctx, GET_TAGS_QUERY, limit, offset)
	if err != nil {
		return result, fmt.Errorf("error at loading tags from db, case after Query: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &name)
		if err != nil {
			return result, fmt.Errorf("error at loading tags from db, case iterating and using rows.Scan: %s", err)
		}
		result = append(result, entities.Tag{Id: id, Name: name})
	}
	err = rows.Err()
	if err != nil {
		return result, fmt.Errorf("error at loading tags from db, case after iterating: %s", err)
	}

	return result, nil
}

func GetTag(tx *sql.Tx, ctx context.Context, id int) (entities.Tag, error) {
	var result entities.Tag

	err := tx.QueryRowContext(ctx, GET_TAG_QUERY, id).
		Scan(&result.Id, &result.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, err
		} else {
			return result, fmt.Errorf("error at loading tag by id '%v' from db, case after QueryRow.Scan: %s", id, err)
		}
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
		return -1, fmt.Errorf("error at inserting tag (Name: '%v') into db, case after QueryRow.Scan: %s", name, err)
	}

	return lastInsertId, nil
}

func UpdateTag(tx *sql.Tx, ctx context.Context, id int, newName string) error {
	stmt, err := tx.PrepareContext(ctx, UPDATE_TAG_QUERY)
	if err != nil {
		return fmt.Errorf("error at updating tag, case after preparing statement: %s", err)
	}
	res, err := stmt.ExecContext(ctx, id, newName)
	if err != nil {
		if err.Error() == ErrorTagDuplicateKey.Error() {
			return ErrorTagDuplicateKey
		}
		return fmt.Errorf("error at updating tag (Id: %v, NewName: '%v'), case after executing statement: %s", id, newName, err)
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at updating tag (Id: %v, NewName: '%v', case after counting affected rows: %s", id, newName, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeleteTag(tx *sql.Tx, ctx context.Context, id int) error {
	stmt, err := tx.PrepareContext(ctx, DELETE_TAG_QUERY)
	if err != nil {
		return fmt.Errorf("error at deleting tag, case after preparing statement: %v", err)
	}
	res, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return fmt.Errorf("error at deleting tag by id '%v', case after executing statement: %v", id, err)
	}
	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error at deleting tag by id '%v', case after counting affected rows: %v", id, err)
	}
	if affectedRowsCount == 0 {
		return sql.ErrNoRows
	}
	return nil
}
