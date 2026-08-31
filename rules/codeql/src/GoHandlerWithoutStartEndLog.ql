/**
 * @name Request handler does not log start and end
 * @description Neither the handler nor the interceptor it sits behind logs
 *              both the start and the end of the call.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id go/spnd/handler-without-start-end-log
 * @tags maintainability
 */

import go
import Handlers

from FuncDecl entry, string kind
where
  entryPoint(entry, kind) and
  not (reachesLog(entry, "call started") and reachesLog(entry, "call finished"))
select entry, kind + " does not log both the start and the end of the call."
