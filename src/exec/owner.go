package exec

import (
	"app/src/utils/db"
	"context"
)

type ownerService struct {
	ctx context.Context
}

func Owner(ctx context.Context) ownerService {
	return ownerService{ctx: ctx}
}

func (service ownerService) Create() (string, error) {
	ctx, cancel := context.WithTimeout(service.ctx, MaxTxTimeout)
	defer cancel()

	var ownerid string
	err := db.Conn().QueryRowContext(ctx, `INSERT INTO core.owner DEFAULT VALUES RETURNING owner_id`).Scan(&ownerid)

	if err != nil {
		return "", err
	}

	return ownerid, nil
}
