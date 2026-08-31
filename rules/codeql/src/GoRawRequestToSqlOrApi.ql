/**
 * @name Raw request data reaches SQL or an outbound call
 * @description Data taken straight from the request is spliced into an SQL
 *              statement or into the address of an outbound call.
 * @kind path-problem
 * @problem.severity error
 * @precision high
 * @id go/spnd/raw-request-to-sql-or-api
 * @tags security
 */

import go
import RequestSources
import RawRequestFlow::PathGraph

module RawRequestConfig implements DataFlow::ConfigSig {
  predicate isSource(DataFlow::Node source) { source = rawRequestValue() }

  predicate isSink(DataFlow::Node sink) {
    // The statement argument of a pgx query. Bind parameters are later
    // arguments and are therefore not sinks.
    exists(DataFlow::MethodCallNode c |
      c.getTarget().getName() = ["Query", "QueryRow", "Exec"] and
      sink = c.getArgument(1)
    )
    or
    // The address of an outbound call.
    exists(DataFlow::CallNode c |
      c.getTarget().hasQualifiedName("net/http", ["Get", "Post", "Head"]) and
      sink = c.getArgument(0)
    )
    or
    exists(DataFlow::CallNode c |
      c.getTarget().hasQualifiedName("net/http", "NewRequest") and sink = c.getArgument(1)
    )
    or
    exists(DataFlow::CallNode c |
      c.getTarget().hasQualifiedName("net/http", "NewRequestWithContext") and
      sink = c.getArgument(2)
    )
  }

  predicate isBarrier(DataFlow::Node node) {
    exists(DataFlow::CallNode c |
      c.getTarget().hasQualifiedName("strconv", ["Atoi", "ParseInt", "ParseFloat"]) or
      c.getTarget().hasQualifiedName("net/url", "QueryEscape")
    |
      node = c.getResult(0)
    )
  }
}

module RawRequestFlow = TaintTracking::Global<RawRequestConfig>;

from RawRequestFlow::PathNode source, RawRequestFlow::PathNode sink
where RawRequestFlow::flowPath(source, sink)
select sink.getNode(), source, sink,
  "Data taken straight from the request is spliced into an SQL statement or an outbound address."
