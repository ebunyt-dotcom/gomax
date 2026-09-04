package types

// InitData contains the query ID and URL returned for a bot web app.
type InitData struct {
	QueryID string `json:"query_id" msgpack:"query_id"`
	URL     string `json:"url" msgpack:"url"`
}
