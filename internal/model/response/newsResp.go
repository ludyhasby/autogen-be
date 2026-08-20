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
	Status string `json:"status"`
	Total  int    `json:"total"`
	Data   []struct {
		Title       string `json:"strjudul"`
		Description string `json:"strringkasan"`
		URL         string `json:"url"`
		Image       []struct {
			RawUrlImage string `json:"strnmfile"`
			Extension   string `json:"extension"`
		} `json:"image"`
		NewsDate string `json:"dtnewsdate"`
	} `json:"data"`
}
