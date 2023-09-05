package main

import (
	"fmt"
	"os"

	"gitlab.mpi-sws.org/cld/blueprint/blueprint/pkg/blueprint"
	"gitlab.mpi-sws.org/cld/blueprint/plugins/goproc"
	"gitlab.mpi-sws.org/cld/blueprint/plugins/workflow"
	"golang.org/x/exp/slog"
)

func serviceDefaults(wiring blueprint.WiringSpec, serviceName string) string {
	procName := fmt.Sprintf("p%s", serviceName)
	// opentelemetry.Instrument(wiring, serviceName)
	// grpc.Deploy(wiring, serviceName)
	return goproc.CreateProcess(wiring, procName, serviceName)
}

func main() {

	fmt.Println("Constructing Wiring Spec")

	wiring := blueprint.NewWiringSpec("leaf_example")

	// Create the wiring spec
	workflow.Init("../workflow")

	// b_cache := memcached.PrebuiltProcess(wiring, "b_cache")
	// b := workflow.Define(wiring, "b", "LeafServiceImpl", b_cache)

	b := workflow.Define(wiring, "b", "LeafServiceImpl")
	// a := workflow.Define(wiring, "a", "NonLeafServiceImpl", b) // Will fail, because no constructors returning the impl directly
	a := workflow.Define(wiring, "a", "NonLeafService", b)

	// pa := serviceDefaults(wiring, a)
	// pb := serviceDefaults(wiring, b)
	proc := goproc.CreateProcess(wiring, "proc", a, b)

	// Let's print out all of the nodes currently defined in the wiring spec
	slog.Info("Wiring Spec: \n" + wiring.String())

	bp := wiring.GetBlueprint()
	// bp.Instantiate(pa, pb)
	bp.Instantiate(proc)

	application, err := bp.Build()
	if err != nil {
		slog.Error("Unable to build blueprint, exiting", "error", err)
		slog.Info("Application: \n" + application.String())
		os.Exit(1)
	}

	slog.Info("Application: \n" + application.String())

	// Below here is a WIP on generating code
	// proc := application.Children["pa"].(*goproc.Process)
	p := application.Children["proc"].(*goproc.Process)
	err = p.GenerateArtifacts("tmp")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
