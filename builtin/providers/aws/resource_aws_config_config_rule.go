package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/configservice"
)

func resourceAwsConfigConfigRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigConfigRulePut,
		Read:   resourceAwsConfigConfigRuleRead,
		Update: resourceAwsConfigConfigRulePut,
		Delete: resourceAwsConfigConfigRuleDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateMaxLength(64),
			},
			"rule_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateMaxLength(256),
			},
			"input_parameters": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateJsonString,
			},
			"maximum_execution_frequency": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateConfigExecutionFrequency,
			},
			"scope": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"compliance_resource_id": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateMaxLength(256),
						},
						"compliance_resource_types": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 100,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"tag_key": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateMaxLength(128),
						},
						"tag_value": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateMaxLength(256),
						},
					},
				},
			},
			"source": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"owner": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateConfigRuleSourceOwner,
						},
						"source_detail": &schema.Schema{
							Type:     schema.TypeSet,
							Set:      configRuleSourceDetailsHash,
							Optional: true,
							MaxItems: 25,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"event_source": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"maximum_execution_frequency": &schema.Schema{
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateConfigExecutionFrequency,
									},
									"message_type": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"source_identifier": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateMaxLength(256),
						},
					},
				},
			},
		},
	}
}

func resourceAwsConfigConfigRulePut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	name := d.Get("name").(string)

	sources := d.Get("source").([]interface{})
	source := sources[0].(map[string]interface{})
	sourceInput := configservice.Source{
		Owner:            aws.String(source["owner"].(string)),
		SourceIdentifier: aws.String(source["source_identifier"].(string)),
	}
	if details, ok := source["source_detail"]; ok {
		sourceInput.SourceDetails = expandConfigRuleSourceDetails(details.(*schema.Set))
	}

	ruleInput := configservice.ConfigRule{
		ConfigRuleName: aws.String(name),
		Source:         &sourceInput,
	}

	scopes := d.Get("scope").([]interface{})
	if len(scopes) > 0 {
		ruleInput.Scope = expandConfigRuleScope(scopes[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("description"); ok {
		ruleInput.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("input_parameters"); ok {
		ruleInput.InputParameters = aws.String(v.(string))
	}
	if v, ok := d.GetOk("maximum_execution_frequency"); ok {
		ruleInput.MaximumExecutionFrequency = aws.String(v.(string))
	}

	input := configservice.PutConfigRuleInput{
		ConfigRule: &ruleInput,
	}
	log.Printf("[DEBUG] Creating AWSConfig config rule: %s", input)
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		_, err := conn.PutConfigRule(&input)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "InsufficientPermissionsException" {
					// IAM is eventually consistent
					return resource.RetryableError(err)
				}
			}

			return resource.NonRetryableError(fmt.Errorf("Failed to create AWSConfig rule: %s", err))
		}

		return nil
	})
	if err != nil {
		return err
	}

	d.SetId(name)

	log.Printf("[DEBUG] AWSConfig config rule %q created", name)

	return resourceAwsConfigConfigRuleRead(d, meta)
}

func resourceAwsConfigConfigRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	out, err := conn.DescribeConfigRules(&configservice.DescribeConfigRulesInput{
		ConfigRuleNames: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchConfigRuleException" {
			log.Printf("[WARN] Config Rule %q is gone", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if len(out.ConfigRules) < 1 {
		log.Printf("[WARN] Config Rule %q is gone", d.Id())
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] AWS Config config rule received: %s", out)

	rule := out.ConfigRules[0]
	d.Set("arn", rule.ConfigRuleArn)
	d.Set("rule_id", rule.ConfigRuleId)
	d.Set("name", rule.ConfigRuleName)
	d.Set("description", rule.Description)
	d.Set("input_parameters", rule.InputParameters)
	d.Set("maximum_execution_frequency", rule.MaximumExecutionFrequency)

	if rule.Scope != nil {
		d.Set("scope", flattenConfigRuleScope(rule.Scope))
	}

	var source []interface{}
	m := make(map[string]interface{})
	m["owner"] = *rule.Source.Owner
	m["source_identifier"] = *rule.Source.SourceIdentifier
	if len(rule.Source.SourceDetails) > 0 {
		m["source_detail"] = schema.NewSet(configRuleSourceDetailsHash, flattenConfigRuleSourceDetails(rule.Source.SourceDetails))
	}
	source = append(source, m)
	d.Set("source", source)

	return nil
}

func resourceAwsConfigConfigRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	name := d.Get("name").(string)

	log.Printf("[DEBUG] Deleting AWS Config config rule %q", name)
	_, err := conn.DeleteConfigRule(&configservice.DeleteConfigRuleInput{
		ConfigRuleName: aws.String(name),
	})
	if err != nil {
		return fmt.Errorf("Deleting Config Rule failed: %s", err)
	}

	conf := resource.StateChangeConf{
		Pending: []string{
			"ACTIVE",
			"DELETING",
			"DELETING_RESULTS",
			"EVALUATING",
		},
		Target:  []string{""},
		Timeout: 5 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeConfigRules(&configservice.DescribeConfigRulesInput{
				ConfigRuleNames: []*string{aws.String(d.Id())},
			})
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchConfigRuleException" {
					return 42, "", nil
				}
				return 42, "", fmt.Errorf("Failed to describe config rule %q: %s", d.Id(), err)
			}
			if len(out.ConfigRules) < 1 {
				return 42, "", nil
			}
			rule := out.ConfigRules[0]
			return out, *rule.ConfigRuleState, nil
		},
	}
	_, err = conf.WaitForState()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] AWS Config config rule %q deleted", name)

	d.SetId("")
	return nil
}

func configRuleSourceDetailsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["message_type"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["event_source"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["maximum_execution_frequency"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	return hashcode.String(buf.String())
}
