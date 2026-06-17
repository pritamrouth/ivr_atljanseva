package repository

import (
	"context"
	"database/sql"

	"ivr_ataljanseva/models"
)

type PoliticalUserRepository struct {
	db *sql.DB
}

func NewPoliticalUserRepository(db *sql.DB) *PoliticalUserRepository {
	return &PoliticalUserRepository{
		db: db,
	}
}


func (r *PoliticalUserRepository) FindMatchingWards(
	ctx context.Context,
	pincode string,
	wardInput string,
) ([]models.WardMatch, error) {

	query := `
	SELECT DISTINCT
		ward,
		id,
		full_name
	FROM political_users
	WHERE pincode::text = $1
	  AND ward ILIKE '%' || $2 || '%'
	  AND is_active = true
	ORDER BY ward
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		pincode,
		wardInput,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []models.WardMatch

	for rows.Next() {

		var m models.WardMatch

		err := rows.Scan(
			&m.Ward,
			&m.NagarsevakID,
			&m.NagarsevakName,
		)
		if err != nil {
			return nil, err
		}

		matches = append(matches, m)
	}

	return matches, nil
}

func (r *PoliticalUserRepository) FindNagarsevaks(
	ctx context.Context,
	pincode string,
	ward string,
) ([]models.NagarsevakRecord, error) {

	query := `
	SELECT
		id,
		full_name
	FROM political_users
	WHERE pincode::text = $1
	  AND ward = $2
	  AND is_active = true
	ORDER BY full_name
	`

	rows, err := r.db.QueryContext(ctx, query, pincode, ward)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.NagarsevakRecord

	for rows.Next() {
		var rec models.NagarsevakRecord
		if err := rows.Scan(&rec.ID, &rec.Name); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	return records, nil
}