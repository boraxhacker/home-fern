

resource "aws_ssm_parameter" "param1" {

  name = "/user/fred"
  value = "fred"
  type = "String"
}

resource "aws_route53_zone" "zone1" {
  name = "fred.net"
}

resource "aws_route53_record" "www" {
  zone_id = aws_route53_zone.zone1.zone_id
  name    = "www.fred.net"
  type    = "A"
  records = [
    "192.168.1.2"
  ]
}