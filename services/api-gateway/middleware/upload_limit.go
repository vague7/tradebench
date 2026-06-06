package middleware

import "net/http"

func UploadLimit(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}

func WriteAPIError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + escapeJSON(message) + `","code":"` + escapeJSON(code) + `"}`))
}

func escapeJSON(value string) string {
	escaped := make([]byte, 0, len(value))
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '\\':
			escaped = append(escaped, '\\', '\\')
		case '"':
			escaped = append(escaped, '\\', '"')
		case '\n':
			escaped = append(escaped, '\\', 'n')
		case '\r':
			escaped = append(escaped, '\\', 'r')
		case '\t':
			escaped = append(escaped, '\\', 't')
		default:
			escaped = append(escaped, value[i])
		}
	}
	return string(escaped)
}
