provider "aws" {
  region = "us-west-2"
  # 不需要显式提供凭证，会从环境变量自动获取
}

# 安全组 - 允许所有流量（仅用于实验环境）
resource "aws_security_group" "price_aggregator_sg" {
  name        = "price-aggregator-sg"
  description = "Allow all traffic for price aggregator services"

  # 允许所有入站流量
  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # 允许所有出站流量
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

# Redis实例
resource "aws_instance" "redis" {
  ami                    = "ami-0efcece6bed30fd98"  # Amazon Linux 2 AMI
  instance_type          = "t3.micro"
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              yum update -y
              amazon-linux-extras install docker -y
              service docker start
              usermod -a -G docker ec2-user
              chkconfig docker on
              
              # 运行Redis容器
              docker run -d -p 6379:6379 --name redis redis:latest
              EOF

  tags = {
    Name = "price-aggregator-redis"
  }
}

# Exchange1实例
resource "aws_instance" "exchange1" {
  ami                    = "ami-0efcece6bed30fd98"  # Amazon Linux 2 AMI
  instance_type          = "t3.micro"
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              yum update -y
              amazon-linux-extras install docker -y
              service docker start
              usermod -a -G docker ec2-user
              chkconfig docker on
              
              # 克隆仓库
              yum install git -y
              git clone https://github.com/Qjr2023/real-time-price-aggregator.git
              cd real-time-price-aggregator
              
              # 构建并启动Exchange1
              docker build -t exchange1 -f mocks/Dockerfile .
              docker run -d -p 8081:8081 exchange1 ./mock_server 8081 exchange1
              EOF

  tags = {
    Name = "price-aggregator-exchange1"
  }
}

# Exchange2实例
resource "aws_instance" "exchange2" {
  ami                    = "ami-0efcece6bed30fd98"  # Amazon Linux 2 AMI
  instance_type          = "t3.micro"
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              yum update -y
              amazon-linux-extras install docker -y
              service docker start
              usermod -a -G docker ec2-user
              chkconfig docker on
              
              # 克隆仓库
              yum install git -y
              git clone https://github.com/Qjr2023/real-time-price-aggregator.git
              cd real-time-price-aggregator
              
              # 构建并启动Exchange2
              docker build -t exchange2 -f mocks/Dockerfile .
              docker run -d -p 8082:8082 exchange2 ./mock_server 8082 exchange2
              EOF

  tags = {
    Name = "price-aggregator-exchange2"
  }
}

# Exchange3实例
resource "aws_instance" "exchange3" {
  ami                    = "ami-0efcece6bed30fd98"  # Amazon Linux 2 AMI
  instance_type          = "t3.micro"
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              yum update -y
              amazon-linux-extras install docker -y
              service docker start
              usermod -a -G docker ec2-user
              chkconfig docker on
              
              # 克隆仓库
              yum install git -y
              git clone https://github.com/Qjr2023/real-time-price-aggregator.git
              cd real-time-price-aggregator
              
              # 构建并启动Exchange3
              docker build -t exchange3 -f mocks/Dockerfile .
              docker run -d -p 8083:8083 exchange3 ./mock_server 8083 exchange3
              EOF

  tags = {
    Name = "price-aggregator-exchange3"
  }
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

# 主API服务器实例
resource "aws_instance" "api_server" {
  ami                    = data.aws_ami.amazon_linux_2023.id
  instance_type          = "t3.small"
  key_name               = "cs6650hw1b"
  vpc_security_group_ids = [aws_security_group.price_aggregator_sg.id]

  user_data = <<-EOF
              #!/bin/bash
              yum update -y
              amazon-linux-extras install docker -y
              service docker start
              usermod -a -G docker ec2-user
              chkconfig docker on
              
              # 克隆仓库
              yum install git -y
              git clone https://github.com/Qjr2023/real-time-price-aggregator.git
              cd real-time-price-aggregator
              
              # 修改main.go以支持环境变量
              sed -i 's/Addr: "redis:6379"/Addr: os.Getenv("REDIS_ADDR")/g' cmd/main.go
              sed -i 's/"http:\/\/exchange1:8081\/mock\/ticker"/os.Getenv("EXCHANGE1_URL")/g' cmd/main.go
              sed -i 's/"http:\/\/exchange2:8082\/mock\/ticker"/os.Getenv("EXCHANGE2_URL")/g' cmd/main.go
              sed -i 's/"http:\/\/exchange3:8083\/mock\/ticker"/os.Getenv("EXCHANGE3_URL")/g' cmd/main.go
              
              # 设置环境变量并构建运行服务器
              export REDIS_ADDR="${aws_instance.redis.private_ip}:6379"
              export EXCHANGE1_URL="http://${aws_instance.exchange1.private_ip}:8081/mock/ticker"
              export EXCHANGE2_URL="http://${aws_instance.exchange2.private_ip}:8082/mock/ticker"
              export EXCHANGE3_URL="http://${aws_instance.exchange3.private_ip}:8083/mock/ticker"
              
              # 创建启动脚本
              cat > start_server.sh <<'SCRIPT'
              #!/bin/bash
              cd /home/ec2-user/real-time-price-aggregator
              
              # 设置环境变量
              export REDIS_ADDR="${aws_instance.redis.private_ip}:6379"
              export EXCHANGE1_URL="http://${aws_instance.exchange1.private_ip}:8081/mock/ticker"
              export EXCHANGE2_URL="http://${aws_instance.exchange2.private_ip}:8082/mock/ticker"
              export EXCHANGE3_URL="http://${aws_instance.exchange3.private_ip}:8083/mock/ticker"
              export AWS_REGION="us-west-2"
              
              # 构建并启动服务器
              docker build -t api-server -f Dockerfile .
              docker run -d -p 8080:8080 \
                -e REDIS_ADDR="$REDIS_ADDR" \
                -e EXCHANGE1_URL="$EXCHANGE1_URL" \
                -e EXCHANGE2_URL="$EXCHANGE2_URL" \
                -e EXCHANGE3_URL="$EXCHANGE3_URL" \
                -e AWS_REGION="us-west-2" \
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

# 输出所有实例的IP地址
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