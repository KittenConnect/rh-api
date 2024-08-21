package model

import (
	"context"
	"errors"
	"fmt"
	"github.com/KittenConnect/rh-api/util"
	"github.com/netbox-community/go-netbox/netbox"
	"github.com/netbox-community/go-netbox/netbox/client"
	"github.com/netbox-community/go-netbox/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/netbox/client/virtualization"
	"github.com/netbox-community/go-netbox/netbox/models"
	"os"
	"strconv"
	"time"
)

// Netbox structure
// For internal use ONLY !
// To get an instance, call NewNetbox method
type Netbox struct {
	ctx context.Context

	Client *client.NetBoxAPI

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

func (n *Netbox) IsConnected() bool {
	return n._isConnected
}

func (n *Netbox) Connect() error {
	if n._isConnected {
		return nil
	}

	n.Client = netbox.NewNetboxWithAPIKey(os.Getenv("NETBOX_API_URL"), os.Getenv("NETBOX_API_TOKEN"))
	n._isConnected = true

	return nil
}

func (n *Netbox) GetDefaultTimeout() time.Duration {
	return time.Duration(30) * time.Second
}

func (n *Netbox) getIpAddress(ip string) *models.WritableIPAddress {
	return &models.WritableIPAddress{
		Address: &ip,
		Status:  models.IPAddressStatusValueActive,
	}
}

func (n *Netbox) CreateVM(msg Message) error {
	if !n._isConnected {
		return errors.New("netbox is not connected")
	}

	vm := NewVM()
	vmCreateData := vm.Create(msg)

	params := virtualization.NewVirtualizationVirtualMachinesCreateParams().WithData(&vmCreateData)
	result, err := n.Client.Virtualization.VirtualizationVirtualMachinesCreate(params, nil)
	if err != nil {
		if result != nil && result.Payload != nil {
			return fmt.Errorf("error creating virtual machine: %w \n\t%s", err, result.Error())
		}

		return fmt.Errorf("error creating virtual machine: %w", err)
	}

	util.Success("Created machine ID: %d", result.Payload.ID)
	vm.Id = result.Payload.ID

	//Create management interface
	res, err := vm.CreateInterface(n, "mgmt")
	if err != nil {
		return err
	}

	var (
		ifId       = res.Payload.ID
		objectType = "virtualization.vminterface"
	)

	//Verify if ip already exists
	ipAlreadyExist := &ipam.IpamIPAddressesListParams{
		Address: &msg.IpAddress,
	}
	req, err := n.Client.Ipam.
		IpamIPAddressesList(ipAlreadyExist.WithTimeout(n.GetDefaultTimeout()), nil)
	if err != nil {
		return fmt.Errorf("error checking ip addresses existance : %w", err)
	}

	util.Info("Found #%d IPs in %v", *req.Payload.Count, *req)
	//We don't have that ip registered on netbox, so let's create him
	if *req.Payload.Count == 0 {
		//Set ip to the interface
		createdIP, err := vm.CreateIP(n, msg.IpAddress, models.IPAddressStatusValueActive, ifId, objectType)
		if err != nil {
			return err
		}

		util.Success("\tSuccessfully created vm management ip: %s", strconv.FormatInt(createdIP.Payload.ID, 10))
	} else if *req.Payload.Count == 1 {
		ip := req.Payload.Results[0]

		linkedInterfaceId := ip.AssignedObjectID

		//Si l'ip n'est pas liée à une interface
		//On l'assigne à l'interface de la machine et zou
		if linkedInterfaceId == nil {
			return vm.changeIPInterface(n, msg.IpAddress, ifId, objectType)
		}

		//Sinon on vérifie si la VM possède d'autres IP sur l'interface de management
		interfaceId := *linkedInterfaceId
		vmInterfaceParam := virtualization.
			NewVirtualizationInterfacesReadParams().
			WithID(interfaceId).
			WithTimeout(n.GetDefaultTimeout())

		vmInterfaceResult, err := n.Client.Virtualization.VirtualizationInterfacesRead(vmInterfaceParam, nil)
		if err != nil {
			return fmt.Errorf("error reading virtual machine interface: %w", err)
		}

		vmID := strconv.FormatInt(vmInterfaceResult.Payload.VirtualMachine.ID, 10)

		mgmtInterfaceName := "mgmt"
		nestedVmParams := &virtualization.VirtualizationInterfacesListParams{
			Name:             &mgmtInterfaceName,
			VirtualMachineID: &vmID,
		}
		nestedVmInterfaces, err := n.Client.Virtualization.
			VirtualizationInterfacesList(nestedVmParams.WithTimeout(n.GetDefaultTimeout()), nil)
		if err != nil {
			return fmt.Errorf("error listing virtual machine interfaces: %w", err)
		}

		mgmtInterface := nestedVmInterfaces.Payload.Results[0]
		if mgmtInterface.CountIpaddresses == 1 {
			//L'interface possède d'autres IPs
			//Du coup, on prend l'ip en question
			util.Info("Remove the link ...")
			err := vm.changeIPInterface(n, msg.IpAddress, ifId, objectType)
			if err != nil {
				return err
			}
			util.Success("IP changed of interface")

			return nil
		} else {
			//Sinon on laisse l'ip sur la VM
			util.Info("L'IP %s reste sur l'interface n°%d", msg.IpAddress, mgmtInterface.ID)
		}

		util.Warn("Trying to using existing IP on VM interface #%s", strconv.FormatInt(mgmtInterface.ID, 10))
	}

	return nil
}

func (n *Netbox) UpdateVM(id int64, msg Message) error {
	vm := NewVM()
	vm.Id = id

	vmConf := vm.Create(msg)

	updateParams := &virtualization.VirtualizationVirtualMachinesPartialUpdateParams{
		Data: &vmConf,
		ID:   id,
	}

	//Update management IP
	// 1. Get current interface IPs list
	var (
		vmId       = strconv.FormatInt(id, 10)
		mgmtIfName = "mgmt"
	)

	interfaces, err := vm.GetInterfaces(n, mgmtIfName)
	if err != nil {
		return err
	}

	// 2. If there is no interface, quit
	var (
		ifCount = interfaces.Payload.Count
	)
	if *ifCount < 1 {
		//No virtual interface, create one
		var (
			mgmtInterfaceName = "mgmt"
		)

		ifParam := models.WritableVMInterface{
			Name:    &mgmtInterfaceName,
			Enabled: true,

			TaggedVlans: []int64{},

			VirtualMachine: &id,
		}
		paramInterface := virtualization.
			NewVirtualizationInterfacesCreateParams().
			WithData(&ifParam).WithTimeout(n.GetDefaultTimeout())
		_, err := n.Client.Virtualization.VirtualizationInterfacesCreate(paramInterface, nil)
		if err != nil {
			return fmt.Errorf("error creating virtual machine interface: %w", err)
		}

		_, err = n.Client.Virtualization.
			VirtualizationVirtualMachinesPartialUpdate(updateParams.WithTimeout(n.GetDefaultTimeout()), nil)
		if err != nil {
			return fmt.Errorf("error updating virtual machine interface: %w", err)
		}

		util.Info("Updated VM #" + strconv.FormatInt(id, 10) + " management interface with IP " + msg.IpAddress)
		return nil
	}

	// 3. Get the current management IP
	mgmtInterface := interfaces.Payload.Results[0]
	var mgmtInterfaceId = strconv.FormatInt(mgmtInterface.ID, 10)
	params := ipam.NewIpamIPAddressesListParams()
	params.SetVminterfaceID(&mgmtInterfaceId)
	util.Info("Found MGMT Iface #%d -> %s", mgmtInterface.ID, mgmtInterfaceId)

	result, err := n.Client.Ipam.IpamIPAddressesList(params, nil)
	if err != nil {
		return fmt.Errorf("error listing ip addresses: %w", err)
	}

	var ipCount = result.Payload.Count
	util.Info("There are actually %s IP(s) associated with the management interface", strconv.FormatInt(*ipCount, 10))

	if *ipCount > 1 {
		return errors.New("there are more than one management ip linked to the management interface")
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
			WithTimeout(n.GetDefaultTimeout())
		_, err = n.Client.Ipam.IpamIPAddressesPartialUpdate(paramUnlinkOldIp, nil)
		if err != nil {
			return fmt.Errorf("error updating management ip addresses of VM #%d: %w", vmId, err)
		}

		util.Success("Successfully updated management ip addresses of VM #%d with new IP: %s", vmId, msg.IpAddress)
		return nil
	}

	// 5. No existing IP, but verify that she doesn't already exist in the netbox
	ipSearchParams := ipam.NewIpamIPAddressesListParams()
	ipSearchParams.Q = &msg.IpAddress
	result, err = n.Client.Ipam.IpamIPAddressesList(ipSearchParams, nil)
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
				AssignedObjectID:   &mgmtInterface.ID,
				AssignedObjectType: &ipType,
			},
		}
		r, err := n.Client.Ipam.IpamIPAddressesCreate(newIp.WithTimeout(n.GetDefaultTimeout()), nil)
		if err != nil {
			return fmt.Errorf("error creating ip address: %w", err)
		}

		newIpAddrId = r.Payload.ID
	} else {
		newIpAddrId = result.Payload.Results[0].ID
	}

	var ipType = "virtualization.vminterface"

	ip := n.getIpAddress(msg.IpAddress)
	ip.ID = newIpAddrId
	ip.AssignedObjectID = &mgmtInterface.ID
	ip.AssignedObjectType = &ipType

	ifUpdateParam := &ipam.IpamIPAddressesPartialUpdateParams{
		Data: ip,
	}
	_, err = n.Client.Ipam.IpamIPAddressesPartialUpdate(ifUpdateParam.WithTimeout(n.GetDefaultTimeout()), nil)
	if err != nil {
		return fmt.Errorf("error updating ip address: %w", err)
	}

	return nil
}

func (n *Netbox) CreateOrUpdateVM(msg Message) error {
	if !n._isConnected {
		return errors.New("netbox is not connected")
	}

	var vmId int64
	var hasFoundVm = false
	var err error

	// Call netbox API with specific serial, then update his settings accordingly
	//exist := contains(MachinesSerials, msg.Hostname) //TODO
	//if !exist {
	//If the vm don't exist in memory, fetch his details, if she exists in netbox
	req := virtualization.
		NewVirtualizationVirtualMachinesListParams().
		WithTimeout(n.GetDefaultTimeout())
	res, err := n.Client.Virtualization.VirtualizationVirtualMachinesList(req, nil)
	if err != nil {
		return fmt.Errorf("unable to get list of machines from netbox: %w", err)
	}

	for _, vm := range res.Payload.Results {
		if vm.Name == &msg.Hostname {
			vmId = vm.ID
			hasFoundVm = true
			break
		}

		var cf = vm.CustomFields.(map[string]interface{})
		var serial = ""
		_ = serial

		for k, v := range cf {
			switch c := v.(type) {
			case string:
				if k == "kc_serial_" {
					serial = c
				}
				//	fmt.Printf("Item %q is a string, containing %q\n", k, c)
				//case float64:
				//	fmt.Printf("Looks like item %q is a number, specifically %f\n", k, c)
				//default:
				//	fmt.Printf("Not sure what type item %q is, but I think it might be %T\n", k, c)
			}
		}

		if serial == msg.GetSerial() {
			vmId = vm.ID
			hasFoundVm = true
			break
		}
	}

	//Create VM if she doesn't exists in netbox
	if !hasFoundVm {
		err = n.CreateVM(msg)

		if err != nil {
			return fmt.Errorf("unable to create VM: %w", err)
		}
	} else {
		err = n.UpdateVM(vmId, msg)
		if err != nil {
			return fmt.Errorf("unable to update VM: %w", err)
		}

		//util.Success("VM updated successfully")
	}

	return nil
}
