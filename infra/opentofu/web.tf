locals {
  web_origin_id = "web-release"

  web_cache_control = {
    runtime_config = "no-store"
    hashed_asset   = "public, max-age=31536000, immutable"
    document       = "no-cache"
  }
}

resource "aws_s3_bucket" "web" {
  bucket = var.web_bucket_name
}

resource "aws_s3_bucket_public_access_block" "web" {
  bucket = aws_s3_bucket.web.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "web" {
  bucket = aws_s3_bucket.web.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_kms_key" "web" {
  description         = "Encrypts the web release bucket"
  enable_key_rotation = true
}

resource "aws_kms_key_policy" "web" {
  key_id = aws_kms_key.web.id
  policy = data.aws_iam_policy_document.web_key.json
}

data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "web_key" {
  statement {
    sid       = "AccountRoot"
    actions   = ["kms:*"]
    resources = ["*"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"]
    }
  }

  statement {
    sid       = "CloudFrontRead"
    actions   = ["kms:Decrypt"]
    resources = ["*"]

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.web.arn]
    }
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "web" {
  bucket = aws_s3_bucket.web.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.web.arn
    }

    bucket_key_enabled = true
  }
}

data "aws_iam_policy_document" "web_bucket" {
  statement {
    sid       = "CloudFrontRead"
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.web.arn}/*"]

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.web.arn]
    }
  }

  statement {
    sid       = "RequireTLS"
    effect    = "Deny"
    actions   = ["s3:*"]
    resources = [aws_s3_bucket.web.arn, "${aws_s3_bucket.web.arn}/*"]

    principals {
      type        = "AWS"
      identifiers = ["*"]
    }

    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
  }
}

resource "aws_s3_bucket_policy" "web" {
  bucket = aws_s3_bucket.web.id
  policy = data.aws_iam_policy_document.web_bucket.json

  depends_on = [aws_s3_bucket_public_access_block.web]
}

resource "aws_cloudfront_function" "spa_router" {
  name    = "accountable-web-spa-router"
  runtime = "cloudfront-js-2.0"
  comment = "Rewrites extensionless document requests to the SPA shell"
  publish = true
  code    = file("${path.module}/spa-router.js")
}

resource "aws_cloudfront_origin_access_control" "web" {
  name                              = "accountable-web"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

data "aws_cloudfront_cache_policy" "caching_optimized" {
  name = "Managed-CachingOptimized"
}

data "aws_cloudfront_cache_policy" "caching_disabled" {
  name = "Managed-CachingDisabled"
}

resource "aws_cloudfront_response_headers_policy" "runtime_config" {
  name = "accountable-web-runtime-config"

  custom_headers_config {
    items {
      header   = "Cache-Control"
      override = true
      value    = local.web_cache_control.runtime_config
    }
  }
}

resource "aws_cloudfront_response_headers_policy" "hashed_assets" {
  name = "accountable-web-hashed-assets"

  custom_headers_config {
    items {
      header   = "Cache-Control"
      override = true
      value    = local.web_cache_control.hashed_asset
    }
  }
}

resource "aws_cloudfront_response_headers_policy" "documents" {
  name = "accountable-web-documents"

  custom_headers_config {
    items {
      header   = "Cache-Control"
      override = true
      value    = local.web_cache_control.document
    }
  }
}

resource "aws_cloudfront_distribution" "web" {
  enabled             = true
  is_ipv6_enabled     = true
  comment             = "accountable web release"
  default_root_object = "index.html"
  price_class         = var.cloudfront_price_class

  origin {
    domain_name              = aws_s3_bucket.web.bucket_regional_domain_name
    origin_id                = local.web_origin_id
    origin_access_control_id = aws_cloudfront_origin_access_control.web.id
  }

  default_cache_behavior {
    allowed_methods            = ["GET", "HEAD"]
    cached_methods             = ["GET", "HEAD"]
    target_origin_id           = local.web_origin_id
    viewer_protocol_policy     = "redirect-to-https"
    compress                   = true
    cache_policy_id            = data.aws_cloudfront_cache_policy.caching_disabled.id
    response_headers_policy_id = aws_cloudfront_response_headers_policy.documents.id

    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.spa_router.arn
    }
  }

  ordered_cache_behavior {
    path_pattern               = "/assets/*"
    allowed_methods            = ["GET", "HEAD"]
    cached_methods             = ["GET", "HEAD"]
    target_origin_id           = local.web_origin_id
    viewer_protocol_policy     = "redirect-to-https"
    compress                   = true
    cache_policy_id            = data.aws_cloudfront_cache_policy.caching_optimized.id
    response_headers_policy_id = aws_cloudfront_response_headers_policy.hashed_assets.id
  }

  ordered_cache_behavior {
    path_pattern               = "/_runtime/config.json"
    allowed_methods            = ["GET", "HEAD"]
    cached_methods             = ["GET", "HEAD"]
    target_origin_id           = local.web_origin_id
    viewer_protocol_policy     = "redirect-to-https"
    compress                   = true
    cache_policy_id            = data.aws_cloudfront_cache_policy.caching_disabled.id
    response_headers_policy_id = aws_cloudfront_response_headers_policy.runtime_config.id
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}
