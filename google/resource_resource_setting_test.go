package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccOrganizationResourceSetting_basic(t *testing.T) {
	t.Parallel()

	const (
		settingName = "iam-serviceAccountKeyExpiry"
	)

	vcrTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		//CheckDestroy: testAccStorageBucketDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "google_organization_resource_setting" "mysetting" {
  organization_id = "%s"
  setting_name = "%s"
  local_value {
     string_value = "1hours"
  }
}
`, getTestOrgFromEnv(t), settingName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"google_organization_resource_setting.mysetting", "setting_name", "iam-serviceAccountKeyExpiry"),
					resource.TestCheckResourceAttr(
						"google_organization_resource_setting.mysetting", "local_value.string_value", "1hours"),
				),
			},
			//{
			//	ResourceName:            "google_storage_bucket.bucket",
			//	ImportState:             true,
			//	ImportStateVerify:       true,
			//	ImportStateVerifyIgnore: []string{"force_destroy"},
			//},
			//{
			//	ResourceName:            "google_storage_bucket.bucket",
			//	ImportStateId:           fmt.Sprintf("%s/%s", getTestProjectFromEnv(), settingName),
			//	ImportState:             true,
			//	ImportStateVerify:       true,
			//	ImportStateVerifyIgnore: []string{"force_destroy"},
			//},
		},
	})
}
