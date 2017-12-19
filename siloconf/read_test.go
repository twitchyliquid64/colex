package siloconf

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	sf, err := LoadSiloFile("testdata/good.hcl")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*Silo{
		&Silo{
			Name:  "hi",
			Class: "go-bin",
			Tags:  []string{"FE"},
			Base:  "img://busybox",
			Network: Network{
				InternetAccess: true,
				Hosts: map[string]string{
					"b4master": "192.168.54.1",
				},
				Nameservers: []string{"8.8.8.8"},
			},
			Binary: Binary{
				Path: "/bin/ls",
			},
		},
		&Silo{
			Name: "welp",
			Base: "img://busybox",
			Files: map[string]File{
				"binary": File{Path: "/bin/ls", SiloPath: "/lister"},
			},
			Binary: Binary{
				Path: "/lister",
			},
		},
	}

	if len(sf.Silos) != len(expected) {
		t.Fatalf("Got len(silos) = %d, want %d", len(sf.Silos), len(expected))
	}

	for i := range expected {
		if !reflect.DeepEqual(sf.Silos[i], expected[i]) {
			t.Errorf("silo[%d] = %+v, want %+v", i, sf.Silos[i], expected[i])
		}
	}
}
