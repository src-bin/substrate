resource "aws_acm_certificate" "intranet" {
  domain_name       = var.dns_domain_name
  validation_method = "DNS"
}

resource "aws_acm_certificate_validation" "intranet" {
  certificate_arn = aws_acm_certificate.intranet.arn
  #validation_record_fqdns = [aws_route53_record.validation.fqdn]
  validation_record_fqdns = [var.validation_fqdn]
}
