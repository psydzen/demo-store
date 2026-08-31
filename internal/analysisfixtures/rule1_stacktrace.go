package analysisfixtures

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- HTTP ---

func stacktraceHTTPBad(w http.ResponseWriter, err error) {
	// ruleid: go-stacktrace-in-response
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func stacktraceHTTPBadFormatted(w http.ResponseWriter, err error) {
	// ruleid: go-stacktrace-in-response
	http.Error(w, fmt.Sprintf("could not load quiz: %v", err), http.StatusInternalServerError)
}

func stacktraceHTTPBadWrite(w http.ResponseWriter, err error) {
	// ruleid: go-stacktrace-in-response
	fmt.Fprintf(w, "internal error: %s", err.Error())
}

func stacktraceHTTPOK(w http.ResponseWriter, err error) {
	_ = err
	// ok: go-stacktrace-in-response
	http.Error(w, "Что-то пошло не так. Попробуйте ещё раз.", http.StatusInternalServerError)
}

// --- gRPC ---

func stacktraceGRPCBad(err error) error {
	// ruleid: go-stacktrace-in-response
	return status.Error(codes.Internal, err.Error())
}

func stacktraceGRPCBadf(err error) error {
	// ruleid: go-stacktrace-in-response
	return status.Errorf(codes.Internal, "search payments: %v", err)
}

func stacktraceGRPCOK(err error) error {
	_ = err
	// ok: go-stacktrace-in-response
	return status.Error(codes.Internal, "payment could not be stored")
}
