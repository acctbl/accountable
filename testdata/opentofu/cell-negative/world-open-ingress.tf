resource "aws_vpc_security_group_ingress_rule" "cloudfront_to_alb" {
  cidr_ipv4      = "0.0.0.0/0"
  prefix_list_id = null
}
