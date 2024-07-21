package history

import (
	"context"
)

type InMemory struct{}

func (i *InMemory) Record(ctx context.Context, userID int64, user, model string) error {
	return nil
}
