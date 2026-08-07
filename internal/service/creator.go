package service

import (
	"context"
	"errors"
	"fmt"

	"geoelastic/internal/model"
	"geoelastic/internal/store"
)

// ErrIncompleteBusiness is returned when a business submitted for creation
// is missing its name, phone number, or a complete address — the fields
// Create needs to check whether the business already exists.
var ErrIncompleteBusiness = errors.New("name, phone_number, and a complete address (street, city, state, zip) are required")

// BusinessCreator creates businesses idempotently: submitting the same
// name + full address + phone number twice returns the existing business
// instead of creating a duplicate.
type BusinessCreator struct {
	Store *store.ElasticsearchStore
}

// CreateResult reports the business Create resolved to, and whether it was
// newly indexed (Created=true) or already existed (Created=false).
type CreateResult struct {
	Business model.Business
	Created  bool
}

// Create requires b to carry a name, full address, and phone number (see
// ErrIncompleteBusiness) — those three fields are what CreateBusiness
// derives a business's identity from. If a business with that identity
// already exists, it's returned with Created=false and nothing new is
// indexed; otherwise b is indexed and returned with Created=true.
// Elasticsearch enforces the uniqueness atomically (see CreateBusiness), so
// there's no separate existence check here to race against it.
func (c *BusinessCreator) Create(ctx context.Context, b model.Business) (CreateResult, error) {
	if b.Name == "" || b.PhoneNumber == "" ||
		b.Address.Street == "" || b.Address.City == "" || b.Address.State == "" || b.Address.Zip == "" {
		return CreateResult{}, ErrIncompleteBusiness
	}

	created, err := c.Store.CreateBusiness(ctx, b)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateBusiness) {
			return CreateResult{Business: created, Created: false}, nil
		}
		return CreateResult{}, fmt.Errorf("creating business: %w", err)
	}
	return CreateResult{Business: created, Created: true}, nil
}
