package model

import (
	"context"
	"errors"
	"fmt"
	"github.com/KittenConnect/rh-api/util"
	"github.com/netbox-community/go-netbox/v4"
	"os"
)

var MachinesSerials map[string]string

var (
	NetboxVmSerialPrefix string = os.Getenv("NETBOX_VM_SERIAL_FIELD")
)

// Netbox structure
// For internal use ONLY !
// To get an instance, call NewNetbox method
type Netbox struct {
	ctx context.Context

	Client *netbox.APIClient

	_isConnected bool
}

// NewNetbox return a fresh Netbox object
func NewNetbox() Netbox {
	nbx := Netbox{
		ctx:    context.Background(),
		Client: nil,

		_isConnected: false,
	}

	return nbx
}

func (n Netbox) IsConnected() bool {
	return n._isConnected
}

func (n Netbox) Connect() error {
	if n._isConnected {
		return nil
	}

	n.Client = netbox.NewAPIClientFor(os.Getenv("NETBOX_API_URL"), os.Getenv("NETBOX_API_KEY"))
	n._isConnected = true

	return nil
}

func (n Netbox) CreateVM(msg Message) (int32, error) {
	var (
		id int32
	)

	if !n._isConnected {
		return id, errors.New("netbox is not connected")
	}

	vm := netbox.WritableVirtualMachineWithConfigContextRequest{
		Name: msg.Hostname,
		CustomFields: map[string]interface{}{
			"machine_serial": msg.Serial,
		},
	}

	_, result, err := n.Client.VirtualizationAPI.VirtualizationVirtualMachinesCreate(n.ctx).WritableVirtualMachineWithConfigContextRequest(vm).Execute()
	if err != nil {
		return id, err
	}

	util.Info(fmt.Sprintf("Created machine ID : %s", result.Body))

	return id, nil
}

func (n Netbox) UpdateVM(serial string, conf string) error {
	if !n._isConnected {
		return errors.New("netbox is not connected")
	}

	// Call netbox API with specific serial, then update his settings accordingly
	_, exist := MachinesSerials[serial]
	if exist {
		return nil
	}

	res, _, err := n.Client.VirtualizationAPI.
		VirtualizationVirtualMachinesList(n.ctx).
		//Name([]string{serial}).
		//Limit(1).
		Execute()
	if err != nil {
		return err
	}

	var vmId int32
	var hasFoundVm bool = false

	for _, vm := range res.Results {
		if vm.CustomFields[NetboxVmSerialPrefix] == serial {
			vmId = vm.Id
			hasFoundVm = true
			break
		}
	}

	if !hasFoundVm {
		//Create VM
		print(vmId) //TODO
	}

	return nil
}
