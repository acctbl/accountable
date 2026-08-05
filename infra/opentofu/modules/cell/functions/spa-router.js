function handler(event) {
	var request = event.request;
	var uri = request.uri;
	if (uri.startsWith("/assets/") || uri.includes(".")) {
		return request;
	}
	request.uri = "/index.html";
	return request;
}
