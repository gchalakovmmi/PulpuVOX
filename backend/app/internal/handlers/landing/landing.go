package landing

import (
    "net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>PulpuVOX</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
</head>
<body>
    <div class="container text-center mt-5">
        <h1>Welcome to PulpuVOX</h1>
        <p>Your English learning companion</p>
        <a href="/auth/google" class="btn btn-primary">Sign in with Google</a>
    </div>
</body>
</html>
    `))
}
