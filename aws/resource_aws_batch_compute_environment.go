package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsBatchComputeEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsBatchComputeEnvironmentCreate,
		Read:   resourceAwsBatchComputeEnvironmentRead,
		Update: resourceAwsBatchComputeEnvironmentUpdate,
		Delete: resourceAwsBatchComputeEnvironmentDelete,

		Schema: map[string]*schema.Schema{
			"compute_environment_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateBatchName,
			},
			"compute_resources": {
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 0,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bid_percentage": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"desired_vcpus": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"ec2_key_pair": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"image_id": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"instance_role": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validateArn,
						},
						"instance_type": {
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"max_vcpus": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"min_vcpus": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"security_group_ids": {
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"spot_iam_fleet_role": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateArn,
						},
						"subnets": {
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"tags": tagsSchema(),
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice([]string{batch.CRTypeEc2, batch.CRTypeSpot}, true),
						},
					},
				},
			},
			"service_role": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},
			"state": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{batch.CEStateEnabled, batch.CEStateDisabled}, true),
				Default:      batch.CEStateEnabled,
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{batch.CETypeManaged, batch.CETypeUnmanaged}, true),
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ecc_cluster_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status_reason": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsBatchComputeEnvironmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn

	computeEnvironmentName := d.Get("compute_environment_name").(string)

	serviceRole := d.Get("service_role").(string)
	computeEnvironmentType := d.Get("type").(string)

	input := &batch.CreateComputeEnvironmentInput{
		ComputeEnvironmentName: aws.String(computeEnvironmentName),
		ServiceRole:            aws.String(serviceRole),
		Type:                   aws.String(computeEnvironmentType),
	}

	if v, ok := d.GetOk("state"); ok {
		input.State = aws.String(v.(string))
	}

	if computeEnvironmentType == batch.CETypeManaged {
		computeResources := d.Get("compute_resources").([]interface{})
		if len(computeResources) == 0 {
			return fmt.Errorf("One compute environment is expected, but no compute environments are set")
		}
		computeResource := computeResources[0].(map[string]interface{})

		instanceRole := computeResource["instance_role"].(string)
		maxvCpus := int64(computeResource["max_vcpus"].(int))
		minvCpus := int64(computeResource["min_vcpus"].(int))
		computeResourceType := computeResource["type"].(string)

		var instanceTypes []*string
		for _, v := range computeResource["instance_type"].(*schema.Set).List() {
			instanceTypes = append(instanceTypes, aws.String(v.(string)))
		}

		var securityGroupIds []*string
		for _, v := range computeResource["security_group_ids"].(*schema.Set).List() {
			securityGroupIds = append(securityGroupIds, aws.String(v.(string)))
		}

		var subnets []*string
		for _, v := range computeResource["subnets"].(*schema.Set).List() {
			subnets = append(subnets, aws.String(v.(string)))
		}

		input.ComputeResources = &batch.ComputeResource{
			InstanceRole:     aws.String(instanceRole),
			InstanceTypes:    instanceTypes,
			MaxvCpus:         aws.Int64(maxvCpus),
			MinvCpus:         aws.Int64(minvCpus),
			SecurityGroupIds: securityGroupIds,
			Subnets:          subnets,
			Type:             aws.String(computeResourceType),
		}

		if v, ok := computeResource["bid_percentage"]; ok {
			input.ComputeResources.BidPercentage = aws.Int64(int64(v.(int)))
		}
		if v, ok := computeResource["desired_vcpus"]; ok {
			input.ComputeResources.DesiredvCpus = aws.Int64(int64(v.(int)))
		}
		if v, ok := computeResource["ec2_key_pair"]; ok {
			input.ComputeResources.Ec2KeyPair = aws.String(v.(string))
		}
		if v, ok := computeResource["image_id"]; ok {
			input.ComputeResources.ImageId = aws.String(v.(string))
		}
		if v, ok := computeResource["spot_iam_fleet_role"]; ok {
			input.ComputeResources.SpotIamFleetRole = aws.String(v.(string))
		}
		if v, ok := computeResource["tags"]; ok {
			input.ComputeResources.Tags = tagsFromMapGeneric(v.(map[string]interface{}))
		}
	}

	log.Printf("[DEBUG] Create compute environment %s.\n", input)

	if _, err := conn.CreateComputeEnvironment(input); err != nil {
		return err
	}

	d.SetId(computeEnvironmentName)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{batch.CEStatusCreating},
		Target:     []string{batch.CEStatusValid},
		Refresh:    resourceAwsBatchComputeEnvironmentStatusRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 5 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	return resourceAwsBatchComputeEnvironmentRead(d, meta)
}

func resourceAwsBatchComputeEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn

	computeEnvironmentName := d.Get("compute_environment_name").(string)

	input := &batch.DescribeComputeEnvironmentsInput{
		ComputeEnvironments: []*string{
			aws.String(computeEnvironmentName),
		},
	}

	log.Printf("[DEBUG] Read compute environment %s.\n", input)

	result, err := conn.DescribeComputeEnvironments(input)
	if err != nil {
		return err
	}

	if len(result.ComputeEnvironments) == 0 {
		return fmt.Errorf("One compute environment is expected, but AWS return no compute environment")
	}
	computeEnvironment := result.ComputeEnvironments[0]

	d.Set("service_role", computeEnvironment.ServiceRole)
	d.Set("state", computeEnvironment.State)
	d.Set("type", computeEnvironment.Type)

	if *(computeEnvironment.Type) == "MANAGED" {
		d.Set("compute_resources", flattenComputeResources(computeEnvironment.ComputeResources))
	}

	d.Set("arn", computeEnvironment.ComputeEnvironmentArn)
	d.Set("ecc_cluster_arn", computeEnvironment.ComputeEnvironmentArn)
	d.Set("status", computeEnvironment.Status)
	d.Set("status_reason", computeEnvironment.StatusReason)

	return nil
}

func flattenComputeResources(computeResource *batch.ComputeResource) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)
	m := make(map[string]interface{})

	m["bid_percentage"] = computeResource.BidPercentage
	m["desired_vcpus"] = computeResource.DesiredvCpus
	m["ec2_key_pair"] = computeResource.Ec2KeyPair
	m["image_id"] = computeResource.ImageId
	m["instance_role"] = computeResource.InstanceRole
	m["instance_type"] = schema.NewSet(schema.HashString, flattenStringList(computeResource.InstanceTypes))
	m["max_vcpus"] = computeResource.MaxvCpus
	m["min_vcpus"] = computeResource.MinvCpus
	m["security_group_ids"] = schema.NewSet(schema.HashString, flattenStringList(computeResource.SecurityGroupIds))
	m["spot_iam_fleet_role"] = computeResource.SpotIamFleetRole
	m["subnets"] = schema.NewSet(schema.HashString, flattenStringList(computeResource.Subnets))
	m["tags"] = tagsToMapGeneric(computeResource.Tags)
	m["type"] = computeResource.Type

	result = append(result, m)
	return result
}

func resourceAwsBatchComputeEnvironmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn

	computeEnvironmentName := d.Get("compute_environment_name").(string)

	updateInput := &batch.UpdateComputeEnvironmentInput{
		ComputeEnvironment: aws.String(computeEnvironmentName),
		State:              aws.String(batch.CEStateDisabled),
	}

	log.Printf("[DEBUG] Delete compute environment %s.\n", updateInput)

	if _, err := conn.UpdateComputeEnvironment(updateInput); err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{batch.CEStatusUpdating},
		Target:     []string{batch.CEStatusValid},
		Refresh:    resourceAwsBatchComputeEnvironmentStatusRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 5 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	input := &batch.DeleteComputeEnvironmentInput{
		ComputeEnvironment: aws.String(computeEnvironmentName),
	}

	if _, err := conn.DeleteComputeEnvironment(input); err != nil {
		return err
	}

	stateConfForDelete := &resource.StateChangeConf{
		Pending:    []string{batch.CEStatusDeleting},
		Target:     []string{batch.CEStatusDeleted},
		Refresh:    resourceAwsBatchComputeEnvironmentDeleteRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 5 * time.Second,
	}
	if _, err := stateConfForDelete.WaitForState(); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourceAwsBatchComputeEnvironmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn

	computeEnvironmentName := d.Get("compute_environment_name").(string)

	input := &batch.UpdateComputeEnvironmentInput{
		ComputeEnvironment: aws.String(computeEnvironmentName),
		ComputeResources:   &batch.ComputeResourceUpdate{},
	}

	if d.HasChange("service_role") {
		input.ServiceRole = aws.String(d.Get("service_role").(string))
	}
	if d.HasChange("state") {
		input.ServiceRole = aws.String(d.Get("state").(string))
	}

	if d.HasChange("compute_resources") {
		computeResources := d.Get("compute_resources").([]interface{})
		if len(computeResources) == 0 {
			return fmt.Errorf("One compute environment is expected, but no compute environments are set")
		}
		computeResource := computeResources[0].(map[string]interface{})

		input.ComputeResources.DesiredvCpus = aws.Int64(int64(computeResource["desired_vcpus"].(int)))
		input.ComputeResources.MaxvCpus = aws.Int64(int64(computeResource["max_vcpus"].(int)))
		input.ComputeResources.MinvCpus = aws.Int64(int64(computeResource["min_vcpus"].(int)))
	}

	log.Printf("[DEBUG] Update compute environment %s.\n", input)

	if _, err := conn.UpdateComputeEnvironment(input); err != nil {
		return err
	}

	return resourceAwsBatchComputeEnvironmentRead(d, meta)
}

func resourceAwsBatchComputeEnvironmentStatusRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).batchconn

		computeEnvironmentName := d.Get("compute_environment_name").(string)

		result, err := conn.DescribeComputeEnvironments(&batch.DescribeComputeEnvironmentsInput{
			ComputeEnvironments: []*string{
				aws.String(computeEnvironmentName),
			},
		})
		if err != nil {
			return nil, "failed", err
		}

		if len(result.ComputeEnvironments) == 0 {
			return nil, "failed", fmt.Errorf("One compute environment is expected, but AWS return no compute environment")
		}

		computeEnvironment := result.ComputeEnvironments[0]
		return result, *(computeEnvironment.Status), nil
	}
}

func resourceAwsBatchComputeEnvironmentDeleteRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).batchconn

		computeEnvironmentName := d.Get("compute_environment_name").(string)

		result, err := conn.DescribeComputeEnvironments(&batch.DescribeComputeEnvironmentsInput{
			ComputeEnvironments: []*string{
				aws.String(computeEnvironmentName),
			},
		})
		if err != nil {
			return nil, "failed", err
		}

		if len(result.ComputeEnvironments) == 0 {
			return result, batch.CEStatusDeleted, nil
		}

		computeEnvironment := result.ComputeEnvironments[0]
		return result, *(computeEnvironment.Status), nil
	}
}
