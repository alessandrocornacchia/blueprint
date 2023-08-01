package golang

import (
	"fmt"

	"gitlab.mpi-sws.org/cld/blueprint/pkg/blueprint"
	"golang.org/x/exp/slog"
)

// This is used to accumulate modules, files, and definitions that is generated by golang instances
type GolangArtifactGenerator struct {
	blueprint.ArtifactGenerator

	Modules map[string]string // Modules that will be included in the generated go.mod as 'require {package} {version}
	Code    map[string]string // Code that has been auto-generated and will be included as generated output
	Files   map[string]string // Code that will be copied into the generated output
}

// This is used to accumulate code for instantiating golang instances
type GolangCodeGenerator struct {
	blueprint.ArtifactGenerator

	Imports     map[string]interface{} // Imports in the generated instantiation code
	Definitions map[string]string      // DI definitions of nodes
	Instantions map[string]interface{} // The DI nodes to instantiate (which will trigger any dependencies too)
}

func NewGolangArtifactGenerator() *GolangArtifactGenerator {
	gc := GolangArtifactGenerator{}
	gc.Modules = make(map[string]string)
	gc.Code = make(map[string]string)
	gc.Files = make(map[string]string)
	return &gc
}

func NewGolangCodeAccumulator() *GolangCodeGenerator {
	gc := GolangCodeGenerator{}
	gc.Imports = make(map[string]interface{})
	gc.Definitions = make(map[string]string)
	gc.Instantions = make(map[string]interface{})
	return &gc
}

// Adds a dependency to the specified package and version to the generated code
// Can return an error if there are conflicting version dependencies
func (cg *GolangArtifactGenerator) AddModule(pkg, version string) error {
	existing_version, ok := cg.Modules[pkg]
	if ok {
		if existing_version != version {
			return fmt.Errorf("incompatible module versions required %s and %s for package %s", existing_version, version, pkg)
		}
	} else {
		cg.Modules[pkg] = version
	}
	return nil
}

// Adds generated code at the specified path in the output
func (cg *GolangArtifactGenerator) AddCode(path, code string) error {
	_, ok := cg.Code[path]
	if ok {
		slog.Warn("Overwriting existing code", "path", path)
	}
	cg.Code[path] = code
	return nil
}

// Copies a file to the path specified
func (cg *GolangArtifactGenerator) AddFile(path, inputpath string) error {
	_, ok := cg.Files[path]
	if ok {
		slog.Warn("Overwriting existing file", "outputpath", path, "inputpath", inputpath)
	}
	cg.Files[path] = inputpath
	return nil
}

func (cg *GolangArtifactGenerator) AddFiles(paths []string) error {
	// TODO: implement
	return nil
}

// This is called by the compiler after nodes have added their code, to write output
func (cg *GolangArtifactGenerator) GenerateOutput(path string) error {
	// TODO implement
	return nil
}

// Adds an import statement
func (cg *GolangCodeGenerator) Import(pkg string) {
	cg.Imports[pkg] = nil
}

// Adds a DI definition
func (cg *GolangCodeGenerator) Def(name, code string) {
	_, ok := cg.Definitions[name]
	if ok {
		slog.Warn("Overwriting existing DI definition", "name", name)
	}
	cg.Definitions[name] = code
}

func (cg *GolangCodeGenerator) Instantiate(name string) {
	cg.Instantions[name] = nil
}
