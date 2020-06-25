data "aws_route53_zone" "intranet" {
  name         = "${var.dns_domain_name}."
  private_zone = false
}

resource "aws_route53_record" "intranet" {
  alias {
    evaluate_target_health = true
    name                   = aws_api_gateway_domain_name.intranet.regional_domain_name
    zone_id                = aws_api_gateway_domain_name.intranet.regional_zone_id
  }
  latency_routing_policy {
    region = data.aws_region.current.name
  }
  name           = aws_api_gateway_domain_name.intranet.domain_name
  set_identifier = data.aws_region.current.name
  type           = "A"
  zone_id        = data.aws_route53_zone.intranet.id
}
