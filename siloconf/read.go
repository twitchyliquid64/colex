package siloconf

import (
	"errors"
	"io/ioutil"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
)

// ParseSiloFile takes a serialized silo file and decodes it into a struct.
func ParseSiloFile(data []byte) (*SiloFile, error) {
	astRoot, err := hcl.ParseBytes(data)
	if err != nil {
		return nil, err
	}

	if _, ok := astRoot.Node.(*ast.ObjectList); !ok {
		return nil, errors.New("schema malformed")
	}

	var outSpec SiloFile
	err = hcl.DecodeObject(&outSpec, astRoot)
	if err != nil {
		return nil, err
	}
	return &outSpec, nil
}

// LoadSiloFile loads a silo config from the specified fs path.
func LoadSiloFile(fpath string) (*SiloFile, error) {
	d, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	return ParseSiloFile(d)
}
