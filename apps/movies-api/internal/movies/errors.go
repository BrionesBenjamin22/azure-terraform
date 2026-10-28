package movies

import "errors"

var (
	ErrInvalidPage             = errors.New("movies: invalid page")
	ErrInvalidQuery            = errors.New("movies: invalid query")
	ErrNotFound                = errors.New("movies: not found")
	ErrProviderUnauthorized    = errors.New("movies: provider unauthorized")
	ErrProviderRateLimited     = errors.New("movies: provider rate limited")
	ErrProviderTimeout         = errors.New("movies: provider timeout")
	ErrProviderUnavailable     = errors.New("movies: provider unavailable")
	ErrProviderInvalidResponse = errors.New("movies: provider invalid response")
)
