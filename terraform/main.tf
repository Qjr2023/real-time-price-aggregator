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

# Security group for all instances
resource "aws_security_group" "price_aggregator_sg" {
  name        = "price-aggregator-sg"
  description = "Allow traffic for price aggregator services"

  # HTTP access for API
  ingress {
    from_port   = 8080
    to_port     = 8090
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTP access for price API and mock exchanges"
  }

  # Redis port
  ingress {
    from_port   = 6379
    to_port     = 6379
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
    description = "Redis port"
  }

  # Prometheus port
  ingress {
    from_port   = 9090
    to_port     = 9090
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Prometheus port"
  }

  # Grafana port
  ingress {
    from_port   = 3000
    to_port     = 3000
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Grafana port"
  }

  # SSH
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "SSH access"
  }

  # Allow all internal traffic between instances
  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    self        = true
    description = "Allow all traffic between instances"
  }

  # Allow all outbound traffic
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound traffic"
  }

  tags = {
    Name = "price-aggregator-sg"
  }
}

# DynamoDB Table
resource "aws_dynamodb_table" "prices_table" {
  name           = "prices"
  billing_mode   = "PAY_PER_REQUEST"  # On-demand capacity
  hash_key       = "asset"
  range_key      = "timestamp"
  table_class    = "STANDARD"

  attribute {
    name = "asset"
    type = "S"
  }

  attribute {
    name = "timestamp"
    type = "N"
  }

  ttl {
    attribute_name = "UpdatedAt"
    enabled        = true
  }

  tags = {
    Name        = "prices-table"
    Environment = "production"
  }
}

# Redis instance
resource "aws_instance" "redis" {
  ami                    = data.aws_ami.amazon_linux_2023.id
  instance_type          = "t3.micro"
  key_name               = var.key_name
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              # Update system
              dnf update -y
              # Install docker
              dnf install -y docker
              systemctl enable docker
              systemctl start docker
              usermod -aG docker ec2-user
              
              # Run redis docker container
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
              
              # Clone repository
              dnf install -y git
              git clone https://github.com/Qjr2023/real-time-price-aggregator.git
              cd real-time-price-aggregator
              
              # Build and run Exchange1
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
              
              # Create AWS credentials directory
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
              
              # Setup permissions
              chmod 600 /home/ec2-user/.aws/credentials
              chmod 600 /home/ec2-user/.aws/config
              chown -R ec2-user:ec2-user /home/ec2-user/.aws
              
              # Create startup script
              cat > start_server.sh <<SCRIPT
              #!/bin/bash
              cd /home/ec2-user/real-time-price-aggregator
              
              # Setup environment variables
              export REDIS_ADDR="${aws_instance.redis.private_ip}:6379"
              export EXCHANGE1_URL="http://${aws_instance.exchange1.private_ip}:8081/mock/ticker"
              export EXCHANGE2_URL="http://${aws_instance.exchange2.private_ip}:8082/mock/ticker"
              export EXCHANGE3_URL="http://${aws_instance.exchange3.private_ip}:8083/mock/ticker"
              export AWS_REGION="${var.region}"
              
              # Build and run API server
              docker build -t api-server -f Dockerfile .
              docker run -d -p 8080:8080 \\
                -v /home/ec2-user/.aws:/root/.aws:ro \\
                -e REDIS_ADDR="\$REDIS_ADDR" \\
                -e EXCHANGE1_URL="\$EXCHANGE1_URL" \\
                -e EXCHANGE2_URL="\$EXCHANGE2_URL" \\
                -e EXCHANGE3_URL="\$EXCHANGE3_URL" \\
                -e AWS_REGION="\$AWS_REGION" \\
                api-server
              SCRIPT
              
              chmod +x start_server.sh
              ./start_server.sh
              EOF

  depends_on = [
    aws_instance.redis,
    aws_instance.exchange1,
    aws_instance.exchange2,
    aws_instance.exchange3,
    aws_dynamodb_table.prices_table
  ]

  tags = {
    Name = "price-aggregator-api-server"
  }
}

# Monitoring instance
resource "aws_instance" "monitoring" {
  ami                    = data.aws_ami.amazon_linux_2023.id
  instance_type          = "t3.small"
  key_name               = var.key_name
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              dnf update -y
              dnf install -y docker git
              systemctl enable docker
              systemctl start docker
              usermod -aG docker ec2-user

              git clone https://github.com/Qjr2023/real-time-price-aggregator.git
              cd real-time-price-aggregator
              
              # create directories for Prometheus and Grafana
              mkdir -p /home/ec2-user/grafana_data
              chmod 777 /home/ec2-user/grafana_data
              mkdir -p /home/ec2-user/grafana/provisioning/datasources
              mkdir -p /home/ec2-user/grafana/provisioning/dashboards
              chmod -R 777 /home/ec2-user/grafana
              
              # update Prometheus config
              cat > /home/ec2-user/prometheus.yml <<PROMCONFIG
              global:
                scrape_interval: 15s
                evaluation_interval: 15s

              scrape_configs:
                - job_name: 'price-aggregator'
                  scrape_interval: 5s
                  static_configs:
                    - targets: ['${aws_instance.api_server.private_ip}:8080']

                - job_name: 'prometheus'
                  scrape_interval: 5s
                  static_configs:
                    - targets: ['localhost:9090']
              PROMCONFIG
              
              # create Grafana datasource config
              cat > /home/ec2-user/grafana/provisioning/datasources/datasource.yml <<DATASOURCE
              apiVersion: 1
              datasources:
                - name: Prometheus
                  type: prometheus
                  url: http://localhost:9090
                  access: proxy
                  isDefault: true
              DATASOURCE
              
              # create Grafana dashboard config
              cat > /home/ec2-user/grafana/provisioning/dashboards/dashboard.yml <<DASHBOARD
              apiVersion: 1
              providers:
                - name: 'default'
                  orgId: 1
                  folder: ''
                  type: file
                  disableDeletion: false
                  updateIntervalSeconds: 10
                  options:
                    path: /etc/grafana/provisioning/dashboards
              DASHBOARD
              
              # copy Grafana dashboard JSON
              cp grafana/dashboards/price_aggregator.json /home/ec2-user/grafana/provisioning/dashboards/
              
              # run Prometheus container
              docker run -d -p 9090:9090 \
                -v /home/ec2-user/prometheus.yml:/etc/prometheus/prometheus.yml \
                -v /home/ec2-user/prometheus_data:/prometheus \
                --name prometheus \
                prom/prometheus:latest
              
              # run Grafana container
              docker run -d -p 3000:3000 \
                -v /home/ec2-user/grafana_data:/var/lib/grafana \
                -v /home/ec2-user/grafana/provisioning:/etc/grafana/provisioning \
                -e "GF_SECURITY_ADMIN_PASSWORD=admin" \
                -e "GF_USERS_ALLOW_SIGN_UP=false" \
                -e "GF_INSTALL_PLUGINS=grafana-piechart-panel" \
                --name grafana \
                grafana/grafana:latest
              EOF

  tags = {
    Name = "price-aggregator-monitoring"
  }
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

output "api_health_check" {
  value = "curl http://${aws_instance.api_server.public_ip}:8080/health"
}

output "monitoring_ip" {
  value = aws_instance.monitoring.public_ip
}

output "prometheus_url" {
  value = "http://${aws_instance.monitoring.public_ip}:9090"
}

output "grafana_url" {
  value = "http://${aws_instance.monitoring.public_ip}:3000 (login with admin/admin)"
}
