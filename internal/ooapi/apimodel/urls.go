package apimodel

// URLsRequest is the URLs request.
type URLsRequest struct {
	CategoryCodes string `query:"category_codes"`
	CountryCode   string `query:"country_code"`
	Limit         int64  `query:"limit"`
}

// URLsResponse is the URLs response.
type URLsResponse struct {
	Metadata URLsMetadata      `json:"metadata"`
	Results  []URLsResponseURL `json:"results"`
}

// URLsMetadata contains metadata in the URLs response.
type URLsMetadata struct {
	Count int64 `json:"count"`
}

// URLsResponseURL is a single URL in the URLs response.
type URLsResponseURL struct {
	CategoryCode string `json:"category_code"`
	CountryCode  string `json:"country_code"`
	URL          string `json:"url"`
}
