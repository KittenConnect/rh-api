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

func getVm(m Message) models.WritableVirtualMachineWithConfigContext {
	var (
		status        = "active"
		cluster int64 = 1
	)

	return models.WritableVirtualMachineWithConfigContext{
		Cluster: &cluster,
		Name:    &m.Hostname,
		Status:  status,

		CustomFields: map[string]interface{}{
			"kc_serial_": m.GetSerial(),
		},
	}
}

func (n *Netbox) changeIPInterface(msg Message, ifId int64, objectType string) error {
	ip := models.WritableIPAddress{
		Address:            &msg.IpAddress,
		AssignedObjectID:   &ifId,
		AssignedObjectType: &objectType,
		Status:             models.IPAddressStatusValueActive,
	}
	ifUpdateParam := &ipam.IpamIPAddressesPartialUpdateParams{
		Data: &ip,
	}

	_, err := n.Client.Ipam.IpamIPAddressesPartialUpdate(ifUpdateParam.WithTimeout(time.Duration(30)*time.Second), nil)
	if err != nil {
		return fmt.Errorf("error updating ip address: %w", err)
	}

	util.Success("Update IP to VM interface")
	return nil
}

func (n *Netbox) CreateVM(msg Message) error {

	if !n._isConnected {
		return errors.New("netbox is not connected")
	}

	vm := getVm(msg)

	params := virtualization.NewVirtualizationVirtualMachinesCreateParams().WithData(&vm)
	result, err := n.Client.Virtualization.VirtualizationVirtualMachinesCreate(params, nil)
	if err != nil {
		if result != nil && result.Payload != nil {
			return fmt.Errorf("error creating virtual machine: %w \n\t%s", err, result.Error())
		}

		return fmt.Errorf("error creating virtual machine: %w", err)
	}

	util.Success(fmt.Sprintf("Created machine ID : %d", result.Payload.ID))

	//Create management interface
	var (
		mgmtInterfaceName = "mgmt"
	)

	ifParam := models.WritableVMInterface{
		Name:    &mgmtInterfaceName,
		Enabled: true,

		TaggedVlans: []int64{1},

		VirtualMachine: &result.Payload.ID,
	}
	paramInterface := virtualization.NewVirtualizationInterfacesCreateParams().WithData(&ifParam)
	res, err := n.Client.Virtualization.VirtualizationInterfacesCreate(paramInterface, nil)
	if err != nil {
		return fmt.Errorf("error creating virtual machine interface: %w", err)
	}
	util.Success("\tSuccessfully created vm interface " + strconv.FormatInt(res.Payload.ID, 10))

	var (
		ifId       = res.Payload.ID
		objectType = "virtualization.vminterface"
	)

	//Verify if ip already exists
	ipAlreadyExist := ipam.IpamIPAddressesListParams{
		Address: &msg.IpAddress,
	}
	req, err := n.Client.Ipam.IpamIPAddressesList(&ipAlreadyExist, nil)
	if err != nil {
		return fmt.Errorf("error checking ip addresses existance : %w", err)
	}
	var (
		zero = int64(0)
		one  = int64(1)
	)

	//We dont have that ip registered on netbox, so lets create him
	if req.Payload.Count == &zero {
		//Set ip to the interface
		ip := models.WritableIPAddress{
			Address:            &msg.IpAddress,
			AssignedObjectID:   &ifId,
			AssignedObjectType: &objectType,
			Status:             models.IPAddressStatusValueActive,
		}
		ipParam := ipam.NewIpamIPAddressesCreateParams().WithData(&ip)
		r, err := n.Client.Ipam.IpamIPAddressesCreate(ipParam, nil)
		if err != nil {
			return fmt.Errorf("error creating ip address: %w", err)
		}
		util.Success("\tSuccessfully created vm management ip : " + strconv.FormatInt(r.Payload.ID, 10))
	} else if req.Payload.Count == &one {
		ip := req.Payload.Results[0]

		linkedInterfaceId := ip.AssignedObjectID

		//Si l'ip n'est pas liée à une interface
		//On l'assigne à l'interface de la machine et zou
		if linkedInterfaceId == nil {
			return n.changeIPInterface(msg, ifId, objectType)
		}

		//Sinon on vérifie sie la VM possède d'autres IP sur l'interface de management
		interfaceId := *linkedInterfaceId
		vmInterfaceParam := virtualization.NewVirtualizationInterfacesReadParams().WithID(interfaceId)
		vmInterfaceResult, err := n.Client.Virtualization.VirtualizationInterfacesRead(vmInterfaceParam, nil)
		if err != nil {
			return fmt.Errorf("error reading virtual machine interface: %w", err)
		}

		vmID := strconv.FormatInt(vmInterfaceResult.Payload.VirtualMachine.ID, 10)

		nestedVmParams := virtualization.VirtualizationInterfacesListParams{
			Name:             &mgmtInterfaceName,
			VirtualMachineID: &vmID,
		}
		nestedVmInterfaces, err := n.Client.Virtualization.VirtualizationInterfacesList(&nestedVmParams, nil)
		if err != nil {
			return fmt.Errorf("error listing virtual machine interfaces: %w", err)
		}

		mgmtInterface := nestedVmInterfaces.Payload.Results[0]
		if mgmtInterface.CountIpaddresses > 1 {
			//L'interface possède d'autres IPs
			//Du coup, on prend l'ip en question
			util.Info("Remove the link ...")
			err := n.changeIPInterface(msg, ifId, objectType)
			if err != nil {
				return err
			}
			util.Success("IP changed of interface")

			return nil
		} //Sinon on laisse l'ip sur la VM

		util.Warn("Trying to using existing IP on VM interface #" + strconv.FormatInt(mgmtInterface.ID, 10))
	}

	return nil
}

func (n *Netbox) UpdateVM(id int64, msg Message) error {
	vm := getVm(msg)

	//authContext

	updateParams := &virtualization.VirtualizationVirtualMachinesPartialUpdateParams{
		Data: &vm,
		ID:   id,
	}

	_, err := n.Client.Virtualization.VirtualizationVirtualMachinesPartialUpdate(updateParams.WithTimeout(time.Duration(30)*time.Second), nil)

	return err
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
		NewVirtualizationVirtualMachinesListParams()
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
