package movies

import "context"

const maxPageResults = 9

type MovieProvider interface {
	Popular(ctx context.Context, page int) (Page, error)
}

type Service struct {
	provider MovieProvider
}

type List struct {
	Movies   []Movie
	Page     int
	Returned int
	HasNext  bool
}

func NewService(provider MovieProvider) *Service {
	return &Service{provider: provider}
}

func (service *Service) Popular(ctx context.Context, page int) (List, error) {
	if page < 1 {
		return List{}, ErrInvalidPage
	}

	providerPage, err := service.provider.Popular(ctx, page)
	if err != nil {
		return List{}, err
	}

	returned := len(providerPage.Movies)
	if returned > maxPageResults {
		returned = maxPageResults
	}
	movieList := append([]Movie(nil), providerPage.Movies[:returned]...)
	if movieList == nil {
		movieList = []Movie{}
	}

	return List{
		Movies:   movieList,
		Page:     page,
		Returned: returned,
		HasNext:  page < providerPage.TotalPages,
	}, nil
}
