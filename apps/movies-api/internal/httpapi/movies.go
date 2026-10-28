package httpapi

import (
	"net/http"
	"strconv"

	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/movies"
)

type movieResponse struct {
	ID            int64           `json:"id"`
	Title         string          `json:"title"`
	OriginalTitle string          `json:"original_title"`
	Overview      string          `json:"overview"`
	ReleaseDate   string          `json:"release_date"`
	PosterPath    *string         `json:"poster_path"`
	BackdropPath  *string         `json:"backdrop_path"`
	VoteAverage   float64         `json:"vote_average"`
	GenreIDs      []int64         `json:"genre_ids,omitempty"`
	Genres        []genreResponse `json:"genres,omitempty"`
}

type genreResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type movieListEnvelope struct {
	Data []movieResponse `json:"data"`
	Meta listMetadata    `json:"meta"`
}

type listMetadata struct {
	Page     int  `json:"page"`
	Returned int  `json:"returned"`
	HasNext  bool `json:"has_next"`
}

func popularHandler(service *movies.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := parsePage(r)
		if err != nil {
			writeMappedError(w, r, err)
			return
		}

		result, err := service.Popular(r.Context(), page)
		if err != nil {
			writeMappedError(w, r, err)
			return
		}

		data := make([]movieResponse, len(result.Movies))
		for index, movie := range result.Movies {
			data[index] = mapMovieResponse(movie)
		}
		writeJSON(w, http.StatusOK, movieListEnvelope{
			Data: data,
			Meta: listMetadata{Page: result.Page, Returned: result.Returned, HasNext: result.HasNext},
		})
	}
}

func parsePage(r *http.Request) (int, error) {
	values, found := r.URL.Query()["page"]
	if !found {
		return 1, nil
	}
	if len(values) != 1 || values[0] == "" {
		return 0, movies.ErrInvalidPage
	}

	page, err := strconv.Atoi(values[0])
	if err != nil || page < 1 {
		return 0, movies.ErrInvalidPage
	}
	return page, nil
}

func mapMovieResponse(movie movies.Movie) movieResponse {
	genres := make([]genreResponse, len(movie.Genres))
	for index, genre := range movie.Genres {
		genres[index] = genreResponse{ID: genre.ID, Name: genre.Name}
	}

	return movieResponse{
		ID:            movie.ID,
		Title:         movie.Title,
		OriginalTitle: movie.OriginalTitle,
		Overview:      movie.Overview,
		ReleaseDate:   movie.ReleaseDate,
		PosterPath:    movie.PosterPath,
		BackdropPath:  movie.BackdropPath,
		VoteAverage:   movie.VoteAverage,
		GenreIDs:      append([]int64(nil), movie.GenreIDs...),
		Genres:        genres,
	}
}
