// Copyright IBM Corp. 2024 All Rights Reserved.
// Licensed under the Mozilla Public License v2.0

package eventstreams

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/conns"
	"github.com/IBM/eventstreams-go-sdk/pkg/adminrestv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	mirroringConfigResourceType = "mirroring-config"
)

// The mirroring config for topic selection in an Event Streams service instance.
// The ID is the CRN with the last two components "mirroring-config:".
// The mirroring topic patterns defines the topic selection.
func DataSourceIBMEventStreamsMirroringConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceIBMEventStreamsMirroringConfigRead,
		Schema: map[string]*schema.Schema{
			"resource_instance_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID or CRN of the Event Streams service instance",
			},
			"mirroring_topic_patterns": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The topic pattern to use for mirroring",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}

}

// read mirroring config using the admin-rest API
func dataSourceIBMEventStreamsMirroringConfigRead(context context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	adminrestClient, err := meta.(conns.ClientSession).ESadminRestSession()
	if err != nil {
		return diag.FromErr(err)
	}

	adminURL, instanceCRN, err := getMirroringConfigInstanceURL(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	adminrestClient.SetServiceURL(adminURL)

	getMirroringConfigOptions := &adminrestv1.GetMirroringTopicSelectionOptions{}
	mirroringConfig, response, err := adminrestClient.GetMirroringTopicSelectionWithContext(context, getMirroringConfigOptions)
	if err != nil {
		log.Printf("[DEBUG] GetMirroringTopicSelectionWithContext failed with error: %s and response:\n%s", err, response)
		return diag.FromErr(err)
	}
	if mirroringConfig == nil {
		return diag.FromErr(fmt.Errorf("[ERROR] Unexpected nil config when getting mirroring topic selection"))
	}
	d.SetId(getMirroringConfigID(instanceCRN))
	d.Set("resource_instance_id", instanceCRN)
	d.Set("mirroring_topic_patterns", mirroringConfig.Includes)
	return nil
}

func getMirroringConfigInstanceURL(d *schema.ResourceData, meta interface{}) (string, string, error) {
	instanceCRN := d.Get("resource_instance_id").(string)
	if instanceCRN == "" { // importing
		id := d.Id()
		crnSegments := strings.Split(id, ":")
		if len(crnSegments) != 10 || crnSegments[8] != mirroringConfigResourceType {
			return "", "", fmt.Errorf("ID '%s' is not a mirroring config resource", id)
		}
		crnSegments[8] = ""
		crnSegments[9] = ""
		instanceCRN = strings.Join(crnSegments, ":")
		d.Set("resource_instance_id", instanceCRN)
	}

	instance, err := getInstanceDetails(instanceCRN, meta)
	if err != nil {
		return "", "", err
	}
	adminURL := instance.Extensions["kafka_http_url"].(string)
	planID := *instance.ResourcePlanID
	valid := strings.Contains(planID, "enterprise")
	if !valid {
		return "", "", fmt.Errorf("mirroring config is not supported by the Event Streams %s plan, enterprise plan is expected",
			planID)
	}
	return adminURL, instanceCRN, nil
}

func getMirroringConfigID(instanceCRN string) string {
	crnSegments := strings.Split(instanceCRN, ":")
	crnSegments[8] = mirroringConfigResourceType
	crnSegments[9] = ""
	return strings.Join(crnSegments, ":")
}
