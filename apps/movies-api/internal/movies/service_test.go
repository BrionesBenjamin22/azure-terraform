package movies

import (
	"context"
	"errors"
	"testing"
)

func TestPopularLimitsResultsAndBuildsMetadata(t *testing.T) {
	providerMovies := make([]Movie, 12)
	for index := range providerMovies {
		providerMovies[index].ID = int64(index + 1)
	}
	provider := &fakeProvider{page: Page{Number: 2, TotalPages: 4, Movies: providerMovies}}
	service := NewService(provider)

	result, err := service.Popular(context.Background(), 2)
	if err != nil {
		t.Fatalf("popular: %v", err)
	}
	if provider.calls != 1 || provider.requestedPage != 2 {
		t.Errorf("unexpected provider calls: calls=%d page=%d", provider.calls, provider.requestedPage)
	}
	if len(result.Movies) != 9 || result.Returned != 9 || result.Page != 2 || !result.HasNext {
		t.Errorf("unexpected result: %+v", result)
	}
	if result.Movies[8].ID != 9 {
		t.Errorf("expected deterministic first nine results, got %+v", result.Movies)
	}
}

func TestPopularReturnsEmptySliceAndNoNextPage(t *testing.T) {
	service := NewService(&fakeProvider{page: Page{Number: 1, TotalPages: 1}})

	result, err := service.Popular(context.Background(), 1)
	if err != nil {
		t.Fatalf("popular: %v", err)
	}
	if result.Movies == nil || result.Returned != 0 || result.HasNext {
		t.Errorf("unexpected empty result: %+v", result)
	}
}

func TestPopularRejectsInvalidPageWithoutCallingProvider(t *testing.T) {
	provider := &fakeProvider{}
	service := NewService(provider)

	if _, err := service.Popular(context.Background(), 0); !errors.Is(err, ErrInvalidPage) {
		t.Fatalf("expected invalid page, got %v", err)
	}
	if provider.calls != 0 {
		t.Fatalf("expected no provider calls, got %d", provider.calls)
	}
}

func TestPopularPreservesProviderError(t *testing.T) {
	provider := &fakeProvider{err: ErrProviderUnavailable}
	service := NewService(provider)

	if _, err := service.Popular(context.Background(), 1); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected provider error, got %v", err)
	}
}

type fakeProvider struct {
	page          Page
	err           error
	calls         int
	requestedPage int
}

func (provider *fakeProvider) Popular(_ context.Context, page int) (Page, error) {
	provider.calls++
	provider.requestedPage = page
	return provider.page, provider.err
}
