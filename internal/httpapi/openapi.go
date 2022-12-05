package httpapi

//
// OpenAPI
//

// OpenAPISchema is the schema of a specific parameter or
// or the schema used by the response body
type OpenAPISchema struct {
	Properties map[string]*OpenAPISchema `json:"properties,omitempty"`
	Items      *OpenAPISchema            `json:"items,omitempty"`
	Type       string                    `json:"type"`
}

// OpenAPIParameter describes an input parameter, which could be in the
// URL path, in the query string, or in the request body
type OpenAPIParameter struct {
	In       string         `json:"in"`
	Name     string         `json:"name"`
	Required bool           `json:"required,omitempty"`
	Schema   *OpenAPISchema `json:"schema,omitempty"`
	Type     string         `json:"type,omitempty"`
}

// OpenAPIBody describes a response body
type OpenAPIBody struct {
	Description interface{}    `json:"description,omitempty"`
	Schema      *OpenAPISchema `json:"schema"`
}

// OpenAPIResponses describes the possible responses
type OpenAPIResponses struct {
	Successful OpenAPIBody `json:"200"`
}

// OpenAPIRoundTrip describes an HTTP round trip with a given method and path
type OpenAPIRoundTrip struct {
	Consumes   []string            `json:"consumes,omitempty"`
	Produces   []string            `json:"produces,omitempty"`
	Parameters []*OpenAPIParameter `json:"parameters,omitempty"`
	Responses  *OpenAPIResponses   `json:"responses,omitempty"`
}

// OpenAPIPath describes a path served by the API
type OpenAPIPath struct {
	Get  *OpenAPIRoundTrip `json:"get,omitempty"`
	Post *OpenAPIRoundTrip `json:"post,omitempty"`
}

// OpenAPIInfo contains info about the OpenAPIInfo
type OpenAPIInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

// OpenAPISwagger is the toplevel structure
type OpenAPISwagger struct {
	Swagger  string                  `json:"swagger"`
	Info     OpenAPIInfo             `json:"info"`
	Host     string                  `json:"host"`
	BasePath string                  `json:"basePath"`
	Schemes  []string                `json:"schemes"`
	Paths    map[string]*OpenAPIPath `json:"paths"`
}
