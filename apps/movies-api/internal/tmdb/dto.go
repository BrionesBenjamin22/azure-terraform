package tmdb

import "github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/movies"

type listResponse struct {
	Page       int        `json:"page"`
	TotalPages int        `json:"total_pages"`
	Results    []movieDTO `json:"results"`
}

type movieDTO struct {
	ID            int64      `json:"id"`
	Title         string     `json:"title"`
	OriginalTitle string     `json:"original_title"`
	Overview      string     `json:"overview"`
	ReleaseDate   string     `json:"release_date"`
	PosterPath    *string    `json:"poster_path"`
	BackdropPath  *string    `json:"backdrop_path"`
	VoteAverage   float64    `json:"vote_average"`
	GenreIDs      []int64    `json:"genre_ids"`
	Genres        []genreDTO `json:"genres"`
}

type genreDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func mapPage(response listResponse) (movies.Page, error) {
	if response.Page <= 0 || response.TotalPages < 0 {
		return movies.Page{}, ErrInvalidResponse
	}

	result := movies.Page{
		Number:     response.Page,
		TotalPages: response.TotalPages,
		Movies:     make([]movies.Movie, len(response.Results)),
	}
	for index, movie := range response.Results {
		if movie.ID <= 0 {
			return movies.Page{}, ErrInvalidResponse
		}
		result.Movies[index] = mapMovie(movie)
	}
	return result, nil
}

func mapMovie(source movieDTO) movies.Movie {
	genres := make([]movies.Genre, len(source.Genres))
	for index, genre := range source.Genres {
		genres[index] = movies.Genre{ID: genre.ID, Name: genre.Name}
	}

	return movies.Movie{
		ID:            source.ID,
		Title:         source.Title,
		OriginalTitle: source.OriginalTitle,
		Overview:      source.Overview,
		ReleaseDate:   source.ReleaseDate,
		PosterPath:    source.PosterPath,
		BackdropPath:  source.BackdropPath,
		VoteAverage:   source.VoteAverage,
		GenreIDs:      append([]int64(nil), source.GenreIDs...),
		Genres:        genres,
	}
}
