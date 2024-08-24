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

var (
	mgmtInterfaceName = "mgmt"
)

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

func (vm *VirtualMachine) CreateOrUpdate(msg Message) {
	//
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

func (vm *VirtualMachine) GetInterfaceByID(id int64) (*models.VMInterface, error) {
	vmId := strconv.FormatInt(vm.NetboxId, 10)
	interfaceId := strconv.FormatInt(id, 10)

	ipIfParam := &virtualization.VirtualizationInterfacesListParams{
		VirtualMachineID: &vmId,
		ID:               &interfaceId,
	}
	i, err := vm.n.Client.Virtualization.
		VirtualizationInterfacesList(ipIfParam.WithTimeout(vm.n.GetDefaultTimeout()), nil)
	if err != nil {
		return nil, fmt.Errorf("error listing virtual machine interfaces: %w", err)
	}

	if *i.Payload.Count != 1 {
		return nil, fmt.Errorf("error listing virtual machine interfaces: expected 1 item, got %d", i.Payload.Count)
	}

	return i.Payload.Results[0], nil
}

func (vm *VirtualMachine) GetManagementInterface() (*models.VMInterface, error) {
	vmId := strconv.FormatInt(vm.NetboxId, 10)

	ipIfParam := &virtualization.VirtualizationInterfacesListParams{
		VirtualMachineID: &vmId,
		Name:             &mgmtInterfaceName,
	}
	in, err := vm.n.Client.Virtualization.
		VirtualizationInterfacesList(ipIfParam.WithTimeout(vm.n.GetDefaultTimeout()), nil)
	if err != nil {
		return nil, fmt.Errorf("error listing virtual machine interfaces: %w", err)
	}

	//If there are no management interface, create it
	if *in.Payload.Count == 0 {
		mgmtInterface, err := vm.CreateInterface("mgmt")
		if err != nil {
			return nil, fmt.Errorf("error creating virtual machine interface: %w", err)
		}

		return mgmtInterface.Payload, nil
	}

	return in.Payload.Results[0], nil
}

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

func (vm *VirtualMachine) UpdateManagementIP(msg Message) error {
	//Get vm management interface
	itf, err := vm.GetManagementInterface()
	if err != nil {
		return fmt.Errorf("error getting interfaces: %w", err)
	}

	objectType := "virtualization.vminterface"

	//Update management interface with latest IP
	err = vm.UpdateInterfaceIP(msg.IpAddress, itf.ID, objectType)
	if err != nil {
		return err
	}

	var mgmtInterfaceId = strconv.FormatInt(itf.ID, 10)
	params := ipam.NewIpamIPAddressesListParams()
	params.SetVminterfaceID(&mgmtInterfaceId)

	result, err := vm.n.Client.Ipam.IpamIPAddressesList(params, nil)
	if err != nil {
		return fmt.Errorf("error listing ip addresses: %w", err)
	}

	var ipCount = result.Payload.Count
	util.Info("There are actually %s IP(s) associated with the management interface", strconv.FormatInt(*ipCount, 10))

	if *ipCount > 1 {
		return fmt.Errorf("there are more than one management ip linked to the management interface")
	}

	if *ipCount == 1 {
		ip := result.Payload.Results[0]
		if *ip.Address == msg.IpAddress {
			//Nothing to do
			return nil
		}

		// 4. The management IP changed, so :
		// - unlink the old ip and interface
		// - set the new ip to the interface

		oldIpUpdatePrams := models.WritableIPAddress{
			Address:            &msg.IpAddress,
			AssignedObjectType: nil,
			AssignedObjectID:   nil,
		}

		paramUnlinkOldIp := ipam.
			NewIpamIPAddressesPartialUpdateParams().
			WithID(ip.ID).
			WithData(&oldIpUpdatePrams).
			WithTimeout(vm.n.GetDefaultTimeout())
		_, err = vm.n.Client.Ipam.IpamIPAddressesPartialUpdate(paramUnlinkOldIp, nil)
		if err != nil {
			return fmt.Errorf("error updating management ip addresses of VM #%d: %w", vm.NetboxId, err)
		}

		util.Success("Successfully updated management ip addresses of VM #%d with new IP: %s", vm.NetboxId, msg.IpAddress)
		return nil
	}

	// 5. No existing IP, but verify that she doesn't already exist in the netbox
	ipSearchParams := ipam.NewIpamIPAddressesListParams()
	ipSearchParams.Q = &msg.IpAddress
	result, err = vm.n.Client.Ipam.IpamIPAddressesList(ipSearchParams, nil)
	if err != nil {
		return fmt.Errorf("error listing existing ip addresses: %w", err)
	}

	existingIpCount := result.Payload.Count
	newIpAddrId := int64(0)
	if *existingIpCount == 0 {
		util.Info("There is no IP registered in the netbox. Create him.")
		var ipType = "virtualization.vminterface"
		newIp := &ipam.IpamIPAddressesCreateParams{
			Data: &models.WritableIPAddress{
				Address:            &msg.IpAddress,
				AssignedObjectID:   &itf.ID,
				AssignedObjectType: &ipType,
			},
		}
		r, err := vm.n.Client.Ipam.IpamIPAddressesCreate(newIp.WithTimeout(vm.n.GetDefaultTimeout()), nil)
		if err != nil {
			return fmt.Errorf("error creating ip address: %w", err)
		}

		newIpAddrId = r.Payload.ID
	} else {
		newIpAddrId = result.Payload.Results[0].ID
	}

	var ipType = "virtualization.vminterface"

	ip := vm.n.getIpAddress(msg.IpAddress)
	ip.ID = newIpAddrId
	ip.AssignedObjectID = &itf.ID
	ip.AssignedObjectType = &ipType

	ifUpdateParam := &ipam.IpamIPAddressesPartialUpdateParams{
		Data: ip,
	}
	_, err = vm.n.Client.Ipam.IpamIPAddressesPartialUpdate(ifUpdateParam.WithTimeout(vm.n.GetDefaultTimeout()), nil)
	if err != nil {
		return fmt.Errorf("error updating ip address: %w", err)
	}

	return nil
}

func (vm *VirtualMachine) Exists(hostname string, serial string) (bool, int64, error) {
	if vm.NetboxId <= 0 && vm.n == nil {
		return false, 0, nil
	}

	return vm.n.VmExists(hostname, serial)
}
