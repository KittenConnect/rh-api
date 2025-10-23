package model

import "testing"

func TestVirtualMachine_NewVM(t *testing.T) {
	netbox := &Netbox{}
	msg := Message{
		Hostname: "vm-12345",
	}

	vm := NewVM(netbox, msg)
	if vm.Serial != "12345" {
		t.Fatalf("expected serial to be %q, got %q", "12345", vm.Serial)
	}

}

func TestVirtualMachine_Exists(t *testing.T) {
	// When VM has no Netbox client and no NetboxId, Exists should early-return (false, 0, nil)
	vm := &VirtualMachine{}

	exists, id, err := vm.Exists("some-host", "some-serial")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatalf("expected exists to be false, got true")
	}
	if id != 0 {
		t.Fatalf("expected id to be 0, got %d", id)
	}
}
