package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	call := serving(t, &fakeGitHub{})

	t.Run("preflight from the site", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "https://chessdocs.org")
		resp := call(req)

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://chessdocs.org" {
			t.Errorf("Allow-Origin = %q", got)
		}
	})

	t.Run("another origin is not allowed", func(t *testing.T) {
		req := get(t, "en/glossary/blockade.md")
		req.Header.Set("Origin", "https://evil.example")
		resp := call(req)

		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Allow-Origin = %q, want it withheld", got)
		}
	})
}

func TestUnroutedRequestsAnswerInTheErrorShape(t *testing.T) {
	call := serving(t, &fakeGitHub{})

	t.Run("unknown path", func(t *testing.T) {
		resp := call(httptest.NewRequest(http.MethodGet, "/elsewhere", nil))
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", resp.StatusCode)
		}
		errorMessage(t, resp)
	})

	t.Run("unsupported method", func(t *testing.T) {
		resp := call(httptest.NewRequest(http.MethodDelete, "/", nil))
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", resp.StatusCode)
		}
		if resp.Header.Get("Allow") == "" {
			t.Error("a 405 should advertise the allowed methods")
		}
		errorMessage(t, resp)
	})
}
