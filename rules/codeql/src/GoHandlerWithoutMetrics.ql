/**
 * @name Request handler reports no metrics
 * @description obs.Observe is not reachable from this handler, so the call is
 *              missing from the request metrics.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id go/spnd/handler-without-metrics
 * @tags maintainability
 */

import go
import Handlers

from FuncDecl entry, string kind
where
  entryPoint(entry, kind) and
  not reachesObserve(entry)
select entry, kind + " reports no metrics: obs.Observe is not reachable from it."
