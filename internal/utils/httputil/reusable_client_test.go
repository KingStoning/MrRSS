package httputil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReusableClientKeepsConnectionsAndReplacesChangedConfiguration(t *testing.T) {
	var pool ReusableClient
	var addresses []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addresses = append(addresses, r.RemoteAddr)
		w.Write([]byte("image"))
	}))
	defer server.Close()
	for i := 0; i < 3; i++ {
		client, err := pool.Get(nil, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		res, err := client.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
	}
	if addresses[0] != addresses[1] || addresses[1] != addresses[2] {
		t.Fatal("image requests did not reuse their connection")
	}
	old, _ := pool.Get(nil, time.Second)
	t.Setenv(InsecureSkipTLSVerifyEnv, "true")
	updated, err := pool.Get(nil, time.Second)
	if err != nil || old == updated {
		t.Fatal("TLS setting change must replace the transport")
	}
	updated.CloseIdleConnections()
}
