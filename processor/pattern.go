package processor

import "go/ast"

type TracePattern struct {
	ContextName    string
	ContextPackage string
	ContextType    string
	ErrorName      string
	ErrorType      string
}

// BasicSpanName is common notation of <class>.<method> or <pkg>.<func>
func BasicSpanName(receiver, function string) string {
	if receiver == "" {
		return function
	}
	return receiver + "." + function
}

func methodReceiverTypeName(fn *ast.FuncDecl) string {
	// function
	if fn == nil || fn.Recv == nil {
		return ""
	}
	// method
	for _, v := range fn.Recv.List {
		if v == nil {
			continue
		}
		t := v.Type
		// pointer receiver
		if v, ok := v.Type.(*ast.StarExpr); ok {
			t = v.X
		}
		// value/pointer receiver
		if v, ok := t.(*ast.Ident); ok {
			return v.Name
		}
	}
	return ""
}

func functionName(fn *ast.FuncDecl) string {
	if fn == nil || fn.Name == nil {
		return ""
	}
	return fn.Name.Name
}

func isContext(e *ast.Field, contextName, contextPackage, contextType string) bool {
	// anonymous arg
	// multiple symbols
	// strange symbol
	if e == nil || len(e.Names) != 1 || e.Names[0] == nil {
		return false
	}
	if e.Names[0].Name != contextName {
		return false
	}

	pkg := ""
	sym := ""

	if se, ok := e.Type.(*ast.SelectorExpr); ok && se != nil {
		if v, ok := se.X.(*ast.Ident); ok && v != nil {
			pkg = v.Name
		}
		if v := se.Sel; v != nil {
			sym = v.Name
		}
	}

	return pkg == contextPackage && sym == contextType
}

func isError(e *ast.Field, errorName, errorType string) bool {
	if e == nil {
		return false
	}
	// anonymous arg
	// multiple symbols
	// strange symbol
	if len(e.Names) != 1 || e.Names[0] == nil {
		return false
	}
	if e.Names[0].Name != errorName {
		return false
	}

	if v, ok := e.Type.(*ast.Ident); ok && v != nil {
		return v.Name == errorType
	}

	return false
}

func functionHasContext(fnType *ast.FuncType, contextName, contextPackage, contextType string) bool {
	if fnType == nil {
		return false
	}

	if ps := fnType.Params; ps != nil {
		for _, q := range ps.List {
			if isContext(q, contextName, contextPackage, contextType) {
				return true
			}
		}
	}

	return false
}

func functionHasError(fnType *ast.FuncType, errorName, errorType string) bool {
	if fnType == nil {
		return false
	}

	if rs := fnType.Results; rs != nil {
		for _, q := range rs.List {
			if isError(q, errorName, errorType) {
				return true
			}
		}
	}

	return false
}
