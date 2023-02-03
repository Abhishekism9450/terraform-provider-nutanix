package nutanix

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-nutanix/client/era"
	"github.com/terraform-providers/terraform-provider-nutanix/utils"
)

var (
	EraDBProvisionTimeout = 30 * time.Minute
)

func resourceNutanixNDBServerVM() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNutanixNDBServerVMCreate,
		ReadContext:   resourceNutanixNDBServerVMRead,
		UpdateContext: resourceNutanixNDBServerVMUpdate,
		DeleteContext: resourceNutanixNDBServerVMDelete,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(EraDBProvisionTimeout),
		},
		Schema: map[string]*schema.Schema{
			"database_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"software_profile_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"time_machine_id"},
				RequiredWith:  []string{"software_profile_version_id"},
			},
			"software_profile_version_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"time_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"software_profile_id"},
			},
			"snapshot_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"timezone": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"network_profile_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"compute_profile_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"nx_cluster_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"vm_password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"latest_snapshot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"postgres_database": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vm_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"client_public_key": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"credentials": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"username": {
							Type:     schema.TypeString,
							Required: true,
						},
						"password": {
							Type:     schema.TypeString,
							Required: true,
						},
						"label": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"maintenance_tasks": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"maintenance_window_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"tasks": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"task_type": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringInSlice([]string{"OS_PATCHING", "DB_PATCHING"}, false),
									},
									"pre_command": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"post_command": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},

			// computed
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"properties": {
				Type:        schema.TypeList,
				Description: "List of all the properties",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "",
						},

						"value": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "",
						},
					},
				},
			},
			"tags": dataSourceEraDBInstanceTags(),
			"era_created": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"internal": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"dbserver_cluster_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vm_cluster_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vm_cluster_uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ip_addresses": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"fqdns": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"mac_addresses": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"placeholder": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"client_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"era_drive_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"era_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vm_timezone": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceNutanixNDBServerVMCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*Client).Era

	req := &era.DBServerInputRequest{}

	// build request for dbServerVMs
	if err := buildDBServerVMRequest(d, req); err != nil {
		return diag.FromErr(err)
	}

	// api to create request

	resp, err := conn.Service.CreateDBServerVM(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resp.Entityid)

	// Get Operation ID from response of Response and poll for the operation to get completed.
	opID := resp.Operationid
	if opID == "" {
		return diag.Errorf("error: operation ID is an empty string")
	}
	opReq := era.GetOperationRequest{
		OperationID: opID,
	}

	log.Printf("polling for operation with id: %s\n", opID)

	// Poll for operation here - Operation GET Call
	stateConf := &resource.StateChangeConf{
		Pending: []string{"PENDING"},
		Target:  []string{"COMPLETED", "FAILED"},
		Refresh: eraRefresh(ctx, conn, opReq),
		Timeout: d.Timeout(schema.TimeoutCreate),
		Delay:   eraDelay,
	}

	if _, errWaitTask := stateConf.WaitForStateContext(ctx); errWaitTask != nil {
		return diag.Errorf("error waiting for db Server VM (%s) to create: %s", resp.Entityid, errWaitTask)
	}
	log.Printf("NDB database Server VM with %s id is created successfully", d.Id())
	return resourceNutanixNDBServerVMRead(ctx, d, meta)
}

func resourceNutanixNDBServerVMRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*Client).Era

	resp, err := conn.Service.ReadDBServerVM(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("description", resp.Description); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("name", resp.Name); err != nil {
		return diag.FromErr(err)
	}

	props := []interface{}{}
	for _, prop := range resp.Properties {
		props = append(props, map[string]interface{}{
			"name":  prop.Name,
			"value": prop.Value,
		})
	}
	if err := d.Set("properties", props); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("tags", flattenDBTags(resp.Tags)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("era_created", resp.EraCreated); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("internal", resp.Internal); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("dbserver_cluster_id", resp.DbserverClusterID); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("vm_cluster_name", resp.VMClusterName); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("vm_cluster_uuid", resp.VMClusterUUID); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("ip_addresses", utils.StringValueSlice(resp.IPAddresses)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("fqdns", resp.Fqdns); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("mac_addresses", utils.StringValueSlice(resp.MacAddresses)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("type", resp.Type); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("placeholder", resp.Placeholder); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("status", resp.Status); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("client_id", resp.ClientID); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("era_drive_id", resp.EraDriveID); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("era_version", resp.EraVersion); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("vm_timezone", resp.VMTimeZone); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceNutanixNDBServerVMUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*Client).Era

	req := &era.UpdateDBServerVMRequest{}

	// setting default values
	req.ResetName = false
	req.ResetDescription = false
	req.ResetCredential = false
	req.ResetTags = false

	if d.HasChange("description") {
		req.Description = utils.StringPtr(d.Get("description").(string))
		req.ResetDescription = true
	}

	if d.HasChange("postgres_database") {
		ps := d.Get("postgres_database").([]interface{})[0].(map[string]interface{})

		vmName := ps["vm_name"]
		req.Name = utils.StringPtr(vmName.(string))
		req.ResetName = true
	}

	if d.HasChange("tags") {
		req.Tags = expandTags(d.Get("tags").([]interface{}))
		req.ResetTags = true
	}

	if d.HasChange("credential") {
		req.ResetCredential = true

		creds := d.Get("credentials")
		credList := creds.([]interface{})

		credArgs := []*era.VMCredentials{}

		for _, v := range credList {
			val := v.(map[string]interface{})
			cred := &era.VMCredentials{}
			if username, ok := val["username"]; ok {
				cred.Username = utils.StringPtr(username.(string))
			}

			if pass, ok := val["password"]; ok {
				cred.Password = utils.StringPtr(pass.(string))
			}

			if label, ok := val["label"]; ok {
				cred.Label = utils.StringPtr(label.(string))
			}

			credArgs = append(credArgs, cred)
		}
		req.Credentials = credArgs
	}

	resp, err := conn.Service.UpdateDBServerVM(ctx, req, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if resp != nil {
		if err = d.Set("description", resp.Description); err != nil {
			return diag.FromErr(err)
		}

		if err = d.Set("name", resp.Name); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("NDB database with %s id updated successfully", d.Id())
	return nil
}

func resourceNutanixNDBServerVMDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*Client).Era

	req := era.DeleteDBServerVMRequest{
		Delete:            true,
		Remove:            false,
		SoftRemove:        false,
		DeleteVgs:         true,
		DeleteVmSnapshots: true,
	}

	res, err := conn.Service.DeleteDBServerVM(ctx, &req, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("Operation to delete dbserver vm with id %s has started, operation id: %s", d.Id(), res.Operationid)
	opID := res.Operationid
	if opID == "" {
		return diag.Errorf("error: operation ID is an empty string")
	}
	opReq := era.GetOperationRequest{
		OperationID: opID,
	}

	log.Printf("polling for operation with id: %s\n", opID)

	// Poll for operation here - Cluster GET Call
	stateConf := &resource.StateChangeConf{
		Pending: []string{"PENDING"},
		Target:  []string{"COMPLETED", "FAILED"},
		Refresh: eraRefresh(ctx, conn, opReq),
		Timeout: d.Timeout(schema.TimeoutCreate),
		Delay:   eraDelay,
	}

	if _, errWaitTask := stateConf.WaitForStateContext(ctx); errWaitTask != nil {
		return diag.Errorf("error waiting for db server VM (%s) to delete: %s", res.Entityid, errWaitTask)
	}
	log.Printf("NDB database Server VM with %s id is deleted successfully", d.Id())
	return nil
}

func buildDBServerVMRequest(d *schema.ResourceData, res *era.DBServerInputRequest) error {
	if dbType, ok := d.GetOk("database_type"); ok {
		res.DatabaseType = utils.StringPtr(dbType.(string))
	}

	if softwareProfile, ok := d.GetOk("software_profile_id"); ok {
		res.SoftwareProfileID = utils.StringPtr(softwareProfile.(string))
	}

	if softwareVersion, ok := d.GetOk("software_profile_version_id"); ok {
		res.SoftwareProfileVersionID = utils.StringPtr(softwareVersion.(string))
	}

	if LatestSnapshot, ok := d.GetOk("latest_snapshot"); ok {
		res.LatestSnapshot = LatestSnapshot.(bool)
	}

	if timeMachine, ok := d.GetOk("time_machine_id"); ok {
		res.TimeMachineId = utils.StringPtr(timeMachine.(string))

		// if snapshot id is provided
		if snapshotid, ok := d.GetOk("snapshot_id"); ok {
			res.SnapshotId = utils.StringPtr(snapshotid.(string))
			res.LatestSnapshot = false
		} else {
			res.LatestSnapshot = true
		}
	}

	if NetworkProfile, ok := d.GetOk("network_profile_id"); ok {
		res.NetworkProfileID = utils.StringPtr(NetworkProfile.(string))
	}

	if ComputeProfile, ok := d.GetOk("compute_profile_id"); ok {
		res.ComputeProfileID = utils.StringPtr(ComputeProfile.(string))
	}

	if ClusterID, ok := d.GetOk("nx_cluster_id"); ok {
		res.NxClusterID = utils.StringPtr(ClusterID.(string))
	}

	if VMPass, ok := d.GetOk("vm_password"); ok {
		res.VMPassword = utils.StringPtr(VMPass.(string))
	}

	if desc, ok := d.GetOk("description"); ok {
		res.Description = utils.StringPtr(desc.(string))
	}

	if postgresDatabase, ok := d.GetOk("postgres_database"); ok && len(postgresDatabase.([]interface{})) > 0 {
		res.ActionArguments = expandDBServerPostgresInput(postgresDatabase.([]interface{}))
	}

	if maintenance, ok := d.GetOk("maintenance_tasks"); ok {
		res.MaintenanceTasks = expandMaintenanceTasks(maintenance.([]interface{}))
	}
	return nil
}

func expandDBServerPostgresInput(pr []interface{}) []*era.Actionarguments {
	if len(pr) > 0 {
		args := make([]*era.Actionarguments, 0)

		for _, v := range pr {
			val := v.(map[string]interface{})

			if vmName, ok := val["vm_name"]; ok {
				args = append(args, &era.Actionarguments{
					Name:  "vm_name",
					Value: vmName,
				})
			}
			if clientKey, ok := val["client_public_key"]; ok {
				args = append(args, &era.Actionarguments{
					Name:  "client_public_key",
					Value: clientKey,
				})
			}
		}
		return args
	}
	return nil
}
