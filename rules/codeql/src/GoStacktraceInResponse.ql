/**
 * @name Error text reaches the client
 * @description The text of an error is written to an HTTP response or returned
 *              in a gRPC status. Errors carry file paths, SQL fragments and
 *              wrapped internals.
 * @kind path-problem
 * @problem.severity error
 * @precision high
 * @id go/spnd/stacktrace-in-response
 * @tags security
 */

import go
import ResponseSinks
import StacktraceFlow::PathGraph

/** Any value whose type is the built-in `error` interface. */
predicate isErrorValue(DataFlow::Node node) {
  node.getType().getUnderlyingType() = Builtin::error().getType().getUnderlyingType()
}

module StacktraceConfig implements DataFlow::ConfigSig {
  predicate isSource(DataFlow::Node source) {
    isErrorValue(source)
    or
    source = any(DataFlow::MethodCallNode c | c.getTarget().getName() = "Error")
  }

  predicate isSink(DataFlow::Node sink) { sink = clientVisibleSink() }
}

module StacktraceFlow = TaintTracking::Global<StacktraceConfig>;

from StacktraceFlow::PathNode source, StacktraceFlow::PathNode sink
where StacktraceFlow::flowPath(source, sink)
select sink.getNode(), source, sink,
  "The text of an error reaches the client; log it and return a fixed message instead."
