package model

import (
	"fmt"
	"github.com/KittenConnect/rh-api/util"
	"github.com/netbox-community/go-netbox/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/netbox/client/virtualization"
	"github.com/netbox-community/go-netbox/netbox/models"
	"net"
	"strconv"
)

type VirtualMachine struct {
	NetboxId int64   `json:"id"`
	Cluster  Cluster `json:"cluster"`
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Serial   string  `json:"serial"`

	ManagementIP net.IP `json:"management_ip"`
	n            *Netbox
}

func NewVM(n *Netbox, msg Message) *VirtualMachine {
	vm := &VirtualMachine{
		n:        n,
		NetboxId: -1,

		Name: msg.Hostname,
	}

	return vm
}

func (vm *VirtualMachine) Get() models.WritableVirtualMachineWithConfigContext {
	// todo: implement netbox func
	return models.WritableVirtualMachineWithConfigContext{
		Cluster: &vm.Cluster.ID,
		Name:    &vm.Name,
		Status:  vm.Status,

		CustomFields: map[string]interface{}{
			"kc_serial_": vm.Serial,
		},
	}
}

func (vm *VirtualMachine) Create(msg Message) (*virtualization.VirtualizationVirtualMachinesCreateCreated, error) {
	conf := models.WritableVirtualMachineWithConfigContext{
		Cluster: &vm.Cluster.ID,
		Name:    &vm.Name,
		Status:  vm.Status,

		CustomFields: map[string]interface{}{
			"kc_serial_": msg.GetSerial(),
		},
	}

	params := virtualization.NewVirtualizationVirtualMachinesCreateParams().WithData(&conf)
	return vm.n.Client.Virtualization.VirtualizationVirtualMachinesCreate(params, nil)
}

// Update vm infos to netbox
func (vm *VirtualMachine) Update() error {
	data := vm.Get()

	updateParams := &virtualization.VirtualizationVirtualMachinesPartialUpdateParams{
		Data: &data,
		ID:   vm.NetboxId,
	}

	_, err := vm.n.Client.Virtualization.
		VirtualizationVirtualMachinesPartialUpdate(updateParams.WithTimeout(vm.n.GetDefaultTimeout()), nil)
	if err != nil {
		return fmt.Errorf("error updating virtual machine interface: %w", err)
	}

	return nil
}

func (vm *VirtualMachine) GetInterfaces(name string) (*virtualization.VirtualizationInterfacesListOK, error) {
	vmId := strconv.FormatInt(vm.NetboxId, 10)

	ipIfParam := &virtualization.VirtualizationInterfacesListParams{
		VirtualMachineID: &vmId,
		Name:             &name,
	}
	interfaces, err := vm.n.Client.Virtualization.
		VirtualizationInterfacesList(ipIfParam.WithTimeout(vm.n.GetDefaultTimeout()), nil)
	if err != nil {
		return nil, fmt.Errorf("error listing virtual machine interfaces: %w", err)
	}

	return interfaces, nil
}

func (vm *VirtualMachine) CreateInterface(n *Netbox, ifName string) (*virtualization.VirtualizationInterfacesCreateCreated, error) {
func (vm *VirtualMachine) CreateInterface(ifName string) (*virtualization.VirtualizationInterfacesCreateCreated, error) {
	ifParam := models.WritableVMInterface{
		Name:    &ifName,
		Enabled: true,

		TaggedVlans: []int64{},

		VirtualMachine: &vm.NetboxId,
	}
	paramInterface := virtualization.
		NewVirtualizationInterfacesCreateParams().
		WithData(&ifParam).
		WithTimeout(vm.n.GetDefaultTimeout())
	res, err := vm.n.Client.Virtualization.VirtualizationInterfacesCreate(paramInterface, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating virtual machine interface: %w", err)
	}
	util.Success("\tSuccessfully created vm interface %s", strconv.FormatInt(res.Payload.ID, 10))

	return res, nil
}

func (vm *VirtualMachine) UpdateInterfaceIP(address string, ifId int64, objectType string) error {
	ip := vm.n.getIpAddress(address)
	ip.AssignedObjectID = &ifId
	ip.AssignedObjectType = &objectType

	ifUpdateParam := &ipam.IpamIPAddressesPartialUpdateParams{
		Data: ip,
	}

	_, err := vm.n.Client.Ipam.
		IpamIPAddressesPartialUpdate(ifUpdateParam.WithID(ip.ID).
			WithTimeout(vm.n.GetDefaultTimeout()), nil)
	if err != nil {
		return fmt.Errorf("error updating ip address: %w", err)
	}

	util.Success("Update IP to VM interface")
	return nil
}

func (vm *VirtualMachine) CreateIP(n *Netbox, address string, status string, linkedObjectId int64, linkedObjectType string) (*ipam.IpamIPAddressesCreateCreated, error) {
	ip := &models.WritableIPAddress{
		Address: &address,
		Status:  status,
	}

	if linkedObjectId != -1 && linkedObjectType != "" {
		ip.AssignedObjectID = &linkedObjectId
		ip.AssignedObjectType = &linkedObjectType
	}

	ipCreateParams := &ipam.IpamIPAddressesCreateParams{
		Data: ip,
	}

	res, err := n.Client.Ipam.IpamIPAddressesCreate(ipCreateParams.WithTimeout(n.GetDefaultTimeout()), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating ip address: %w", err)
	}

	return res, nil
}
