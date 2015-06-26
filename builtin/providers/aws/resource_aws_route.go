package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

// AWS Route resource Schema delcaration
func resourceAwsRoute() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRouteCreate,
		Read:   resourceAwsRouteRead,
		Update: resourceAwsRouteUpdate,
		Delete: resourceAwsRouteDelete,

		Schema: map[string]*schema.Schema{
			"destination_cidr_block": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"destination_prefix_list_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"gateway_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"instance_owner_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"network_interface_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"origin": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"vpc_peering_connection_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsRouteCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	var numTargets int
	var setTarget string
	allowedTargets := []string{
		"gateway_id",
		"instance_id",
		"network_interface_id",
		"route_table_id",
		"vpc_peering_connection_id",
	}
	createOpts := &ec2.CreateRouteInput{}

	// Check if more than 1 target is specified
	for _, target := range allowedTargets {
		if len(d.Get(target).(string)) > 0 {
			numTargets++
			setTarget = target
		}
	}

	if numTargets > 1 {
		fmt.Errorf("Error: more than 1 target specified. Only 1 of gateway_id" +
			"instance_id, network_interface_id, route_table_id or" +
			"vpc_peering_connection_id is allowed.")
	}

	// Formulate CreateRouteInput based on the target type
	switch setTarget {
	case "gateway_id":
		createOpts = &ec2.CreateRouteInput{
			DestinationCIDRBlock: aws.String(d.Get("destination_cidr_block").(string)),
			GatewayID:            aws.String(d.Get("gateway_id").(string)),
		}
	case "instance_id":
		createOpts = &ec2.CreateRouteInput{
			DestinationCIDRBlock: aws.String(d.Get("destination_cidr_block").(string)),
			InstanceID:           aws.String(d.Get("instance_id").(string)),
		}
	case "network_interface_id":
		createOpts = &ec2.CreateRouteInput{
			DestinationCIDRBlock: aws.String(d.Get("destination_cidr_block").(string)),
			NetworkInterfaceID:   aws.String(d.Get("network_interface_id").(string)),
		}
	case "route_table_id":
		createOpts = &ec2.CreateRouteInput{
			DestinationCIDRBlock: aws.String(d.Get("destination_cidr_block").(string)),
			RouteTableID:         aws.String(d.Get("route_table_id").(string)),
		}
	case "vpc_peering_connection_id":
		createOpts = &ec2.CreateRouteInput{
			DestinationCIDRBlock:   aws.String(d.Get("destination_cidr_block").(string)),
			VPCPeeringConnectionID: aws.String(d.Get("route_table_id").(string)),
		}
	default:
		fmt.Errorf("Error: invalid target type specified.")
	}
	log.Printf("[DEBUG] Route create config: %#v", createOpts)

	// Create the route
	_, err := conn.CreateRoute(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating route: %s", err)
	}

	return resourceAwsRouteRead(d, meta)
}

func resourceAwsRouteRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	route, err := findResourceRoute(conn, d)
	if err != nil {
		return err
	}

	d.Set("destination_prefix_list_id", route.DestinationPrefixListID)
	d.Set("gateway_id", route.DestinationPrefixListID)
	d.Set("instance_id", route.InstanceID)
	d.Set("instance_owner_id", route.InstanceOwnerID)
	d.Set("network_interface_id", route.NetworkInterfaceID)
	d.Set("origin", route.Origin)
	d.Set("state", route.State)
	d.Set("vpc_peering_connection_id", route.VPCPeeringConnectionID)

	return nil
}

func resourceAwsRouteUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	replaceOpts := &ec2.ReplaceRouteInput{
		DestinationCIDRBlock:   aws.String(d.Get("destination_cidr_block").(string)),
		GatewayID:              aws.String(d.Get("gateway_id").(string)),
		InstanceID:             aws.String(d.Get("instance_id").(string)),
		NetworkInterfaceID:     aws.String(d.Get("network_interface_id").(string)),
		RouteTableID:           aws.String(d.Get("route_table_id").(string)),
		VPCPeeringConnectionID: aws.String(d.Get("vpc_peering_connection_id").(string)),
	}
	log.Printf("[DEBUG] Route replace config: %s", awsutil.StringValue(replaceOpts))

	_, err := conn.ReplaceRoute(replaceOpts)
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsRouteDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	deleteOpts := &ec2.DeleteRouteInput{
		DestinationCIDRBlock: aws.String(d.Get("destination_cidr_block").(string)),
		RouteTableID:         aws.String(d.Get("route_table_id").(string)),
	}

	_, err := conn.DeleteRoute(deleteOpts)
	if err != nil {
		return err
	}

	return nil
}

// Helper: retrieve a route
func findResourceRoute(conn *ec2.EC2, d *schema.ResourceData) (*ec2.Route, error) {
	routeTableId := d.Get("route_table_id").(string)

	findOpts := &ec2.DescribeRouteTablesInput{
		RouteTableIDs: []*string{&routeTableId},
	}

	resp, err := conn.DescribeRouteTables(findOpts)
	if err != nil {
		return nil, err
	}

	for _, route := range (*resp.RouteTables[0]).Routes {
		if *route.DestinationCIDRBlock == d.Get("destination_cidr_block").(string) {
			return route, nil
		}
	}

	return nil, nil
}
