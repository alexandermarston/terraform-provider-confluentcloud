package ccloud

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ignoreConnectorConfigs() []string {
	return []string{
		"config.kafka.endpoint",
		"config.kafka.region",
		"config.kafka.dedicated",
		"config.cloud.provider",
		"config.cloud.environment",
		"config.valid.kafka.api.key",
		"config.schema.registry.url",
	}
}

func connectorResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: connectorCreate,
		ReadContext:   connectorRead,
		UpdateContext: connectorUpdate,
		DeleteContext: connectorDelete,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
		},
		Importer: &schema.ResourceImporter{
			StateContext: connectorImport,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the connector",
			},
			"environment_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of containing environment, e.g. env-abc123",
			},
			"cluster_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of containing cluster, e.g. lkc-abc123",
			},
			"config": {
				Type:        schema.TypeMap,
				Required:    true,
				ForceNew:    false,
				Description: "Type-specific Configuration of connector. String keys and values",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// ignore common auto-generated config fields
					for _, ik := range ignoreConnectorConfigs() {
						if ik == k {
							return true
						}
					}

					if strings.HasPrefix(k, "config.internal.") {
						return true
					}

					// Sensitive data are not returned from status query, so changes made to it outside of terraform could not be tracked
					masked, _ := regexp.MatchString("\\*+", old)
					if masked {
						return true
					}

					return false
				},
			},
			"config_sensitive": {
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    false,
				Sensitive:   true,
				Description: "Sensitive part of connector configuration. String keys and values",
			},
		},
	}
}

func connectorUpdate(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(Client)

	name := d.Get("name").(string)
	config := d.Get("config").(map[string]interface{})
	configSensitive := d.Get("config_sensitive").(map[string]interface{})
	accountID := d.Get("environment_id").(string)
	clusterID := d.Get("cluster_id").(string)

	log.Printf("[DEBUG] Updating connector config")
	configStrings := make(map[string]string)
	for key, value := range config {
		configStrings[key] = value.(string)
	}
	for key, value := range configSensitive {
		configStrings[key] = value.(string)
	}

	_, err := c.confluentcloudClient.UpdateConnectorConfig(accountID, clusterID, name, configStrings)
	d.SetId(name)

	if err != nil {
		log.Printf("[ERROR] updateConnector failed %s, %v, %s", name, config, err)
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] Updated connector %s in cluster %s", name, clusterID)

	return nil
}

func connectorCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(Client)

	name := d.Get("name").(string)
	config := d.Get("config").(map[string]interface{})
	configSensitive := d.Get("config_sensitive").(map[string]interface{})
	accountID := d.Get("environment_id").(string)
	clusterID := d.Get("cluster_id").(string)

	log.Printf("[DEBUG] Creating connector")
	configStrings := make(map[string]string)
	for key, value := range config {
		configStrings[key] = value.(string)
	}
	for key, value := range configSensitive {
		configStrings[key] = value.(string)
	}

	return diag.FromErr(resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		_, err := c.confluentcloudClient.CreateConnector(accountID, clusterID, name, configStrings)

		if err != nil {
			if !strings.Contains(err.Error(), "provisioning") {
				return resource.NonRetryableError(fmt.Errorf("createConnector failed %s, %v, %s", name, config, err))
			}
			return resource.RetryableError(fmt.Errorf("API Key is still being provisioned, waiting for provisioning"))
		}

		d.SetId(name)
		return nil
	}))
}

func connectorDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(Client)
	name := d.Get("name").(string)
	accountID := d.Get("environment_id").(string)
	clusterID := d.Get("cluster_id").(string)

	var diags diag.Diagnostics

	if err := c.confluentcloudClient.DeleteConnector(accountID, clusterID, name); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func connectorImport(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	idsAndName := d.Id()
	parts := strings.Split(idsAndName, "/")

	var err error
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid format for connector import: expected '<env ID>/<cluster ID>/<name>'")
	}

	d.SetId(parts[2])
	err = d.Set("environment_id", parts[0])
	if err != nil {
		return nil, err
	}
	err = d.Set("cluster_id", parts[1])
	if err != nil {
		return nil, err
	}
	err = d.Set("name", parts[2])
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func connectorRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(Client)
	accountID := d.Get("environment_id").(string)
	clusterID := d.Get("cluster_id").(string)
	name := d.Id()

	connector, err := c.confluentcloudClient.GetConnector(accountID, clusterID, name)
	if err == nil {
		err = d.Set("config", connector.Config)
	}
	if err == nil {
		err = d.Set("name", connector.Name)
	}

	return diag.FromErr(err)
}
