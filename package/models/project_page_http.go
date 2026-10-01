package models

// ProjectPageHTTP is bound to the GraphQL ProjectPageHttp type in gqlgen.yml (not generated):
// landing and author fields are served by REST only and are not in the schema.
type ProjectPageHTTP struct {
	ProjectPageID    string   `json:"projectPageId"`
	LastModified     string   `json:"lastModified"`
	ProjectID        string   `json:"projectId"`
	Instruction      string   `json:"instruction"`
	Notes            string   `json:"notes"`
	Preview          string   `json:"preview"`
	LinkScratch      string   `json:"linkScratch"`
	Title            string   `json:"title"`
	IsShared         bool     `json:"isShared"`
	LandingFeatured  bool     `json:"landingFeatured"`
	LandingSortOrder int      `json:"landingSortOrder"`
	Tags             []string `json:"tags"`
	AuthorUserID     string   `json:"authorUserId"`
	AuthorName       string   `json:"authorName"`
	AuthorAvatarID   string   `json:"authorAvatarId,omitempty"`
	IsOwner          bool     `json:"isOwner"`
	ReactionCount    int64    `json:"reactionCount"`
}

func (ProjectPageHTTP) IsProjectPageResult() {}
