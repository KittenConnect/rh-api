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

	vm := NewVM(n, msg)
	res, err := vm.Create(msg)
	if err != nil {
		if res != nil && res.Payload != nil {
			return fmt.Errorf("error creating virtual machine: %w \n\t%s", err, res.Error())
		}

		return fmt.Errorf("error creating virtual machine: %w", err)
	}

	util.Success("Created machine ID: %d", res.Payload.ID)
	vm.NetboxId = res.Payload.ID

	//Create management interface
	r, err := vm.CreateInterface("mgmt")
	if err != nil {
		return err
	}

	var (
		ifId       = r.Payload.ID
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
			return vm.UpdateInterfaceIP(msg.IpAddress, ifId, objectType)
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
			err := vm.UpdateInterfaceIP(msg.IpAddress, ifId, objectType)
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
	vm := NewVM(n, msg)
	vm.NetboxId = id

	_, err := vm.Create(msg)

	err = vm.Update()
	if err != nil {
		return err
	}

	//Update management IP
	return vm.UpdateManagementIP(msg)
}

func (n *Netbox) CreateOrUpdateVM(msg Message) error {
	if !n._isConnected {
		return errors.New("netbox is not connected")
	}

	var vmId int64
	var err error

	// Call netbox API with specific serial, then update his settings accordingly
	//exist := contains(MachinesSerials, msg.Hostname) //TODO
	//if !exist {
	//If the vm don't exist in memory, fetch his details, if she exists in netbox
	exist, vmId, err := n.VmExists(msg.Hostname, msg.GetSerial())
	if err != nil {
		return fmt.Errorf("error checking if VM exists: %w", err)
	}

	//Create VM if she doesn't exists in netbox
	if !exist {
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

func (n *Netbox) VmExists(hostname string, serial string) (bool, int64, error) {
	//Check if the vm exist in netbox
	req := virtualization.
		NewVirtualizationVirtualMachinesListParams().
		WithTimeout(n.GetDefaultTimeout())
	res, err := n.Client.Virtualization.VirtualizationVirtualMachinesList(req, nil)
	if err != nil {
		return false, 0, fmt.Errorf("unable to get list of machines from netbox: %w", err)
	}

	for _, v := range res.Payload.Results {
		if *v.Name == hostname {
			return true, v.ID, nil
		}

		var cf = v.CustomFields.(map[string]interface{})
		var serial = ""
		_ = serial

		for k, v := range cf {
			switch c := v.(type) {
			case string:
				if k == "kc_serial_" {
					serial = c
				}
			}
		}

		if serial == serial {
			return true, v.ID, nil
		}
	}

	return false, 0, nil
}
