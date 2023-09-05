package gocode

import (
	"fmt"
	"strings"

	"gitlab.mpi-sws.org/cld/blueprint/blueprint/pkg/core/service"
)

/*
Basic structs used by IR nodes to describe Golang service interfaces

These structs implement the generic interfaces described in the core 'service' package

TypeName is defined separately in typename.go
*/

/* Module, version, and package that contains a definition */
type Source struct {
	ModuleName    string
	ModuleVersion string
	PackageName   string
}

type Variable struct {
	service.Variable
	Name string
	Type TypeName
}

type Func struct {
	service.Method
	Name      string
	Arguments []Variable
	Returns   []Variable
}

type Constructor struct {
	Func
	Source
}

type ServiceInterface struct {
	service.ServiceInterface
	UserType // Has a Name and a Source location
	Methods  map[string]Func
}

func (s *ServiceInterface) GetName() string {
	return s.UserType.Name
}

func (s *ServiceInterface) GetMethods() []service.Method {
	var methods []service.Method
	for _, method := range s.Methods {
		methods = append(methods, &method)
	}
	return methods
}

func (f *Func) GetName() string {
	return f.Name
}

func (f *Func) GetArguments() []service.Variable {
	var variables []service.Variable
	for _, variable := range f.Arguments {
		variables = append(variables, &variable)
	}
	return variables
}

func (f *Func) GetReturns() []service.Variable {
	var variables []service.Variable
	for _, variable := range f.Returns {
		variables = append(variables, &variable)
	}
	return variables
}

func (v *Variable) GetName() string {
	return v.Name
}

func (v *Variable) GetType() string {
	return v.Type.String()
}

func (v *Variable) String() string {
	if v.Name == "" {
		return v.Type.String()
	} else {
		return fmt.Sprintf("%v %v", v.Name, v.Type)
	}
}

func (f Func) String() string {
	var arglist []string
	for _, arg := range f.Arguments {
		arglist = append(arglist, arg.String())
	}
	args := strings.Join(arglist, ", ")
	var retlist []string
	for _, ret := range f.Returns {
		retlist = append(retlist, ret.String())
	}
	rets := strings.Join(retlist, ", ")
	if len(f.Returns) > 1 {
		return fmt.Sprintf("func %v(%v) (%v)", f.Name, args, rets)
	} else if len(f.Returns) == 1 {
		return fmt.Sprintf("func %v(%v) %v", f.Name, args, rets)
	} else {
		return fmt.Sprintf("func %v(%v)", f.Name, args)
	}
}
