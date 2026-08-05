package tofupolicy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEvaluateAcceptsFailClosedCellPlan(t *testing.T) {
	t.Parallel()

	if violations, err := Evaluate(planJSON(t, safeResources()), "453722413624"); err != nil {
		t.Fatalf("Evaluate error = %v", err)
	} else if len(violations) != 0 {
		t.Fatalf("Evaluate violations = %v", violations)
	}
}

func TestEvaluateRejectsUnsafeCellMutations(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		mutate func([]map[string]any)
		want   string
	}{
		"public database": {
			mutate: mutate("aws_db_instance", "publicly_accessible", true),
			want:   "publicly_accessible",
		},
		"single AZ database": {
			mutate: mutate("aws_db_instance", "multi_az", false),
			want:   "Multi-AZ",
		},
		"internet facing load balancer": {
			mutate: mutate("aws_lb", "internal", false),
			want:   "internal",
		},
		"public ECS task": {
			mutate: func(resources []map[string]any) {
				resource(resources, "aws_ecs_service")["change"].(map[string]any)["after"].(map[string]any)["network_configuration"].([]any)[0].(map[string]any)["assign_public_ip"] = true
			},
			want: "public IP",
		},
		"world open ingress": {
			mutate: mutate("aws_vpc_security_group_ingress_rule", "cidr_ipv4", "0.0.0.0/0"),
			want:   "world-open",
		},
		"weak bucket block": {
			mutate: mutate("aws_s3_bucket_public_access_block", "restrict_public_buckets", false),
			want:   "public-access",
		},
		"mutable image": {
			mutate: mutate("aws_ecs_task_definition", "container_definitions", `[{"image":"repository:main"}]`),
			want:   "digest",
		},
		"malformed image digest": {
			mutate: mutate("aws_ecs_task_definition", "container_definitions", `[{"image":"repository@sha256:zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"}]`),
			want:   "digest",
		},
		"missing WAF": {
			mutate: mutate("aws_cloudfront_distribution", "web_acl_id", ""),
			want:   "WAF",
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resources := safeResources()
			test.mutate(resources)
			violations, err := Evaluate(planJSON(t, resources), "453722413624")
			if err != nil {
				t.Fatalf("Evaluate error = %v", err)
			}
			if !containsViolation(violations, test.want) {
				t.Fatalf("Evaluate violations = %v, want text %q", violations, test.want)
			}
		})
	}
}

func TestEvaluateRejectsWrongOrUnprovableAccount(t *testing.T) {
	t.Parallel()

	for name, providerConfig := range map[string]any{
		"wrong":   []any{"000000000000"},
		"missing": nil,
	} {
		providerConfig := providerConfig
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plan := planFixture(safeResources())
			plan["configuration"].(map[string]any)["provider_config"].(map[string]any)["aws"].(map[string]any)["expressions"].(map[string]any)["allowed_account_ids"].(map[string]any)["constant_value"] = providerConfig
			payload, err := json.Marshal(plan)
			if err != nil {
				t.Fatal(err)
			}
			violations, err := Evaluate(payload, "453722413624")
			if err != nil {
				t.Fatalf("Evaluate error = %v", err)
			}
			if !containsViolation(violations, "account") {
				t.Fatalf("Evaluate violations = %v, want account rejection", violations)
			}
		})
	}
}

func TestEvaluateRejectsWrongGlobalProviderAccount(t *testing.T) {
	t.Parallel()

	plan := planFixture(safeResources())
	providers := plan["configuration"].(map[string]any)["provider_config"].(map[string]any)
	providers["aws.global"] = map[string]any{
		"expressions": map[string]any{
			"allowed_account_ids": map[string]any{"constant_value": []any{"000000000000"}},
		},
	}
	payload, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	violations, err := Evaluate(payload, "453722413624")
	if err != nil {
		t.Fatalf("Evaluate error = %v", err)
	}
	if !containsViolation(violations, "account") {
		t.Fatalf("Evaluate violations = %v, want global account rejection", violations)
	}
}

func TestEvaluateRejectsUnknownPublicAccessValues(t *testing.T) {
	t.Parallel()

	for name, resourceType := range map[string]string{
		"database exposure": "aws_db_instance",
		"ECS public IP":     "aws_ecs_service",
		"ingress source":    "aws_vpc_security_group_ingress_rule",
	} {
		resourceType := resourceType
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resources := safeResources()
			change := resource(resources, resourceType)["change"].(map[string]any)
			after := change["after"].(map[string]any)
			afterUnknown := map[string]any{}
			switch resourceType {
			case "aws_db_instance":
				after["publicly_accessible"] = nil
				afterUnknown["publicly_accessible"] = true
			case "aws_ecs_service":
				after["network_configuration"] = []any{map[string]any{"assign_public_ip": nil}}
				afterUnknown["network_configuration"] = []any{map[string]any{"assign_public_ip": true}}
			case "aws_vpc_security_group_ingress_rule":
				afterUnknown["cidr_ipv4"] = true
			}
			change["after_unknown"] = afterUnknown
			violations, err := Evaluate(planJSON(t, resources), "453722413624")
			if err != nil {
				t.Fatalf("Evaluate error = %v", err)
			}
			if len(violations) == 0 {
				t.Fatal("Evaluate accepted an unknown public-access value")
			}
		})
	}
}

func TestEvaluateAcceptsUnknownTaskDefinitionsWhenConfigurationPinsTheImageInput(t *testing.T) {
	t.Parallel()

	plan := planFixture(safeResources())
	task := planResource(plan, "aws_ecs_task_definition")
	task["address"] = "module.cell.aws_ecs_task_definition.api"
	taskChange := task["change"].(map[string]any)
	taskChange["after"].(map[string]any)["container_definitions"] = nil
	taskChange["after_unknown"] = map[string]any{"container_definitions": true}
	plan["variables"] = map[string]any{
		"image_uri": map[string]any{
			"value": "453722413624.dkr.ecr.eu-west-2.amazonaws.com/accountable@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
	plan["configuration"].(map[string]any)["root_module"] = map[string]any{
		"module_calls": map[string]any{
			"cell": map[string]any{
				"expressions": map[string]any{
					"image_uri": map[string]any{"references": []any{"var.image_uri"}},
				},
				"module": map[string]any{
					"resources": []any{
						map[string]any{
							"address": "aws_ecs_task_definition.api",
							"type":    "aws_ecs_task_definition",
							"expressions": map[string]any{
								"container_definitions": map[string]any{"references": []any{"var.image_uri"}},
							},
						},
					},
				},
			},
		},
	}

	payload, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	violations, err := Evaluate(payload, "453722413624")
	if err != nil {
		t.Fatalf("Evaluate error = %v", err)
	}
	if containsViolation(violations, "digest") {
		t.Fatalf("Evaluate violations = %v", violations)
	}
}

func TestEvaluateRejectsAnUnknownTaskDefinitionWithoutAProvenDigestInput(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		imageURI  string
		reference string
	}{
		"mutable input": {
			imageURI:  "453722413624.dkr.ecr.eu-west-2.amazonaws.com/accountable:main",
			reference: "var.image_uri",
		},
		"missing input": {
			reference: "var.image_uri",
		},
		"disconnected input": {
			imageURI:  "453722413624.dkr.ecr.eu-west-2.amazonaws.com/accountable@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			reference: "local.mutable_image",
		},
	} {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plan := planFixture(safeResources())
			task := planResource(plan, "aws_ecs_task_definition")
			taskChange := task["change"].(map[string]any)
			taskChange["after"].(map[string]any)["container_definitions"] = nil
			taskChange["after_unknown"] = map[string]any{"container_definitions": true}
			plan["variables"] = map[string]any{"image_uri": map[string]any{"value": test.imageURI}}
			plan["configuration"].(map[string]any)["root_module"] = map[string]any{
				"resources": []any{
					map[string]any{
						"address": "aws_ecs_task_definition.api",
						"type":    "aws_ecs_task_definition",
						"expressions": map[string]any{
							"container_definitions": map[string]any{"references": []any{test.reference}},
						},
					},
				},
			}
			payload, err := json.Marshal(plan)
			if err != nil {
				t.Fatal(err)
			}
			violations, err := Evaluate(payload, "453722413624")
			if err != nil {
				t.Fatalf("Evaluate error = %v", err)
			}
			if !containsViolation(violations, "digest") {
				t.Fatalf("Evaluate violations = %v, want digest rejection", violations)
			}
		})
	}
}

func TestEvaluateCorrelatesAnUnknownS3BlockBucketFromConfiguration(t *testing.T) {
	t.Parallel()

	for name, reference := range map[string]string{
		"matching bucket":     "aws_s3_bucket.data.id",
		"disconnected bucket": "aws_s3_bucket.other.id",
	} {
		reference := reference
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plan := planFixture(safeResources())
			block := planResource(plan, "aws_s3_bucket_public_access_block")
			blockChange := block["change"].(map[string]any)
			blockChange["after"].(map[string]any)["bucket"] = nil
			blockChange["after_unknown"] = map[string]any{"bucket": true}
			plan["configuration"].(map[string]any)["root_module"] = map[string]any{
				"resources": []any{
					map[string]any{
						"address": "aws_s3_bucket_public_access_block.data",
						"type":    "aws_s3_bucket_public_access_block",
						"expressions": map[string]any{
							"bucket": map[string]any{"references": []any{reference}},
						},
					},
				},
			}
			payload, err := json.Marshal(plan)
			if err != nil {
				t.Fatal(err)
			}
			violations, err := Evaluate(payload, "453722413624")
			if err != nil {
				t.Fatalf("Evaluate error = %v", err)
			}
			rejected := containsViolation(violations, "no complete public-access block")
			if name == "matching bucket" && rejected {
				t.Fatalf("Evaluate violations = %v", violations)
			}
			if name == "disconnected bucket" && !rejected {
				t.Fatalf("Evaluate violations = %v, want disconnected block rejection", violations)
			}
		})
	}
}

func TestEvaluateRejectsMalformedExpectedAccount(t *testing.T) {
	t.Parallel()

	if _, err := Evaluate(planJSON(t, safeResources()), "aaaaaaaaaaaa"); err == nil {
		t.Fatal("Evaluate accepted a malformed expected account")
	}
}

func TestEvaluateAcceptsAPlannedWAFReferenceThatIsUnknownUntilApply(t *testing.T) {
	t.Parallel()

	plan := planFixture(safeResources())
	plan["planned_values"] = map[string]any{
		"root_module": map[string]any{
			"resources": []any{
				map[string]any{
					"address": "aws_wafv2_web_acl.edge",
					"type":    "aws_wafv2_web_acl",
					"values":  map[string]any{},
				},
				map[string]any{
					"address": "aws_cloudfront_distribution.cell",
					"type":    "aws_cloudfront_distribution",
					"values":  map[string]any{"web_acl_id": nil},
				},
			},
		},
	}
	for _, change := range plan["resource_changes"].([]any) {
		if change.(map[string]any)["type"] == "aws_cloudfront_distribution" {
			change.(map[string]any)["change"].(map[string]any)["after_unknown"] = map[string]any{"web_acl_id": true}
		}
	}
	payload, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	violations, err := Evaluate(payload, "453722413624")
	if err != nil {
		t.Fatalf("Evaluate error = %v", err)
	}
	if containsViolation(violations, "WAF") {
		t.Fatalf("Evaluate violations = %v", violations)
	}
}

func TestEvaluateRejectsUnknownWAFReferenceWithoutAPlannedACL(t *testing.T) {
	t.Parallel()

	plan := planFixture(safeResources())
	plan["planned_values"] = map[string]any{
		"root_module": map[string]any{
			"resources": []any{
				map[string]any{
					"address": "aws_cloudfront_distribution.cell",
					"type":    "aws_cloudfront_distribution",
					"values":  map[string]any{"web_acl_id": nil},
				},
			},
		},
	}
	for _, change := range plan["resource_changes"].([]any) {
		if change.(map[string]any)["type"] == "aws_cloudfront_distribution" {
			change.(map[string]any)["change"].(map[string]any)["after_unknown"] = map[string]any{"web_acl_id": true}
		}
	}
	payload, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	violations, err := Evaluate(payload, "453722413624")
	if err != nil {
		t.Fatalf("Evaluate error = %v", err)
	}
	if !containsViolation(violations, "WAF") {
		t.Fatalf("Evaluate violations = %v, want WAF rejection", violations)
	}
}

func safeResources() []map[string]any {
	return []map[string]any{
		resourceChangeFixture("aws_db_instance.cell", "aws_db_instance", map[string]any{"multi_az": true, "publicly_accessible": false, "storage_encrypted": true}),
		resourceChangeFixture("aws_lb.api", "aws_lb", map[string]any{"internal": true}),
		resourceChangeFixture("aws_ecs_service.api", "aws_ecs_service", map[string]any{"network_configuration": []any{map[string]any{"assign_public_ip": false}}}),
		resourceChangeFixture("aws_vpc_security_group_ingress_rule.cloudfront", "aws_vpc_security_group_ingress_rule", map[string]any{"cidr_ipv4": nil}),
		resourceChangeFixture("aws_s3_bucket.data", "aws_s3_bucket", map[string]any{"bucket": "accountable-data"}),
		resourceChangeFixture("aws_s3_bucket_public_access_block.data", "aws_s3_bucket_public_access_block", map[string]any{
			"bucket": "accountable-data", "block_public_acls": true, "block_public_policy": true,
			"ignore_public_acls": true, "restrict_public_buckets": true,
		}),
		resourceChangeFixture("aws_ecs_task_definition.api", "aws_ecs_task_definition", map[string]any{"container_definitions": `[{"image":"repository@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]`}),
		resourceChangeFixture("aws_wafv2_web_acl.edge", "aws_wafv2_web_acl", map[string]any{}),
		resourceChangeFixture("aws_cloudfront_distribution.cell", "aws_cloudfront_distribution", map[string]any{"web_acl_id": "arn:aws:wafv2:us-east-1:453722413624:global/webacl/cell/id"}),
	}
}

func resourceChangeFixture(address, resourceType string, after map[string]any) map[string]any {
	return map[string]any{
		"address": address,
		"type":    resourceType,
		"change": map[string]any{
			"actions": []any{"create"},
			"after":   after,
		},
	}
}

func planJSON(t *testing.T, resources []map[string]any) []byte {
	t.Helper()
	payload, err := json.Marshal(planFixture(resources))
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func planFixture(resources []map[string]any) map[string]any {
	changes := make([]any, len(resources))
	for index, resource := range resources {
		changes[index] = resource
	}
	return map[string]any{
		"format_version":   "1.2",
		"resource_changes": changes,
		"configuration": map[string]any{
			"provider_config": map[string]any{
				"aws": map[string]any{
					"expressions": map[string]any{
						"allowed_account_ids": map[string]any{"constant_value": []any{"453722413624"}},
					},
				},
			},
		},
	}
}

func resource(resources []map[string]any, resourceType string) map[string]any {
	for _, candidate := range resources {
		if candidate["type"] == resourceType {
			return candidate
		}
	}
	panic("missing test resource " + resourceType)
}

func planResource(plan map[string]any, resourceType string) map[string]any {
	for _, candidate := range plan["resource_changes"].([]any) {
		resource := candidate.(map[string]any)
		if resource["type"] == resourceType {
			return resource
		}
	}
	panic("missing test resource " + resourceType)
}

func mutate(resourceType, field string, value any) func([]map[string]any) {
	return func(resources []map[string]any) {
		resource(resources, resourceType)["change"].(map[string]any)["after"].(map[string]any)[field] = value
	}
}

func containsViolation(violations []Violation, text string) bool {
	for _, violation := range violations {
		if strings.Contains(violation.Message, text) {
			return true
		}
	}
	return false
}
