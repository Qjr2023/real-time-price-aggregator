provider "aws" {
  region = "us-west-2"
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
  key_name               = "cs6650hw1b"
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
  key_name               = "cs6650hw1b"
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
  key_name               = "cs6650hw1b"
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
  key_name               = "cs6650hw1b"
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
  key_name               = "cs6650hw1b"
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
              sed -i 's/"http:\/\/exchange1:8081\/mock\/ticker"/os.Getenv("EXCHANGE1_URL")/g' cmd/main.go
              sed -i 's/"http:\/\/exchange2:8082\/mock\/ticker"/os.Getenv("EXCHANGE2_URL")/g' cmd/main.go
              sed -i 's/"http:\/\/exchange3:8083\/mock\/ticker"/os.Getenv("EXCHANGE3_URL")/g' cmd/main.go
              
              # create AWS credentials file
              mkdir -p /home/ec2-user/.aws
              cat > /home/ec2-user/.aws/credentials <<CREDENTIALS
              [default]
              aws_access_key_id=your_access_key_id
              aws_secret_access_key=your_secret_access_key
              aws_session_token=your_session_token
              CREDENTIALS
              
              cat > /home/ec2-user/.aws/config <<CONFIG
              [default]
              region=us-west-2
              CONFIG
              
              # setup permissions
              chmod 600 /home/ec2-user/.aws/credentials
              chmod 600 /home/ec2-user/.aws/config

              # setup environment variables
              export REDIS_ADDR="${aws_instance.redis.private_ip}:6379"
              export EXCHANGE1_URL="http://${aws_instance.exchange1.private_ip}:8081/mock/ticker"
              export EXCHANGE2_URL="http://${aws_instance.exchange2.private_ip}:8082/mock/ticker"
              export EXCHANGE3_URL="http://${aws_instance.exchange3.private_ip}:8083/mock/ticker"
              
              # build and run API server
              cat > start_server.sh <<SCRIPT
              #!/bin/bash
              cd /home/ec2-user/real-time-price-aggregator
              
              # setup environment variables
              export REDIS_ADDR="${aws_instance.redis.private_ip}:6379"
              export EXCHANGE1_URL="http://${aws_instance.exchange1.private_ip}:8081/mock/ticker"
              export EXCHANGE2_URL="http://${aws_instance.exchange2.private_ip}:8082/mock/ticker"
              export EXCHANGE3_URL="http://${aws_instance.exchange3.private_ip}:8083/mock/ticker"
              export AWS_REGION="us-west-2"

              # build and run API server
              docker build -t api-server -f Dockerfile .
              docker run -d -p 8080:8080 \
                -v /home/ec2-user/.aws:/root/.aws:ro \\
                -e REDIS_ADDR="\$REDIS_ADDR" \\
                -e EXCHANGE1_URL="\$EXCHANGE1_URL" \\
                -e EXCHANGE2_URL="\$EXCHANGE2_URL" \\
                -e EXCHANGE3_URL="\$EXCHANGE3_URL" \\
                -e AWS_REGION="us-west-2" \\
                api-server
              SCRIPT
              
              chmod +x start_server.sh
              ./start_server.sh
              EOF

  depends_on = [
    aws_instance.redis,
    aws_instance.exchange1,
    aws_instance.exchange2,
    aws_instance.exchange3
  ]

  tags = {
    Name = "price-aggregator-api-server"
  }
}

# output the public IPs of the instances
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