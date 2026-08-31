/**
 * @name Logging without the request context
 * @description A log line is written through slog directly inside a function
 *              that has a request context, so it loses the request tags.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id go/spnd/logger-without-logtag-context
 * @tags maintainability
 */

import go

/** A call to a log method that bypasses logtag. */
predicate bareSlogCall(DataFlow::CallNode c) {
  c.getTarget().hasQualifiedName("log/slog", ["Debug", "Info", "Warn", "Error"])
  or
  c.getTarget().(Method).hasQualifiedName("log/slog", "Logger", ["Debug", "Info", "Warn", "Error"])
}

/** A function that has access to a request context. */
predicate hasRequestContext(FuncDef f) {
  exists(Parameter p | p.getFunction() = f |
    p.getType().getName() = "Context" or
    p.getType().(PointerType).getBaseType().getName() = "Request"
  )
}

from DataFlow::CallNode c, FuncDef f
where
  bareSlogCall(c) and
  f = c.getRoot() and
  hasRequestContext(f) and
  not c.getFile().getBaseName().matches("%.pb.go")
select c,
  "This log line bypasses logtag, so it loses the request tags. Use logtag.From(ctx) instead."
