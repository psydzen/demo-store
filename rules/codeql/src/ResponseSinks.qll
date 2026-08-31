/** Places a value becomes visible to the caller of an HTTP or gRPC endpoint. */

import go

/** A value written into an HTTP response or a gRPC status. */
DataFlow::Node clientVisibleSink() {
  // http.Error(w, msg, code)
  exists(DataFlow::CallNode c |
    c.getTarget().hasQualifiedName("net/http", "Error") and
    result = c.getArgument(1)
  )
  or
  // fmt.Fprintf(w, ...) / fmt.Fprint(w, ...) where w is an http.ResponseWriter
  exists(DataFlow::CallNode c, int i |
    c.getTarget().hasQualifiedName("fmt", ["Fprintf", "Fprint", "Fprintln"]) and
    c.getArgument(0).getType().getUnderlyingType() =
      any(Type t | t.hasQualifiedName("net/http", "ResponseWriter")).getUnderlyingType() and
    i > 0 and
    result = c.getArgument(i)
  )
  or
  // status.Error(code, msg) / status.Errorf(code, format, args...)
  exists(DataFlow::CallNode c, int i |
    c.getTarget().hasQualifiedName("google.golang.org/grpc/status", ["Error", "Errorf"]) and
    i > 0 and
    result = c.getArgument(i)
  )
}
