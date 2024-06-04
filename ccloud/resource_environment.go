package ccloud

import (
	"context"
	"log"

	ccloud "github.com/cgroschupp/go-client-confluent-cloud/confluentcloud"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func environmentResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: environmentCreate,
		ReadContext:   environmentRead,
		UpdateContext: environmentUpdate,
		DeleteContext: environmentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    false,
				Description: "The name of the environment",
			},
		},
	}
}

func environmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(Client)

	name := d.Get("name").(string)

	log.Printf("[INFO] Creating Environment %s", name)
	orgID, err := getOrganizationID(c.confluentcloudClient)
	if err != nil {
		return diag.FromErr(err)
	}

	env, err := c.confluentcloudClient.CreateEnvironment(name, orgID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(env.ID)

	return nil
}

func environmentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(Client)

	newName := d.Get("name").(string)

	log.Printf("[INFO] Updating Environment %s", d.Id())
	orgID, err := getOrganizationID(c.confluentcloudClient)
	if err != nil {
		return diag.FromErr(err)
	}

	env, err := c.confluentcloudClient.UpdateEnvironment(d.Id(), newName, orgID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(env.ID)

	return nil
}

func environmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(Client)

	log.Printf("[INFO] Reading Environment %s", d.Id())
	env, err := c.confluentcloudClient.GetEnvironment(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("name", env.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func environmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(Client)

	log.Printf("[INFO] Deleting Environment %s", d.Id())
	err := c.confluentcloudClient.DeleteEnvironment(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func getOrganizationID(client *ccloud.Client) (int, error) {
	userData, err := client.Me()
	if err != nil {
		return 0, err
	}

	return userData.Account.OrganizationID, nil
}
