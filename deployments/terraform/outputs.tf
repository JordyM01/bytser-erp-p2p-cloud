output "instance_id" {
  description = "EC2 instance ID"
  value       = aws_instance.relay.id
}

output "elastic_ip" {
  description = "Elastic IP address of the relay server"
  value       = aws_eip.relay.public_ip
}

output "relay_domain" {
  description = "FQDN of the relay server"
  value       = aws_route53_record.relay.fqdn
}

output "ecr_repository_url" {
  description = "ECR repository URL"
  value       = aws_ecr_repository.relay.repository_url
}

output "ssh_command" {
  description = "SSH command to connect to the instance"
  value       = "ssh -i ${local_file.ssh_key.filename} ubuntu@${aws_eip.relay.public_ip}"
}

output "ssh_private_key" {
  description = "SSH private key (save to file and chmod 600)"
  value       = tls_private_key.relay.private_key_pem
  sensitive   = true
}
