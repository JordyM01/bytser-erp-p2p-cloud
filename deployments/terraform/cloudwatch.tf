resource "aws_cloudwatch_log_group" "relay" {
  name              = "/bytsers/p2p-cloud/${var.environment}"
  retention_in_days = 30
}
