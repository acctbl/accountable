output "web_bucket_name" {
  description = "Upload target for the built web release and runtime config artifact"
  value       = aws_s3_bucket.web.bucket
}

output "web_distribution_id" {
  description = "CloudFront distribution to invalidate after a release upload"
  value       = aws_cloudfront_distribution.web.id
}

output "web_distribution_domain" {
  description = "Public domain serving the web release"
  value       = aws_cloudfront_distribution.web.domain_name
}
