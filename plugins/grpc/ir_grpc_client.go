package grpc

import (
	"fmt"

	"gitlab.mpi-sws.org/cld/blueprint/blueprint/pkg/blueprint"
	"gitlab.mpi-sws.org/cld/blueprint/blueprint/pkg/core/irutil"
	"gitlab.mpi-sws.org/cld/blueprint/blueprint/pkg/core/service"
	"gitlab.mpi-sws.org/cld/blueprint/plugins/golang"
	"gitlab.mpi-sws.org/cld/blueprint/plugins/golang/gocode"
	"gitlab.mpi-sws.org/cld/blueprint/plugins/grpc/grpccodegen"
	"golang.org/x/exp/slog"
)

/*
IRNode representing a client to a Golang server.
This node does not introduce any new runtime interfaces or types that can be used by other IRNodes
GRPC code generation happens during the ModuleBuilder GenerateFuncs pass
*/
type GolangClient struct {
	golang.Node
	golang.Service
	golang.GeneratesFuncs
	golang.Instantiable

	InstanceName string
	ServerAddr   *GolangServerAddress

	outputPackage string
}

func newGolangClient(name string, addr *GolangServerAddress) (*GolangClient, error) {
	node := &GolangClient{}
	node.InstanceName = name
	node.ServerAddr = addr
	node.outputPackage = "grpc"

	return node, nil
}

func (n *GolangClient) String() string {
	return n.InstanceName + " = GRPCClient(" + n.ServerAddr.Name() + ")"
}

func (n *GolangClient) Name() string {
	return n.InstanceName
}

func (node *GolangClient) GetInterface(visitor irutil.BuildContext) service.ServiceInterface {
	return node.GetGoInterface(visitor)
}

func (node *GolangClient) GetGoInterface(visitor irutil.BuildContext) *gocode.ServiceInterface {
	grpc, isGrpc := node.ServerAddr.GetInterface(visitor).(*GRPCInterface)
	if !isGrpc {
		return nil
	}
	wrapped, isValid := grpc.Wrapped.(*gocode.ServiceInterface)
	if !isValid {
		return nil
	}
	return wrapped
}

// Generates proto files and the RPC client
func (node *GolangClient) GenerateFuncs(builder golang.ModuleBuilder) error {
	// Get the service that we are wrapping
	service := node.GetGoInterface(builder)
	if service == nil {
		return blueprint.Errorf("expected %v to have a gocode.ServiceInterface but got %v",
			node.Name(), node.ServerAddr.GetInterface(builder))
	}

	// Only generate grpc client instantiation code for this service once
	if builder.Visited(service.Name + ".grpc.client") {
		return nil
	}

	// Generate the .proto files
	err := grpccodegen.GenerateGRPCProto(builder, service, node.outputPackage)
	if err != nil {
		return err
	}

	// Generate the RPC client
	err = grpccodegen.GenerateClient(builder, service, node.outputPackage)
	if err != nil {
		return err
	}

	return nil
}

func (node *GolangClient) AddInstantiation(builder golang.GraphBuilder) error {
	// Only generate instantiation code for this instance once
	if builder.Visited(node.InstanceName) {
		return nil
	}

	constructor := &gocode.Constructor{
		Package: builder.Module().Info().Name + "/" + node.outputPackage,
		Func: gocode.Func{
			Name: fmt.Sprintf("New_%v_GRPCClient", node.GetGoInterface(builder).Name),
			Arguments: []gocode.Variable{
				{Name: "addr", Type: &gocode.BasicType{Name: "string"}},
			},
		},
	}

	slog.Info(fmt.Sprintf("Instantiating GRPCClient %v in %v/%v", node.InstanceName, builder.Info().Package.PackageName, builder.Info().FileName))
	return builder.DeclareConstructor(node.InstanceName, constructor, []blueprint.IRNode{node.ServerAddr})
}

func (node *GolangClient) ImplementsGolangNode()    {}
func (node *GolangClient) ImplementsGolangService() {}
