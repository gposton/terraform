package google

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeAddress() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeAddressCreate,
		Read:   resourceComputeAddressRead,
		Delete: resourceComputeAddressDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func getOptionalRegion(d *schema.ResourceData, config *Config) string {
	if res, ok := d.GetOk("region"); !ok {
		return config.Region
	} else {
		return res.(string)
	}
}

func resourceComputeAddressCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	region := getOptionalRegion(d, config)

	// Build the address parameter
	addr := &compute.Address{Name: d.Get("name").(string)}
	op, err := config.clientCompute.Addresses.Insert(
		config.Project, region, addr).Do()
	if err != nil {
		return fmt.Errorf("Error creating address: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(addr.Name)

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Region:  region,
		Type:    OperationWaitRegion,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for address to create: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// The resource didn't actually create
		d.SetId("")

		// Return the error
		return OperationError(*op.Error)
	}

	return resourceComputeAddressRead(d, meta)
}

func resourceComputeAddressRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	region := getOptionalRegion(d, config)

	addr, err := config.clientCompute.Addresses.Get(
		config.Project, region, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading address: %s", err)
	}

	d.Set("address", addr.Address)
	d.Set("self_link", addr.SelfLink)

	return nil
}

func resourceComputeAddressDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	region := getOptionalRegion(d, config)
	// Delete the address
	log.Printf("[DEBUG] address delete request")
	op, err := config.clientCompute.Addresses.Delete(
		config.Project, region, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting address: %s", err)
	}

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Region:  region,
		Type:    OperationWaitRegion,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for address to delete: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// Return the error
		return OperationError(*op.Error)
	}

	d.SetId("")
	return nil
}
