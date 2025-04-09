

resource "aws_ssm_parameter" "param1" {

  name = "/user/fred/username"
  value = "fred"
  type = "String"
}

resource "aws_ssm_parameter" "param2" {

  name = "/user/fred/password"
  value = "fred"
  type = "SecureString"
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