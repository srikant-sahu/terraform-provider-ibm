// Copyright IBM Corp. 2021 All Rights Reserved.
// Licensed under the Mozilla Public License v2.0

package eventstreams_test

import (
	"fmt"
	"testing"

	acc "github.com/IBM-Cloud/terraform-provider-ibm/ibm/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccIBMEventStreamsMirroringConfigResource(t *testing.T) {
	targetInstanceName := fmt.Sprintf("terraform_support_%d", acctest.RandInt())
	planID := "enterprise-3nodes-2tb"
	serviceName := "messagehub"
	location := "eu-gb"
	parameters := map[string]string{
		"service-endpoints":    "public-and-private",
		"private_ip_allowlist": "[9.0.0.0/8]", // allowing jenkins access
		"throughput":           "150",
		"storage_size":         "256",
		"kms_key_crn":          "crn:v1:staging:public:kms:us-south:a/6db1b0d0b5c54ee5c201552547febcd8:0aa69b09-941b-41b2-bbf9-9f9f0f6a6f79:key:dd37a0b6-eff4-4708-8459-e29ae0a8f256", //preprod-byok-customer-key from KMS instance keyprotect-preprod-customer-keys
		"target_alias":         "target-cluster",
		"source_alias":         "source-cluster",
	}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acc.TestAccPreCheck(t) },
		Providers:    acc.TestAccProviders,
		CheckDestroy: testAccCheckIBMEventStreamsMirroringConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: createPlatformResourcesWithMirroring(sourceInstanceName, targetInstanceName, serviceName, planID, location, parameters),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIBMEventStreamsMirroringConfigProperties("resource.ibm_event_streams_mirroring_config.es_mirroring_config"),
					resource.TestCheckResourceAttr("resource.ibm_event_streams_mirroring_config.es_mirroring_config", "mirroring_topic_patterns.#", "2"),
					resource.TestCheckResourceAttr("resource.ibm_event_streams_mirroring_config.es_mirroring_config", "mirroring_topic_patterns.0", "topicA"),
					resource.TestCheckResourceAttr("resource.ibm_event_streams_mirroring_config.es_mirroring_config", "mirroring_topic_patterns.1", "topicB"),
				),
			},
		},
	})
}
