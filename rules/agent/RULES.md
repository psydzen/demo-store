# Interprocedural review rules

Six rules. Every finding must name the rule id, the file, the line, and the
path from the source to the sink (or the entry point that is missing a call).

A rule reports a **defect in this repository's own code**. Generated files
(`internal/grpcapi/paymentspb/`) and vendored assets are out of scope.

---

## 1. `go-stacktrace-in-response`

The text of an error must not reach the client.

- **Source**: any `error` value — a returned `err`, `err.Error()`, an error
  formatted into a string with `%v`, `%s` or `%w`.
- **Sink**: `http.Error`, `fmt.Fprintf`/`fmt.Fprint` writing to an
  `http.ResponseWriter`, `status.Error`, `status.Errorf`, or any value placed
  in a gRPC response message.
- **Not a finding**: a fixed message that carries nothing from the error;
  logging the error and returning a fixed message.

## 2. `go-raw-request-to-sql-or-api`

Data taken from the request must not be spliced into an SQL statement or into
the address of an outbound call.

- **Source**: `r.URL.Query().Get`, `r.FormValue`, `r.PostFormValue`,
  `r.PathValue`, `r.Header.Get`, and the `Get…()` getters of a gRPC request
  message.
- **Sink**: the SQL string argument of `Query`, `QueryRow`, `Exec`; the URL
  argument of `http.Get`, `http.Post`, `http.NewRequest`,
  `http.NewRequestWithContext`.
- **Not a finding**: the value passed as a bind parameter (`$1`, `?`); a value
  checked against a fixed list before use; a value converted by
  `strconv.ParseInt`/`Atoi`.
- **Follow the value across functions and across packages.** A handler that
  passes the raw value to a storage method which then splices it is a finding,
  reported at the line where the splicing happens, with the handler named.

## 3. `go-logger-without-logtag-context`

Inside a function that has a request context, log through
`logtag.From(ctx)`, not through `slog` directly.

- **Finding**: a call to `slog.Info/Warn/Error/Debug`, `slog.Default().…`, or a
  method on an `*slog.Logger` value, inside a function that takes a
  `context.Context` or an `*http.Request`.
- **Not a finding**: logging at start-up or in `main`, where no request context
  exists; logging through `logtag.From(ctx)`.

## 4. `go-handler-without-metrics`

Every HTTP handler and every gRPC method must be counted in the request
metrics.

- **Entry points**: methods registered on a `grpc.Server`; functions registered
  on an `http.ServeMux`.
- **Satisfied when** `obs.Observe` is reached — either directly in the handler,
  or through an interceptor or middleware the handler is registered behind.
  **Follow the registration**: a server built with
  `grpc.NewServer(grpc.UnaryInterceptor(...))` covers everything registered on
  it; a server built with a plain `grpc.NewServer()` covers nothing.
- **Finding**: an entry point from which `obs.Observe` is not reachable.

## 5. `go-handler-without-start-end-log`

Every HTTP handler and every gRPC method must log both the start and the end of
the call.

- Same entry points and same registration-following rule as rule 4.
- **Satisfied when** both a "call started" log and a "call finished" log are
  reached, in the handler or in the interceptor/middleware it sits behind.
- **Finding**: an entry point missing either of the two.

## 6. `go-sensitive-data-in-log-tags`

Card data and personal data must not reach a log tag or a trace attribute.

- **Source**: the fields `PAN`, `CardNumber`, `CVV`, `CVC`, `Holder`,
  `FullName`, `Email`, `Phone`, `TaxID`, `Passport`, `SSN` and their protobuf
  getters.
- **Sink**: the value argument of `logtag.With`; any argument of a log call
  (`logtag.From(ctx).Info/Error`, `slog.…`); `SetAttributes` on a trace span.
- **Not a finding**: `Masked()`, `CardLast4`, a value passed through `last4`,
  or any other masked or truncated form.
- **Follow the value across functions and across packages.**
