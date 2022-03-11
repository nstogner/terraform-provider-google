package google

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	resourceSettingsV1 "google.golang.org/api/resourcesettings/v1"
)

func resourceGoogleResourceSettings(parentType string) *schema.Resource {
	localValueKeys := []string{
		"local_value.0.boolean_value",
		"local_value.0.string_value",
		"local_value.0.enum_value",
		"local_value.0.duration_value",
	}

	return &schema.Resource{
		Create: resourceGoogleResourceSettingsCreate(parentType),
		Read:   resourceGoogleResourceSettingsRead(parentType),
		Update: resourceGoogleResourceSettingsUpdate(parentType),
		Delete: resourceGoogleResourceSettingsDelete(parentType),

		Importer: &schema.ResourceImporter{
			State: resourceGoogleResourceSettingsImportState,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(4 * time.Minute),
			Update: schema.DefaultTimeout(4 * time.Minute),
			Read:   schema.DefaultTimeout(4 * time.Minute),
			Delete: schema.DefaultTimeout(4 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			resourceSettingsParentKey(parentType): {
				Type:        schema.TypeString,
				Description: fmt.Sprintf(`The %s id the resource setting with be applied to.`, parentType),
				Required:    true,
			},
			"setting_name": {
				Type:        schema.TypeString,
				Description: `The resource settings name. For example, "gcp-enableMyFeature".`,
				Required:    true,
			},

			"local_value": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				Description: fmt.Sprintf(`The configured value of the setting at the %s.`, parentType),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"boolean_value": {
							Type:         schema.TypeBool,
							Optional:     true,
							Description:  `Holds the value for a tag field with boolean type.`,
							AtLeastOneOf: localValueKeys,
						},
						"string_value": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  `Holds the value for a tag field with string type.`,
							AtLeastOneOf: localValueKeys,
						},
						// TODO: String set.
						"enum_value": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  `The display name of the enum value.`,
							AtLeastOneOf: localValueKeys,
						},
						"duration_value": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  `TODO`,
							AtLeastOneOf: localValueKeys,
						},
						// TODO: String map
					},
				},
			},
		},
		UseJSONNumber: true,
	}
}

func resourceSettingsFullName(parentType, parentIdentifier, settingName string) string {
	return fmt.Sprintf("%ss/%s/settings/%s", parentType, parentIdentifier, settingName)
}

func resourceSettingsShortName(fullName string) string {
	split := strings.Split(fullName, "/")
	return split[len(split)-1]
}

func resourceSettingsParentKey(parentType string) string {
	return parentType + "_id"
}

func resourceGoogleResourceSettingsCreate(parentType string) func(d *schema.ResourceData, meta interface{}) error {
	return func(d *schema.ResourceData, meta interface{}) error {
		settingName := d.Get("setting_name").(string)
		parentIdentifier := d.Get(resourceSettingsParentKey(parentType)).(string)
		id := resourceSettingsFullName(parentType, parentIdentifier, settingName)

		if err := patchResourceSetting(d, meta, false, id); err != nil {
			return fmt.Errorf("Error creating: %s", err)
		}

		d.SetId(id)

		return resourceGoogleResourceSettingsRead(parentType)(d, meta)
	}
}

func resourceGoogleResourceSettingsRead(parentType string) func(d *schema.ResourceData, meta interface{}) error {
	return func(d *schema.ResourceData, meta interface{}) error {
		config := meta.(*Config)
		userAgent, err := generateUserAgentString(d, config.userAgent)
		if err != nil {
			return err
		}

		// Fetch metadata about the setting.
		settingBasic, err := config.NewResourceSettingsClient(userAgent).Organizations.Settings.
			Get(d.Id()).
			View("SETTING_VIEW_BASIC").
			Do()
		if err != nil {
			return handleNotFoundError(err, d, fmt.Sprintf("ResourceSettings Not Found : %s", d.Id()))
		}

		// Fetch the localValue field.
		settingLocal, err := config.NewResourceSettingsClient(userAgent).Organizations.Settings.
			Get(d.Id()).
			View("SETTING_VIEW_LOCAL_VALUE").
			Do()
		if err != nil {
			return handleNotFoundError(err, d, fmt.Sprintf("ResourceSettings Not Found : %s", d.Id()))
		}

		if err := d.Set("setting_name", resourceSettingsShortName(settingLocal.Name)); err != nil {
			return fmt.Errorf("Error setting setting_name: %s", err)
		}

		localValue := []map[string]interface{}{
			{},
		}
		if lv := settingLocal.LocalValue; lv != nil {
			switch settingBasic.Metadata.DataType {
			case "BOOLEAN":
				localValue[0]["boolean_value"] = settingLocal.LocalValue.BooleanValue
			case "STRING":
				localValue[0]["string_value"] = settingLocal.LocalValue.StringValue
			case "STRING_SET":
				// TODO
			case "ENUM_VALUE":
				localValue[0]["enum_value"] = settingLocal.LocalValue.EnumValue.Value
			case "DURATION_VALUE":
				localValue[0]["duration_value"] = settingLocal.LocalValue.DurationValue
			case "STRING_MAP":
				// TODO
			}
		}

		if err := d.Set("local_value", localValue); err != nil {
			return fmt.Errorf("Error setting local_value: %s", err)
		}

		return nil
	}
}

func resourceGoogleResourceSettingsUpdate(parentType string) func(d *schema.ResourceData, meta interface{}) error {
	return func(d *schema.ResourceData, meta interface{}) error {
		if err := patchResourceSetting(d, meta, false, d.Id()); err != nil {
			return fmt.Errorf("Error updating: %s", err)
		}

		return nil
	}
}

func resourceGoogleResourceSettingsDelete(parentType string) func(d *schema.ResourceData, meta interface{}) error {
	return func(d *schema.ResourceData, meta interface{}) error {
		if err := patchResourceSetting(d, meta, true, d.Id()); err != nil {
			return fmt.Errorf("Error deleting: %s", err)
		}

		return nil
	}
}

func resourceGoogleResourceSettingsImportState(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	// TODO
	//id := d.Id()

	//if !strings.HasPrefix(d.Id(), "folders/") {
	//	id = fmt.Sprintf("folders/%s", id)
	//}

	//d.SetId(id)

	return []*schema.ResourceData{d}, nil
}

// patchResourceSetting is used by Create/Update/Delete.
// Delete is implemented by setting localValue to nil/null.
func patchResourceSetting(d *schema.ResourceData, meta interface{}, unset bool, id string) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	var localValue *resourceSettingsV1.GoogleCloudResourcesettingsV1Value
	if !unset {
		localValue = &resourceSettingsV1.GoogleCloudResourcesettingsV1Value{}
		if val, ok := d.GetOk("local_value.0.boolean_value"); ok {
			localValue.BooleanValue = val.(bool)
		} else if val, ok := d.GetOk("local_value.0.string_value"); ok {
			localValue.StringValue = val.(string)
		} else if val, ok := d.GetOk("local_value.0.enum_value"); ok {
			localValue.EnumValue = &resourceSettingsV1.GoogleCloudResourcesettingsV1ValueEnumValue{Value: val.(string)}
		} else if val, ok := d.GetOk("local_value.0.duration_value"); ok {
			localValue.DurationValue = val.(string)
		}
	}

	if _, err := config.NewResourceSettingsClient(userAgent).Organizations.Settings.Patch(id, &resourceSettingsV1.GoogleCloudResourcesettingsV1Setting{
		Name:       id,
		LocalValue: localValue,
	}).Do(); err != nil {
		return fmt.Errorf("patching: %s", err)
	}

	return nil
}
