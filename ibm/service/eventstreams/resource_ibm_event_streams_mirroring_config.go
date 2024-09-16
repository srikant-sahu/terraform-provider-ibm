// Copyright IBM Corp. 2024 All Rights Reserved.
// Licensed under the Mozilla Public License v2.0

package eventstreams

import (
	"context"

	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/conns"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/flex"
	"github.com/IBM/eventstreams-go-sdk/pkg/adminrestv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The mirroring config for topic selection in an Event Streams service instance.
// The ID is the CRN with the last two components "mirroring-config:".
// The mirroring topic patterns defines the topic selection.
func ResourceIBMEventStreamsMirroringConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIBMEventStreamsMirroringConfigReplace,
		ReadContext:   resourceIBMEventStreamsMirroringConfigRead,
		UpdateContext: resourceIBMEventStreamsMirroringConfigReplace,
		DeleteContext: resourceIBMEventStreamsMirroringConfigDelete,
		Importer:      &schema.ResourceImporter{},
		Schema: map[string]*schema.Schema{
			"resource_instance_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID or CRN of the Event Streams service instance",
			},
			"mirroring_topic_patterns": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "The topic pattern to use for mirroring",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceIBMEventStreamsMirroringConfigRead(context context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return dataSourceIBMEventStreamsMirroringConfigRead(context, d, meta)
}

// The mirroring topic selection for a mirroring enabled instance is always replaced,
// so create and update have the same behavior
func resourceIBMEventStreamsMirroringConfigReplace(context context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	adminrestClient, err := meta.(conns.ClientSession).ESadminRestSession()
	if err != nil {
		return diag.FromErr(err)
	}

	adminURL, _, err := getMirroringConfigInstanceURL(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	adminrestClient.SetServiceURL(adminURL)
	mirroringOptions := &adminrestv1.ReplaceMirroringTopicSelectionOptions{}
	mirroringOptions.SetIncludes(flex.ExpandStringList(d.Get("mirroring_topic_patterns").([]interface{})))

	_, _, err = adminrestClient.ReplaceMirroringTopicSelectionWithContext(context, mirroringOptions)
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceIBMEventStreamsMirroringConfigRead(context, d, meta)
}

// The mirroring config can't be deleted, but we reset with an empty list.
func resourceIBMEventStreamsMirroringConfigDelete(context context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	adminrestClient, err := meta.(conns.ClientSession).ESadminRestSession()
	if err != nil {
		return diag.FromErr(err)
	}

	adminURL, _, err := getMirroringConfigInstanceURL(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	adminrestClient.SetServiceURL(adminURL)
	mirroringOptions := &adminrestv1.ReplaceMirroringTopicSelectionOptions{}
	mirroringOptions.SetIncludes([]string{})
	_, _, err = adminrestClient.ReplaceMirroringTopicSelectionWithContext(context, mirroringOptions)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
