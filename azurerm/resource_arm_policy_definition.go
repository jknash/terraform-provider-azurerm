package azurerm

import (
	"fmt"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-12-01/policy"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmPolicyDefinition() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmPolicyDefinitionCreateUpdate,
		Update: resourceArmPolicyDefinitionCreateUpdate,
		Read:   resourceArmPolicyDefinitionRead,
		Delete: resourceArmPolicyDefinitionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"policy_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(policy.TypeBuiltIn),
					string(policy.TypeCustom),
					string(policy.TypeNotSpecified),
				}, true)},

			"mode": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(policy.All),
					string(policy.Indexed),
					string(policy.NotSpecified),
				}, true),
			},

			"display_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"policy_rule": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: structure.SuppressJsonDiff,
			},

			"meta_data": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: structure.SuppressJsonDiff,
			},

			"parameters": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: structure.SuppressJsonDiff,
			},
		},
	}
}

func resourceArmPolicyDefinitionCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).policyDefinitionsClient
	ctx := meta.(*ArmClient).StopContext

	name := d.Get("name").(string)
	policyType := d.Get("policy_type").(string)
	mode := d.Get("mode").(string)
	displayName := d.Get("display_name").(string)
	description := d.Get("description").(string)

	properties := policy.DefinitionProperties{
		PolicyType:  policy.Type(policyType),
		Mode:        policy.Mode(mode),
		DisplayName: utils.String(displayName),
		Description: utils.String(description),
	}

	if policyRuleString := d.Get("policy_rule").(string); policyRuleString != "" {
		policyRule, err := structure.ExpandJsonFromString(policyRuleString)
		if err != nil {
			return fmt.Errorf("unable to parse policy_rule: %s", err)
		}
		properties.PolicyRule = &policyRule
	}

	if metaDataString := d.Get("meta_data").(string); metaDataString != "" {
		metaData, err := structure.ExpandJsonFromString(metaDataString)
		if err != nil {
			return fmt.Errorf("unable to parse meta_data: %s", err)
		}
		properties.Metadata = &metaData
	}

	if parametersString := d.Get("parameters").(string); parametersString != "" {
		parameters, err := structure.ExpandJsonFromString(parametersString)
		if err != nil {
			return fmt.Errorf("unable to parse parameters: %s", err)
		}
		properties.Parameters = &parameters
	}

	definition := policy.Definition{
		Name:                 utils.String(name),
		DefinitionProperties: &properties,
	}

	_, err := client.CreateOrUpdate(ctx, name, definition)
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, name)
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	return resourceArmPolicyDefinitionRead(d, meta)
}

func resourceArmPolicyDefinitionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).policyDefinitionsClient
	ctx := meta.(*ArmClient).StopContext

	name, err := parsePolicyDefinitionNameFromId(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] Error reading Policy Definition %q - removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading Policy Definition %+v", err)
	}

	d.Set("name", resp.Name)
	if props := resp.DefinitionProperties; props != nil {
		d.Set("policy_type", props.PolicyType)
		d.Set("mode", props.Mode)
		d.Set("display_name", props.DisplayName)
		d.Set("description", props.Description)
		d.Set("policy_rule", props.PolicyRule)
		d.Set("meta_data", props.Metadata)
		d.Set("parameters", props.Parameters)
	}

	return nil
}

func resourceArmPolicyDefinitionDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).policyDefinitionsClient
	ctx := meta.(*ArmClient).StopContext

	name, err := parsePolicyDefinitionNameFromId(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Delete(ctx, name)

	if err != nil {
		if utils.ResponseWasNotFound(resp) {
			return nil
		}

		return fmt.Errorf("Error deleting Policy Definition %q: %+v", name, err)
	}

	return nil
}

func parsePolicyDefinitionNameFromId(id string) (string, error) {
	components := strings.Split(id, "/")

	if len(components) == 0 {
		return "", fmt.Errorf("Azure Policy Definition Id is empty or not formatted correctly: %s", id)
	}

	if len(components) != 7 {
		return "", fmt.Errorf("Azure Policy Definition Id should have 6 segments, got %d: '%s'", len(components)-1, id)
	}

	return components[6], nil
}
