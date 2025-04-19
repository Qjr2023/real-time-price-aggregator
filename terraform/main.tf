provider "aws" {
  region     = var.region
  access_key = var.access_key
  secret_key = var.secret_key
  token      = var.session_token
}

data "aws_ami" "amazon_linux_2023" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-2023*-x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }
}

# security group, allow all traffic
resource "aws_security_group" "price_aggregator_sg" {
  name        = "price-aggregator-sg"
  description = "Allow all traffic for price aggregator services"

  # allow all inbound traffic
  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # allow all outbound traffic
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "price-aggregator-sg"
  }
}

# redis instance
resource "aws_instance" "redis" {
  ami                    = data.aws_ami.amazon_linux_2023.id
  instance_type          = "t3.micro"
  key_name               = var.key_name
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              # update system
              dnf update -y
              # install docker
              dnf install -y docker
              systemctl enable docker
              systemctl start docker
              usermod -aG docker ec2-user
              
              # run redis docker container
              docker run -d -p 6379:6379 --name redis redis:latest
              EOF

  tags = {
    Name = "price-aggregator-redis"
  }
}

# Exchange1 instance
resource "aws_instance" "exchange1" {
  ami                    = data.aws_ami.amazon_linux_2023.id
  instance_type          = "t3.micro"
  key_name               = var.key_name
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              dnf update -y
              dnf install -y docker
              systemctl enable docker
              systemctl start docker
              usermod -aG docker ec2-user
              
              # clone repository
              dnf install -y git
              git clone https://github.com/Qjr2023/real-time-price-aggregator.git
              cd real-time-price-aggregator
              
              # build and run Exchange1
              docker build -t exchange1 -f mocks/Dockerfile .
              docker run -d -p 8081:8081 exchange1 ./mock_server 8081 exchange1
              EOF

  tags = {
    Name = "price-aggregator-exchange1"
  }
}

# Exchange2 instance
resource "aws_instance" "exchange2" {
  ami                    = data.aws_ami.amazon_linux_2023.id
  instance_type          = "t3.micro"
  key_name               = var.key_name
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              dnf update -y
              dnf install -y docker
              systemctl enable docker
              systemctl start docker
              usermod -aG docker ec2-user
              
              dnf install -y git
              git clone https://github.com/Qjr2023/real-time-price-aggregator.git
              cd real-time-price-aggregator

              docker build -t exchange2 -f mocks/Dockerfile .
              docker run -d -p 8082:8082 exchange2 ./mock_server 8082 exchange2
              EOF

  tags = {
    Name = "price-aggregator-exchange2"
  }
}

# Exchange3 instance
resource "aws_instance" "exchange3" {
  ami                    = data.aws_ami.amazon_linux_2023.id
  instance_type          = "t3.micro"
  key_name               = var.key_name
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              dnf update -y
              dnf install -y docker
              systemctl enable docker
              systemctl start docker
              usermod -aG docker ec2-user
              
              dnf install -y git
              git clone https://github.com/Qjr2023/real-time-price-aggregator.git
              cd real-time-price-aggregator

              docker build -t exchange3 -f mocks/Dockerfile .
              docker run -d -p 8083:8083 exchange3 ./mock_server 8083 exchange3
              EOF

  tags = {
    Name = "price-aggregator-exchange3"
  }
}

# API server instance
resource "aws_instance" "api_server" {
  ami                    = data.aws_ami.amazon_linux_2023.id
  instance_type          = "t3.small"
  key_name               = var.key_name
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              dnf update -y
              dnf install -y docker
              systemctl enable docker
              systemctl start docker
              usermod -aG docker ec2-user
              
              dnf install -y git
              git clone https://github.com/Qjr2023/real-time-price-aggregator.git
              cd real-time-price-aggregator
              
              # revise main.go to use environment variables
              sed -i 's/Addr: "redis:6379"/Addr: os.Getenv("REDIS_ADDR")/g' cmd/main.go
              
              # create AWS credentials file
              mkdir -p /home/ec2-user/.aws
              cat > /home/ec2-user/.aws/credentials <<CREDENTIALS
              [default]
              aws_access_key_id=${var.access_key}
              aws_secret_access_key=${var.secret_key}
              aws_session_token=${var.session_token}
              CREDENTIALS
              
              cat > /home/ec2-user/.aws/config <<CONFIG
              [default]
              region=${var.region}
              CONFIG
              
              # setup permissions
              chmod 600 /home/ec2-user/.aws/credentials
              chmod 600 /home/ec2-user/.aws/config

              # setup environment variables
              export REDIS_ADDR="${aws_instance.redis.private_ip}:6379"
              
              # build and run API server
              cat > start_server.sh <<SCRIPT
              #!/bin/bash
              cd /home/ec2-user/real-time-price-aggregator
              
              # setup environment variables
              export REDIS_ADDR="${aws_instance.redis.private_ip}:6379"
              export AWS_REGION="${var.region}"

              # build and run API server
              docker build -t api-server -f Dockerfile .
              docker run -d -p 8080:8080 \
                -v /home/ec2-user/.aws:/root/.aws:ro \\
                -e REDIS_ADDR="\$REDIS_ADDR" \\
                -e AWS_REGION="${var.region}" \\
                api-server
              SCRIPT
              
              chmod +x start_server.sh
              ./start_server.sh
              EOF

  depends_on = [
    aws_instance.redis,
  ]

  tags = {
    Name = "price-aggregator-api-server"
  }
}

# Lambda function configuration

# Terraform configuration with provider requirements
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}

# Lambda function deployment package
data "archive_file" "lambda_refresh_price" {
  type        = "zip"
  source_file = "${path.module}/lambda/main"
  output_path = "${path.module}/lambda.zip"
}

# Lambda function
resource "aws_lambda_function" "refresh_price" {
  function_name = "price-aggregator-refresh-price"
  filename      = data.archive_file.lambda_refresh_price.output_path
  handler       = "main"
  runtime       = "go1.x"
  timeout       = 30
  memory_size   = 256

  # Environment variables, including AWS credentials
  environment {
    variables = {
      REDIS_ADDR           = "${aws_instance.redis.private_ip}:6379"
      EXCHANGE1_URL        = "http://${aws_instance.exchange1.private_ip}:8081/mock/ticker"
      EXCHANGE2_URL        = "http://${aws_instance.exchange2.private_ip}:8082/mock/ticker"
      EXCHANGE3_URL        = "http://${aws_instance.exchange3.private_ip}:8083/mock/ticker"
      AWS_REGION           = var.region
      AWS_ACCESS_KEY_ID    = var.access_key
      AWS_SECRET_ACCESS_KEY = var.secret_key
      AWS_SESSION_TOKEN    = var.session_token
    }
  }

  depends_on = [
    aws_instance.redis,
    aws_instance.exchange1,
    aws_instance.exchange2,
    aws_instance.exchange3
  ]
}

# API Gateway REST API
resource "aws_api_gateway_rest_api" "price_api" {
  name        = "price-aggregator-api"
  description = "Price Aggregator API"
}

# API Gateway resource - /refresh
resource "aws_api_gateway_resource" "refresh" {
  rest_api_id = aws_api_gateway_rest_api.price_api.id
  parent_id   = aws_api_gateway_rest_api.price_api.root_resource_id
  path_part   = "refresh"
}

# API Gateway resource - /refresh/{asset}
resource "aws_api_gateway_resource" "asset" {
  rest_api_id = aws_api_gateway_rest_api.price_api.id
  parent_id   = aws_api_gateway_resource.refresh.id
  path_part   = "{asset}"
}

# API Gateway method - POST /refresh/{asset}
resource "aws_api_gateway_method" "post_refresh" {
  rest_api_id   = aws_api_gateway_rest_api.price_api.id
  resource_id   = aws_api_gateway_resource.asset.id
  http_method   = "POST"
  authorization_type = "NONE"
}

# API Gateway integration with Lambda
resource "aws_api_gateway_integration" "lambda_integration" {
  rest_api_id = aws_api_gateway_rest_api.price_api.id
  resource_id = aws_api_gateway_resource.asset.id
  http_method = aws_api_gateway_method.post_refresh.http_method

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.refresh_price.invoke_arn
}

# API Gateway deployment
resource "aws_api_gateway_deployment" "api_deployment" {
  depends_on = [
    aws_api_gateway_integration.lambda_integration
  ]

  rest_api_id = aws_api_gateway_rest_api.price_api.id
  stage_name  = "prod"
}

# Lambda function permission for API Gateway
resource "aws_lambda_permission" "api_gateway_permission" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.refresh_price.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.price_api.execution_arn}/*/*"
}

# CloudWatch event rule - high priority assets (every 5 seconds)
resource "aws_cloudwatch_event_rule" "high_tier_refresh" {
  name                = "price-aggregator-high-tier-refresh"
  description         = "Refresh high-tier assets every 5 seconds"
  schedule_expression = "rate(5 seconds)"
}

resource "aws_cloudwatch_event_target" "high_tier_target" {
  rule      = aws_cloudwatch_event_rule.high_tier_refresh.name
  target_id = "lambda"
  arn       = aws_lambda_function.refresh_price.arn
  input     = jsonencode({
    tier = "high"
  })
}

# CloudWatch event rule - medium priority assets (every 30 seconds)
resource "aws_cloudwatch_event_rule" "medium_tier_refresh" {
  name                = "price-aggregator-medium-tier-refresh"
  description         = "Refresh medium-tier assets every 30 seconds"
  schedule_expression = "rate(30 seconds)"
}

resource "aws_cloudwatch_event_target" "medium_tier_target" {
  rule      = aws_cloudwatch_event_rule.medium_tier_refresh.name
  target_id = "lambda"
  arn       = aws_lambda_function.refresh_price.arn
  input     = jsonencode({
    tier = "medium"
  })
}

# CloudWatch event rule - low priority assets (every 5 minutes)
resource "aws_cloudwatch_event_rule" "low_tier_refresh" {
  name                = "price-aggregator-low-tier-refresh"
  description         = "Refresh low-tier assets every 5 minutes"
  schedule_expression = "rate(5 minutes)"
}

resource "aws_cloudwatch_event_target" "low_tier_target" {
  rule      = aws_cloudwatch_event_rule.low_tier_refresh.name
  target_id = "lambda"
  arn       = aws_lambda_function.refresh_price.arn
  input     = jsonencode({
    tier = "low"
  })
}

# Lambda function permission for CloudWatch Events - high priority
resource "aws_lambda_permission" "high_tier_permission" {
  statement_id  = "AllowExecutionFromCloudWatchHighTier"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.refresh_price.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.high_tier_refresh.arn
}

# Lambda function permission for CloudWatch Events - medium priority
resource "aws_lambda_permission" "medium_tier_permission" {
  statement_id  = "AllowExecutionFromCloudWatchMediumTier"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.refresh_price.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.medium_tier_refresh.arn
}

# Lambda function permission for CloudWatch Events - low priority
resource "aws_lambda_permission" "low_tier_permission" {
  statement_id  = "AllowExecutionFromCloudWatchLowTier"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.refresh_price.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.low_tier_refresh.arn
}

# Output API Gateway URL
output "lambda_api_url" {
  value = "${aws_api_gateway_deployment.api_deployment.invoke_url}/refresh/{asset}"
  description = "URL for manual price refresh via Lambda"
}

# Output the public IPs of the instances
output "redis_ip" {
  value = aws_instance.redis.public_ip
}

output "exchange1_ip" {
  value = aws_instance.exchange1.public_ip
}

output "exchange2_ip" {
  value = aws_instance.exchange2.public_ip
}

output "exchange3_ip" {
  value = aws_instance.exchange3.public_ip
}

output "api_server_ip" {
  value = aws_instance.api_server.public_ip
}

output "connection_command" {
  value = "curl http://${aws_instance.api_server.public_ip}:8080/health"
}

# Lambda 相关输出
output "lambda_function_name" {
  value       = aws_lambda_function.refresh_price.function_name
  description = "Lambda 函数名称"
}

output "lambda_api_url" {
  value       = "${aws_api_gateway_deployment.api_deployment.invoke_url}/refresh/{asset}"
  description = "Lambda API Gateway URL (用于手动刷新价格)"
}

# Lambda 测试命令
output "lambda_test_command" {
  value       = "curl -X POST ${aws_api_gateway_deployment.api_deployment.invoke_url}/refresh/asset1"
  description = "测试 Lambda 函数的 curl 命令 (刷新 asset1)"
}

# 查看 Lambda 日志的命令
output "lambda_logs_command" {
  value       = "aws logs get-log-events --log-group-name /aws/lambda/${aws_lambda_function.refresh_price.function_name} --log-stream-name=`aws logs describe-log-streams --log-group-name /aws/lambda/${aws_lambda_function.refresh_price.function_name} --query 'logStreams[0].logStreamName' --output text` --region ${var.region}"
  description = "查看 Lambda 日志的 AWS CLI 命令"
}

# 验证分层刷新策略的命令
output "verify_tiered_refresh_command" {
  value = "aws logs filter-log-events --log-group-name /aws/lambda/${aws_lambda_function.refresh_price.function_name} --filter-pattern \"Successfully refreshed\" --region ${var.region}"
  description = "验证分层刷新策略的 AWS CLI 命令"
}