variable "aws_region" {
  description = "Region holding the web release bucket"
  type        = string
  default     = "eu-west-2"
}

variable "web_bucket_name" {
  description = "Globally unique name of the private bucket holding the built web release and its runtime config artifact"
  type        = string
}

variable "cloudfront_price_class" {
  description = "CloudFront price class for the web distribution"
  type        = string
  default     = "PriceClass_100"
}
