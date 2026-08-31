/**
 * Request entry points, and what each one can reach.
 *
 * This is the part a single-file engine cannot express. An entry point
 * satisfies the metrics and the logging rule when the required call is
 * reachable from it through the call graph — including through the interceptor
 * of the server the entry point was registered on.
 */

import go

/** A gRPC service implementation type, as passed to a generated Register call. */
Type registeredServiceType(DataFlow::CallNode register) {
  register.getTarget().getName().matches("Register%Server") and
  result = register.getArgument(1).getType().(PointerType).getBaseType()
}

/** A gRPC unary method on a type that is actually registered on a server. */
predicate grpcMethod(FuncDecl fd, string kind) {
  exists(Method m, DataFlow::CallNode register |
    m = fd.getFunction() and
    m.getReceiverBaseType() = registeredServiceType(register) and
    m.getNumParameter() = 2 and
    m.getParameterType(0).getName() = "Context" and
    m.getNumResult() = 2 and
    m.getResultType(1).getName() = "error" and
    not m.getName().matches("mustEmbed%") and
    not fd.getFile().getBaseName().matches("%.pb.go") and
    kind = "gRPC method " + m.getName()
  )
}

/** A net/http handler function. */
predicate httpHandler(FuncDecl fd, string kind) {
  exists(Function f |
    f = fd.getFunction() and
    f.getNumParameter() = 2 and
    f.getParameterType(0).getName() = "ResponseWriter" and
    f.getParameterType(1).(PointerType).getBaseType().getName() = "Request" and
    f.getNumResult() = 0 and
    not fd.getFile().getBaseName().matches("%.pb.go") and
    kind = "HTTP handler " + f.getName()
  )
}

/** A request entry point. */
predicate entryPoint(FuncDecl fd, string kind) { grpcMethod(fd, kind) or httpHandler(fd, kind) }

/** `inner` is `outer` itself, or a function literal written inside it. */
predicate nested(FuncDef outer, FuncDef inner) {
  inner = outer
  or
  exists(FuncLit lit | lit.getEnclosingFunction+() = outer and inner = lit)
}

/** A function whose body is reachable from `caller`, literals included. */
predicate reachable(FuncDef caller, FuncDef callee) {
  exists(DataFlow::CallNode c, FuncDef root |
    nested(caller, root) and c.getRoot() = root and callee = c.getTarget().getFuncDecl()
  )
  or
  exists(FuncDef mid | reachable(caller, mid) and reachable(mid, callee))
}

/** `body` is the entry point itself, something it calls, or its interceptor. */
predicate inScopeOf(FuncDecl entry, FuncDef body) {
  nested(entry, body)
  or
  exists(FuncDef callee | reachable(entry, callee) and nested(callee, body))
  or
  exists(FuncDef interceptor, FuncDef callee |
    behindInterceptor(entry, interceptor) and
    (callee = interceptor or reachable(interceptor, callee)) and
    nested(callee, body)
  )
}

/** The interceptor a gRPC entry point sits behind, if there is one. */
predicate behindInterceptor(FuncDecl entry, FuncDef interceptor) {
  exists(DataFlow::CallNode register, DataFlow::CallNode newServer, DataFlow::CallNode option |
    // The entry point belongs to the service this call registers.
    entry.getFunction().(Method).getReceiverBaseType() = registeredServiceType(register) and
    newServer.getTarget().hasQualifiedName("google.golang.org/grpc", "NewServer") and
    // ... on the server this call builds ...
    DataFlow::localFlow(newServer.getResult(0), register.getArgument(0)) and
    // ... which was built with a unary interceptor option.
    option
        .getTarget()
        .hasQualifiedName("google.golang.org/grpc", ["UnaryInterceptor", "ChainUnaryInterceptor"]) and
    option.asExpr().getParent*() = newServer.asExpr() and
    // The interceptor is whatever built the value the option was given.
    exists(DataFlow::CallNode build |
      build.asExpr().getParent*() = option.asExpr() and
      interceptor = build.getTarget().getFuncDecl()
    )
  )
}

/** `obs.Observe` is reached from the entry point or from its interceptor. */
predicate reachesObserve(FuncDecl entry) {
  exists(DataFlow::CallNode c, FuncDef body |
    c.getTarget().hasQualifiedName("github.com/spndxyz/quiz/internal/obs", "Observe") and
    inScopeOf(entry, body) and
    c.getRoot() = body
  )
}

/** A log call carrying `msg` is reached from the entry point or its interceptor. */
predicate reachesLog(FuncDecl entry, string msg) {
  exists(DataFlow::CallNode c, FuncDef body |
    c.getTarget().getName() = ["Info", "Debug", "Warn", "Error"] and
    c.getArgument(0).getStringValue() = msg and
    inScopeOf(entry, body) and
    c.getRoot() = body
  )
}
