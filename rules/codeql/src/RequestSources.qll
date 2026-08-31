/** Values that come straight off an incoming HTTP or gRPC request. */

import go

/** A value read from the incoming request without any validation. */
DataFlow::Node rawRequestValue() {
  // r.URL.Query().Get(...), r.FormValue(...), r.PostFormValue(...),
  // r.PathValue(...), r.Header.Get(...)
  exists(DataFlow::MethodCallNode c |
    c.getTarget().hasQualifiedName("net/url", "Values", "Get") or
    c.getTarget().hasQualifiedName("net/http", "Request", ["FormValue", "PostFormValue", "PathValue"]) or
    c.getTarget().hasQualifiedName("net/textproto", "MIMEHeader", "Get") or
    c.getTarget().hasQualifiedName("net/http", "Header", "Get")
  |
    result = c.getResult()
  )
  or
  // Protobuf getters on a generated request message.
  exists(DataFlow::MethodCallNode c |
    c.getTarget().getName().matches("Get%") and
    c.getTarget().getReceiverType().getName().matches("%Request") and
    result = c.getResult()
  )
}
