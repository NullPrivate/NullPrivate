#!/bin/bash
set -euo pipefail
set -x

# warn if not root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root"
    exit
fi

# install nodejs
curl -fsSL https://deb.nodesource.com/setup_21.x | sudo -E bash - && sudo apt-get install -y nodejs

# install npm
sudo apt install npm

# install yarn
npm install -g yarn

# install go
wget https://go.dev/dl/go1.22.1.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.22.1.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# install tools
sudo apt install -y gpg gzip sed tar zip

# In order to build AdGuard Home with Node.js 17 and later, specify `--openssl-legacy-provider` option.
export NODE_OPTIONS=--openssl-legacy-provider

# build dist
make build-release CHANNEL='release' VERSION='0.0.1' ARCH='amd64 arm64' OS='linux' SIGN='0'

# build docker
docker run --rm --privileged multiarch/qemu-user-static --reset -p yes --credential yes
docker buildx rm buildx-builder
docker buildx create --name buildx-builder --driver docker-container --use
# make build-docker CHANNEL='release' VERSION='0.0.1' DOCKER_IMAGE_NAME='adguard-private'
make build-docker CHANNEL='beta' VERSION='0.0.1' DOCKER_IMAGE_NAME='adguardhome' DOCKER_OUTPUT='type=image,name=docker.io/jqknono/adguardhome:v1.0,push=false'

# alpine证书错误解决方法
# nginx -s stop -c /home/test/code/nullprivate/temp/master/app/nginx/nginx.conf -p /home/test/code/nullprivate/temp/master/app/nginx
# sudo iptables -D  PREROUTING -i ens33 -t nat -p tcp --dport 80 -j REDIRECT --to-port 1080
# sudo iptables -D  PREROUTING -i ens33 -t nat -p tcp --dport 443 -j REDIRECT --to-port 1443
# sudo iptables -D  PREROUTING -i ens33 -t nat -p tcp --dport 853 -j REDIRECT --to-port 1853
# sudo ip6tables -D PREROUTING -i ens33 -t nat -p tcp --dport 80 -j REDIRECT --to-port 1080
# sudo ip6tables -D PREROUTING -i ens33 -t nat -p tcp --dport 443 -j REDIRECT --to-port 1443
# sudo ip6tables -D PREROUTING -i ens33 -t nat -p tcp --dport 853 -j REDIRECT --to-port 1853

# 推送到docker
docker login --username=jqknono
docker tag jqknono/adguardhome:v1.0 jqknono/adguardhome:v1.0
docker push jqknono/adguardhome:v1.0

# 推送到aliyun
docker login --username=jqknono@outlook.com https://registry.cn-hangzhou.aliyuncs.com
docker tag jqknono/adguardhome:v1.0 registry.cn-hangzhou.aliyuncs.com/jqknono/adguardhome:v1.0
docker push registry.cn-hangzhou.aliyuncs.com/jqknono/adguardhome:v1.0

# 推送到自维护generic registry v2
docker login --username=jqknono https://dev.jqknono.com:34005
docker tag jqknono/adguardhome:v1.0 dev.jqknono.com:34005/adguardhome:v1.0
docker push dev.jqknono.com:34005/adguardhome:v1.0
