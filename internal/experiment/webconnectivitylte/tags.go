package webconnectivitylte

import "fmt"

// generateTagsForEndpoints generates the tags for the endpoints.
func generateTagsForEndpoints(depth int64, ps *prioritySelector, classic bool) (output []string) {
	// The classic flag marks all observations using IP addresses
	// fetched using the system resolver. Strictly speaking classic
	// means that these measurements derive from the resolver that
	// we consider primary, and for us it it the system one.
	if classic {
		output = append(output, "classic")
	}

	// The depth=0|1|... tag indicates the current redirect depth.
	//
	// When the depth is zero, we also include the tcptls_experiment tag
	// for backwards compatibility with Web Connectivity v0.4.
	if depth < 1 {
		output = append(output, "tcptls_experiment")
	}
	output = append(output, fmt.Sprintf("depth=%d", depth))

	// The fetch_body=true|false tag allows to distinguish between observations
	// with the objective of fetching the body and extra observations. For example,
	// for http:// requests we perform TLS handshakes for the purpose of checking
	// whether IP addresses are valid without fetching the body.
	output = append(output, fmt.Sprintf("fetch_body=%v", ps != nil))

	return output
}
