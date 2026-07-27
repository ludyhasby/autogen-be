package response

type ListNewsResp struct {
	List       []ListNews `json:"list"`
	TotalItems int32      `json:"total_items"`
	TotalPages int32      `json:"total_pages"`
	Page       int32      `json:"page"`
	PageSize   int32      `json:"page_size"`
}
type ListNews struct {
	Source      string `json:"source"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Url         string `json:"url"`
	UrlToImage  string `json:"url_to_image"`
	PublishedAt string `json:"published_at"`
}

type NewsFetchResponse struct {
	Status       string `json:"status"`
	TotalResults int    `json:"totalResults"`
	Articles     []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		URLToImage  string `json:"urlToImage"`
		PublishedAt string `json:"publishedAt"`
		Source      struct {
			ID   int32  `json:"id"`
			Name string `json:"name"`
		}
	} `json:"articles"`
}
