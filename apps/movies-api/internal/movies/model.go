package movies

type Movie struct {
	ID            int64
	Title         string
	OriginalTitle string
	Overview      string
	ReleaseDate   string
	PosterPath    *string
	BackdropPath  *string
	VoteAverage   float64
	GenreIDs      []int64
	Genres        []Genre
}

type Genre struct {
	ID   int64
	Name string
}

type Page struct {
	Number     int
	TotalPages int
	Movies     []Movie
}
