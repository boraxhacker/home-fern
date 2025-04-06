

resource "aws_ssm_parameter" "param1" {

  name = "/user/fred"
  value = "fred"
  type = "String"
}

resource "aws_route53_zone" "zone1" {
  name = "fred.net"
}