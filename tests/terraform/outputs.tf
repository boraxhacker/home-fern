
output "HostedZone1" {
  value = aws_route53_zone.zone1.name
}

output "Param1Tier" {
  value = aws_ssm_parameter.param1.tier
}
