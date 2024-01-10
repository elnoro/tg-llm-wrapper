package http

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
)

type DebugRoundTripper struct {
	Proxied http.RoundTripper
}

func (lrt DebugRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	requestDump, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		slog.Error("dumping request failed", slog.String("error", err.Error()))
	} else {
		slog.Info("request", slog.String("request", string(requestDump)))
	}

	response, err := lrt.Proxied.RoundTrip(r)
	if err != nil {
		slog.Error("sending request failed", slog.String("error", err.Error()))

		return nil, err
	}

	responseDump, err := httputil.DumpResponse(response, true)
	if err != nil {
		slog.Error("dumping response failed", slog.String("error", err.Error()))
	} else {
		slog.Info("response", slog.String("response", string(responseDump)))
	}

	return response, err
}
