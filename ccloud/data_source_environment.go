package ccloud

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func environmentDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: environmentDataSourceRead,
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

func environmentDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(Client)

	name := d.Get("name").(string)
	log.Printf("[INFO] Reading Environment %s", name)
	environments, err := c.confluentcloudClient.ListEnvironments()
	if err != nil {
		return diag.FromErr(err)
	}

	for _, environment := range environments {
		if environment.Name == name {
			d.SetId(environment.ID)
			d.Set("name", environment.Name)

			return nil
		}
	}

	return nil
}
