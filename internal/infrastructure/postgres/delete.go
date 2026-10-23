package postgres

import (
	"database/sql"
	"errors"
)

// deleteReturning executa uma query DELETE ... RETURNING com um único id e devolve a entidade removida.
// scan copia as colunas do RETURNING para a entidade. Quando nenhuma linha é apagada,
// devolve notFound; os demais erros passam por translate.
func deleteReturning[T any](
	db *sql.DB,
	query string,
	id string,
	scan func(row *sql.Row, entity *T) error,
	notFound error,
	translate func(error) error,
) (*T, error) {
	var entity T

	err := scan(db.QueryRow(query, id), &entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound
		}
		return nil, translate(err)
	}

	return &entity, nil
}
