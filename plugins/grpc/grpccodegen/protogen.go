package grpccodegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gitlab.mpi-sws.org/cld/blueprint/plugins/golang"
	"gitlab.mpi-sws.org/cld/blueprint/plugins/golang/gocode"
	"gitlab.mpi-sws.org/cld/blueprint/plugins/golang/goparser"
)

func GenerateGRPCProto(builder golang.ModuleBuilder, service *gocode.ServiceInterface, outputPackage string) error {
	// Re-parse all of the modules, which can include generated code from other plugins
	modules, err := goparser.ParseWorkspace(builder.Workspace().Path())
	if err != nil {
		return err
	}

	// Construct and validate the GRPC proto builder for the service
	pb := NewProtoBuilder(modules)
	err = pb.AddService(service)
	if err != nil {
		return err
	}

	// Filename munging
	splits := strings.Split(outputPackage, "/")
	outputPackageName := splits[len(splits)-1]
	pb.Package = outputPackageName
	pb.GoPackage = outputPackage

	outputDir := filepath.Join(builder.Path(), filepath.Join(splits...))
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("unable to create grpc output dir %v due to %v", outputDir, err.Error())
	}

	outputFilename := service.Name + ".proto"
	return pb.WriteProtoFile(filepath.Join(outputDir, outputFilename))
}

type GRPCField struct {
	Type     string // The GRPC type
	Name     string
	Position int
}

type GRPCMessageDecl struct {
	Builder   *GRPCProtoBuilder
	Name      string
	FieldList []*GRPCField
}

type GRPCMethodDecl struct {
	Service  *GRPCServiceDecl
	Name     string
	Request  *GRPCMessageDecl
	Response *GRPCMessageDecl
}

type GRPCServiceDecl struct {
	Builder *GRPCProtoBuilder
	Name    string
	Methods map[string]*GRPCMethodDecl
}

type GRPCProtoBuilder struct {
	Code      *goparser.ParsedModuleSet
	Package   string
	GoPackage string
	Services  map[string]*GRPCServiceDecl
	Messages  map[string]*GRPCMessageDecl
	Structs   map[gocode.UserType]*GRPCMessageDecl // Mapping from golang struct to the corresponding message
}

func NewProtoBuilder(code *goparser.ParsedModuleSet) *GRPCProtoBuilder {
	b := &GRPCProtoBuilder{}
	b.Code = code
	b.Services = make(map[string]*GRPCServiceDecl)
	b.Messages = make(map[string]*GRPCMessageDecl)
	b.Structs = make(map[gocode.UserType]*GRPCMessageDecl)
	return b
}

func (b *GRPCProtoBuilder) newMessage(name string) *GRPCMessageDecl {
	s := &GRPCMessageDecl{}
	s.Builder = b
	s.Name = name
	s.FieldList = nil
	b.Messages[name] = s // Not implemented yet: name collision possible for same-named struct from different packages
	return s
}

func (b *GRPCProtoBuilder) newService(name string) *GRPCServiceDecl {
	s := &GRPCServiceDecl{}
	s.Builder = b
	s.Name = name
	s.Methods = make(map[string]*GRPCMethodDecl)
	b.Services[name] = s // Not implemented yet: name collision possible for same-named struct from different packages
	return s
}

func (b *GRPCServiceDecl) newMethod(name string) *GRPCMethodDecl {
	m := &GRPCMethodDecl{}
	m.Service = b
	m.Name = name
	m.Request = b.Builder.newMessage(fmt.Sprintf("%s_%s_Request", b.Name, name))
	m.Response = b.Builder.newMessage(fmt.Sprintf("%s_%s_Response", b.Name, name))
	b.Methods[name] = m
	return m
}

/*
Adds a service declaration for the provided golang service interface.

# This will create message and service definitions within the grpc proto

For arguments and return values on methods in the interface, corresponding GRPC message objects
are needed.  The ProtoBuilder will consult the parsed code to find the definitions of arguments
and return values.
*/
func (b *GRPCProtoBuilder) AddService(iface *gocode.ServiceInterface) error {
	serviceDecl := b.newService(iface.Name) // TODO: (not implemented yet) possibility of name collisions
	for _, method := range iface.Methods {
		methodDecl := serviceDecl.newMethod(method.Name)

		// Expect first argument to be context.Context, and last return value to be error
		if len(method.Arguments) == 0 || method.Arguments[0].Type.String() != "context.Context" {
			return fmt.Errorf("invalid method %v.%v due to missing context.Context argument", iface.Name, method.Name)
		}
		if len(method.Returns) == 0 || method.Returns[len(method.Returns)-1].Type.String() != "error" {
			return fmt.Errorf("invalid method %v.%v due to missing error return value", iface.Name, method.Name)
		}

		req := methodDecl.Request
		for i, arg := range method.Arguments[1:] {
			argType := arg.Type
			if ptrType, isPtrType := argType.(*gocode.Pointer); isPtrType {
				// Pointer arguments are allowed
				argType = ptrType.PointerTo
			}

			grpcType, err := b.getGRPCType(argType)
			if err != nil {
				return fmt.Errorf("cannot serialize %v.%v argument %v for GRPC due to %v", iface.Name, method.Name, arg.Name, err.Error())
			}

			req.FieldList = append(req.FieldList, &GRPCField{
				Type:     grpcType,
				Name:     arg.Name,
				Position: i,
			})
		}

		rsp := methodDecl.Response
		for i, ret := range method.Returns[:len(method.Returns)-1] {
			retType := ret.Type
			if ptrType, isPtrType := retType.(*gocode.Pointer); isPtrType {
				// Pointer retvals are allowed
				retType = ptrType.PointerTo
			}

			grpcType, err := b.getGRPCType(retType)
			if err != nil {
				return fmt.Errorf("cannot serialize %v.%v retval %v for GRPC due to %v", iface.Name, method.Name, ret.Name, err.Error())
			}

			rsp.FieldList = append(rsp.FieldList, &GRPCField{
				Type:     grpcType,
				Name:     ret.Name,
				Position: i,
			})
		}
	}
	return nil
}

func (b *GRPCProtoBuilder) GetOrAddMessage(t *gocode.UserType) (*GRPCMessageDecl, error) {
	// Message might already exist
	if msgDecl, exists := b.Structs[*t]; exists {
		return msgDecl, nil
	}

	for _, mod := range b.Code.Modules {
		fmt.Println("module: " + mod.Name)
	}

	// Find the struct definition in the module
	mod, hasModule := b.Code.Modules[t.ModuleName]
	if !hasModule {
		return nil, fmt.Errorf("could not find module containing %v, expected %v", t.String(), t.ModuleName)
	}
	pkg, hasPackage := mod.Packages[t.PackageName]
	if !hasPackage {
		return nil, fmt.Errorf("could not find package containing %v, expected %v", t.String(), t.PackageName)
	}
	struc, hasStruct := pkg.Structs[t.Name]
	if !hasStruct {
		// It's possible that the type does exist but it wasn't declared as a struct, e.g. it is
		// an enum or a type alias. Non-struct types are not-yet-implemented
		if _, hasTypeDef := pkg.DeclaredTypes[t.Name]; hasTypeDef {
			return nil, fmt.Errorf("expected %v to be a struct but it is an unsupported type", t.String())
		} else {
			return nil, fmt.Errorf("could not find %v within %v", t.Name, t.PackageName)
		}
	}

	// Create the message
	// TODO (not implemented yet): edge-case name collision for same-named struct from different packages
	msg := b.newMessage(t.Name)
	b.Structs[*t] = msg
	for _, field := range struc.FieldsList {
		// We ignore promoted and anonymous struct / interface extensions
		if _, isNamed := struc.Fields[field.Name]; !isNamed {
			// TODO (not implemented yet): support promoted and anonymous, handle interfaces and promoted struct fields
			continue
		}

		// Gets the type name of this field, possibly internally creating the GRPC message if it's a struct
		fieldType, err := b.getGRPCType(field.Type)
		if err != nil {
			return nil, err
		}

		msg.FieldList = append(msg.FieldList, &GRPCField{
			Type:     fieldType,
			Name:     field.Name,
			Position: field.Position,
		})
	}

	return msg, nil
}

var basicToGrpc = map[string]string{
	"bool":   "bool",
	"string": "string",
	"int":    "sint64", "int8": "sint32", "int16": "sint32", "int32": "sint32", "int64": "sint64",
	"uint": "uint64", "uint8": "uint32", "uint16": "uint32", "uint32": "uint32", "uint64": "uint64",
	"byte":    "uint8",
	"rune":    "uint8",
	"float32": "float", "float64": "double",
}

var acceptableMapKeys map[string]struct{}

func getMapKeyType(t gocode.TypeName) (string, bool) {
	if acceptableMapKeys == nil {
		keys := []string{
			"int32", "int64", "uint32", "uint64", "sint32", "sint64",
			"fixed32", "fixed64", "sfixed32", "sfixed64", "bool", "string",
		}
		acceptableMapKeys = make(map[string]struct{})
		for _, key := range keys {
			acceptableMapKeys[key] = struct{}{}
		}
	}
	if basic, isBasic := t.(*gocode.BasicType); isBasic {
		if grpcType, hasGrpcType := basicToGrpc[basic.Name]; hasGrpcType {
			if _, isValid := acceptableMapKeys[grpcType]; isValid {
				return grpcType, true
			}
		}
	}
	return "", false
}

func (b *GRPCProtoBuilder) getGRPCType(t gocode.TypeName) (string, error) {
	switch arg := t.(type) {
	case *gocode.UserType:
		{
			msg, err := b.GetOrAddMessage(arg)
			if err != nil {
				return "", err
			}
			return msg.Name, nil
		}
	case *gocode.BasicType:
		{
			if grpcType, hasGrpcType := basicToGrpc[arg.Name]; hasGrpcType {
				return grpcType, nil
			}
			return "", fmt.Errorf("%v is not supported by GRPC", arg.Name)
		}
	case *gocode.Map:
		{
			keyType, isValidKey := getMapKeyType(arg.KeyType)
			if !isValidKey {
				return "", fmt.Errorf("GRPC cannot use %v as a map key", arg.KeyType)
			}
			valueType, err := b.getGRPCType(arg.ValueType)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("map<%v,%v>", keyType, valueType), nil
		}
	case *gocode.Slice:
		{
			// []byte is a special case where the type is 'bytes', everything else is a repeated
			if basic, isBasic := arg.SliceOf.(*gocode.BasicType); isBasic && basic.Name == "byte" {
				return "bytes", nil
			}
			// map is a special case that can't be repeated
			if _, isMap := arg.SliceOf.(*gocode.Map); isMap {
				return "", fmt.Errorf("GRPC does not support arrays of maps %v", t.String())
			}
			name, err := b.getGRPCType(arg.SliceOf)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("repeated %v", name), nil
		}
	default:
		{
			// all others are invalid or not yet supported
			return "", fmt.Errorf("GRPC cannot serialize %v", t.String())
		}
	}
}

var protoFileTemplate = `syntax="proto3";
option go_package="{{ .GoPackage }}";
package {{ .Package }};

{{ range $k, $msg := .Messages }}
message {{$msg.Name}} {
    {{- range $k, $field := $msg.FieldList}}
    {{$field.Type}} {{$field.Name}} = {{$field.Position}}
    {{- end}}
}
{{ end -}}

{{ range $k, $service := .Services }}
service {{$service.Name}} {
    {{- range $k, $method := $service.Methods}}
    rpc {{$method.Name}} ({{$method.Request.Name}}) returns ({{$method.Response.Name}}) {}
    {{- end}}
}
{{ end }}
`

func (b *GRPCProtoBuilder) WriteProtoFile(outputFilePath string) error {
	t, err := template.New("protofile").Parse(protoFileTemplate)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(outputFilePath, os.O_CREATE, 0755)
	if err != nil {
		return err
	}

	return t.Execute(f, b)

}
