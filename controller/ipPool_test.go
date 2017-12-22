package controller

import (
	"net"
	"testing"
)

func TestIpPool(t *testing.T) {
	pool, err := NewIPPool("10.200.0.1/16")
	if err != nil {
		t.Fatal(err)
	}

	n1, err := pool.Assignment()
	if err != nil {
		t.Errorf("pool.Assignment().err = %v, want nil", err)
	}
	if !n1.Equal(net.IP{10, 200, 0, 1}) {
		t.Errorf("pool.Assignment() #1 = %v, want 10.200.0.1", n1)
	}

	n2, err := pool.Assignment()
	if err != nil {
		t.Errorf("pool.Assignment().err = %v, want nil", err)
	}
	if !n2.Equal(net.IP{10, 200, 0, 2}) {
		t.Errorf("pool.Assignment() #2 = %v, want 10.200.0.2", n2)
	}

	n3, err := pool.Assignment()
	if err != nil {
		t.Errorf("pool.Assignment().err = %v, want nil", err)
	}
	if !n3.Equal(net.IP{10, 200, 0, 3}) {
		t.Errorf("pool.Assignment() #3 = %v, want 10.200.0.1", n3)
	}

	t.Logf("Freeing: %v", n1)
	pool.FreeAssignment([]net.IP{n1})

	n4, err := pool.Assignment()
	if err != nil {
		t.Errorf("pool.Assignment().err = %v, want nil", err)
	}
	if !n4.Equal(net.IP{10, 200, 0, 1}) {
		t.Errorf("pool.Assignment() #4 = %v, want 10.200.0.1", n4)
	}

	n5, err := pool.Assignment()
	if err != nil {
		t.Errorf("pool.Assignment().err = %v, want nil", err)
	}
	if !n5.Equal(net.IP{10, 200, 0, 4}) {
		t.Errorf("pool.Assignment() #5 = %v, want 10.200.0.4", n5)
	}
}

func TestIpPoolExhausted(t *testing.T) {
	pool, err := NewIPPool("192.168.0.1/2")
	if err != nil {
		t.Fatal(err)
	}

	n1, err := pool.Assignment()
	if err != nil {
		t.Fatalf("pool.Assignment().err = %v, want nil", err)
	}
	if !n1.Equal(net.IP{192, 168, 0, 1}) {
		t.Errorf("pool.Assignment() #1 = %v, want 192.168.0.1", n1)
	}

	n2, err := pool.Assignment()
	if err != nil {
		t.Fatalf("pool.Assignment().err = %v, want nil", err)
	}
	if !n2.Equal(net.IP{192, 168, 0, 2}) {
		t.Errorf("pool.Assignment() #2 = %v, want 192.168.0.2", n2)
	}
	if _, err := pool.Assignment(); err != nil {
		t.Error("pool.Assignment() #3 error = nil, want 'pool exhausted'")
	}
}
