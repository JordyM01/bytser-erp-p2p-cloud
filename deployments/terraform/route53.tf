data "aws_route53_zone" "main" {
  name = var.domain_name
}

resource "aws_route53_record" "relay" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = "${var.subdomain}.${var.domain_name}"
  type    = "A"
  ttl     = 300
  records = [aws_eip.relay.public_ip]
}
