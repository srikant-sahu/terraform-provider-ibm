// Copyright IBM Corp. 2021 All Rights Reserved.
// Licensed under the Mozilla Public License v2.0

package eventstreams_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/IBM-Cloud/bluemix-go/models"
	acc "github.com/IBM-Cloud/terraform-provider-ibm/ibm/acctest"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/conns"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	sourceInstanceName = getTestInstanceName(mzrKey) //ES Preprod Pipeline MZR
)

func TestAccIBMEventStreamsMirroringConfigDataSource(t *testing.T) {
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
				Config: testAccCheckIBMEventStreamsMirroringConfigDataSource(sourceInstanceName, targetInstanceName, serviceName, planID, location, parameters),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIBMEventStreamsMirroringConfigProperties("data.ibm_event_streams_mirroring_config.es_mirroring_config"),
					resource.TestCheckResourceAttr("data.ibm_event_streams_mirroring_config.es_mirroring_config", "mirroring_topic_patterns.#", "2"),
					resource.TestCheckResourceAttr("data.ibm_event_streams_mirroring_config.es_mirroring_config", "mirroring_topic_patterns.0", "topicA"),
					resource.TestCheckResourceAttr("data.ibm_event_streams_mirroring_config.es_mirroring_config", "mirroring_topic_patterns.1", "topicB"),
				),
			},
		},
	})
}

// check properties of the mirroring config data source or resource object
func testAccCheckIBMEventStreamsMirroringConfigProperties(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		mcID := rs.Primary.ID
		if mcID == "" {
			return fmt.Errorf("[ERROR] Mirroring config ID is not set")
		}
		if !strings.HasSuffix(mcID, ":mirroring-config:") {
			return fmt.Errorf("[ERROR] Mirroring config ID %s not expected CRN", mcID)
		}
		return nil
	}
}
func createPlatformResourcesWithMirroring(sourceInstanceName, targetInstanceName, serviceName, planID, location string, params map[string]string) string {
	// create enterprise instance with mirroring
	return fmt.Sprintf(`
data "ibm_resource_group" "group" {
  is_default = true
}
data "ibm_resource_instance" "es_source_instance" {
  resource_group_id = data.ibm_resource_group.group.id
  name              = "%s"
}
resource "ibm_resource_instance" "es_target_instance" {
  name              = "%s"
  service           = "%s"
  plan              = "%s"
  location          = "%s"
  resource_group_id = data.ibm_resource_group.group.id
  parameters_json = jsonencode(
    {
      service-endpoints 	= "%s"
	  private_ip_allowlist 	= "%s"
      throughput        	= "%s"
      storage_size      	= "%s"
	  kms_key_crn       	= "%s"
      mirroring = {
        source_crn   = data.ibm_resource_instance.es_source_instance.id
        source_alias = "%s"
        target_alias = "%s"
      }
    }
  )
  timeouts {
    create = "4h"
    update = "1h"
    delete = "15m"
  }
}
# setup s2s policy between source and target instance
resource "ibm_iam_authorization_policy" "instance-policy" {
  source_service_name         = "%s"
  source_resource_instance_id = resource.ibm_resource_instance.es_target_instance.guid
  target_service_name         = "%s"
  target_resource_instance_id = data.ibm_resource_instance.es_source_instance.guid
  roles                       = ["Reader"]
  description                 = "test mirroring setup via terraform"
}
resource "ibm_event_streams_mirroring_config" "es-config" {
  resource_instance_id     = resource.ibm_resource_instance.es_target_instance.id
  mirroring_topic_patterns = ["topicA", "topicB"]
}`, sourceInstanceName, targetInstanceName, serviceName, planID, location,
		params["service-endpoints"], params["private_ip_allowlist"], params["throughput"],
		params["storage_size"], params["kms_key_crn"], params["source_alias"], params["target_alias"], serviceName, serviceName)
}

func testAccCheckIBMEventStreamsMirroringConfigDataSource(sourceInstanceName, targetInstanceName, serviceName, planID, location string, params map[string]string) string {
	return createPlatformResourcesWithMirroring(sourceInstanceName, targetInstanceName, serviceName, planID, location, params) + "\n" +
		`
data "ibm_event_streams_mirroring_config" "es_mirroring_config" {
  resource_instance_id = resource.ibm_resource_instance.es_target_instance.id
  depends_on = [ ibm_event_streams_mirroring_config.es-config ]
}`
}

func testAccCheckIBMEventStreamsMirroringConfigDestroy(s *terraform.State) error {
	rsContClient, err := acc.TestAccProvider.Meta().(conns.ClientSession).ResourceControllerAPI()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ibm_resource_instance" && rs.Type != "ibm_iam_authorization_policy" {
			continue
		}
		// adding authotization policy check for mirroring instance
		if rs.Type == "ibm_iam_authorization_policy" {
			iamPolicyManagementClient, err := acc.TestAccProvider.Meta().(conns.ClientSession).IAMPolicyManagementV1API()
			if err != nil {
				return err
			}
			authPolicyID := rs.Primary.ID

			getPolicyOptions := iamPolicyManagementClient.NewGetPolicyOptions(
				authPolicyID,
			)
			destroyedPolicy, response, err := iamPolicyManagementClient.GetPolicy(getPolicyOptions)

			if err == nil && *destroyedPolicy.State != "deleted" {
				return fmt.Errorf("Authorization policy still exists: %s\n", rs.Primary.ID)
			} else if response.StatusCode != 404 && destroyedPolicy.State != nil && *destroyedPolicy.State != "deleted" {
				return fmt.Errorf("[ERROR] Error waiting for authorization policy (%s) to be destroyed: %s", rs.Primary.ID, err)
			}
		}
		if rs.Type == "ibm_resource_instance" && rs.Primary.Attributes["name"] != sourceInstanceName {
			instanceID := rs.Primary.ID
			instance, err := rsContClient.ResourceServiceInstance().GetInstance(instanceID)

			if err == nil {
				if !reflect.DeepEqual(instance, models.ServiceInstance{}) && instance.State == "active" {
					return fmt.Errorf("Instance still exists: %s", rs.Primary.ID)
				}
			} else {
				if !strings.Contains(err.Error(), "ResourceServiceInstanceDoesnotExist") {
					return fmt.Errorf("[ERROR] Error checking if instance (%s) has been destroyed: %s", rs.Primary.ID, err)
				}
			}
		}
	}
	return nil
}
