package projectPage

// PlayTokenHTTP is returned by GET /projectPage/:id/play-token.
type PlayTokenHTTP struct {
	PlayURL   string `json:"playUrl"`
	JSONURL   string `json:"jsonUrl"`
	ExpiresAt string `json:"expiresAt"`
}
