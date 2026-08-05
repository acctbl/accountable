function handler(event) {
	var request = event.request;
	if (request.uri === "/api") {
		request.uri = "/";
	} else if (request.uri.startsWith("/api/")) {
		request.uri = request.uri.substring(4);
	}
	return request;
}
