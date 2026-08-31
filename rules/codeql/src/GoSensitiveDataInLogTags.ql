/**
 * @name Card or personal data in a log tag
 * @description Card data or personal data reaches a log tag or a trace
 *              attribute.
 * @kind path-problem
 * @problem.severity error
 * @precision high
 * @id go/spnd/sensitive-data-in-log-tags
 * @tags security
 */

import go
import SensitiveFlow::PathGraph

/** The field and getter names that carry card or personal data. */
predicate sensitiveName(string name) {
  name =
    [
      "PAN", "Pan", "CardNumber", "CVV", "Cvv", "CVC", "Cvc", "Holder", "CardHolder", "FullName",
      "Email", "Phone", "TaxID", "TaxId", "Passport", "SSN"
    ]
}

module SensitiveConfig implements DataFlow::ConfigSig {
  predicate isSource(DataFlow::Node source) {
    exists(Field f | sensitiveName(f.getName()) | source = f.getARead())
    or
    exists(DataFlow::MethodCallNode c, string n |
      c.getTarget().getName() = "Get" + n and sensitiveName(n)
    |
      source = c.getResult()
    )
  }

  predicate isSink(DataFlow::Node sink) {
    // The value argument of logtag.With.
    exists(DataFlow::CallNode c |
      c.getTarget().hasQualifiedName("github.com/spndxyz/quiz/internal/logtag", "With") and
      sink = c.getArgument(2)
    )
    or
    // Any argument of a log call.
    exists(DataFlow::CallNode c, int i |
      (
        c.getTarget().hasQualifiedName("log/slog", ["Debug", "Info", "Warn", "Error"]) or
        c.getTarget().(Method).hasQualifiedName("log/slog", "Logger", ["Debug", "Info", "Warn", "Error"]) or
        c.getTarget()
            .(Method)
            .hasQualifiedName("github.com/spndxyz/quiz/internal/logtag", "Logger",
              ["Info", "Error"])
      ) and
      i > 0 and
      sink = c.getArgument(i)
    ) and
    not isSinkLocationExcluded(sink)
  }

  /**
   * The logger's own implementation forwards its tags to slog. Reporting there
   * repeats the caller's defect at a place no one can fix it.
   */
  additional predicate isSinkLocationExcluded(DataFlow::Node sink) {
    sink.getFile().getAbsolutePath().matches("%/internal/logtag/%")
  }

  predicate isBarrier(DataFlow::Node node) {
    exists(DataFlow::MethodCallNode c | c.getTarget().getName() = "Masked" | node = c.getResult())
    or
    exists(DataFlow::CallNode c |
      c.getTarget().getName() = "last4" and node = c.getResult(0)
    )
  }
}

module SensitiveFlow = TaintTracking::Global<SensitiveConfig>;

from SensitiveFlow::PathNode source, SensitiveFlow::PathNode sink
where SensitiveFlow::flowPath(source, sink)
select sink.getNode(), source, sink,
  "Card or personal data reaches a log tag. Log the masked form instead."
