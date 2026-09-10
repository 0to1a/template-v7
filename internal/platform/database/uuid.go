package database

import "github.com/jackc/pgx/v5/pgtype"

// Shared so a malformed UUID is handled consistently
func ParsePublicUUID(publicUUID string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(publicUUID); err != nil {
		return pgtype.UUID{}, err
	}
	return id, nil
}
