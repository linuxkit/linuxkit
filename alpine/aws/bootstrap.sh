#!/bin/sh

docker cluster init
docker network create editionsmanager
cat << EOF >/tmp/taco-web.yml
version: '3'
namespace: editionsmanager

# Currently must run 'docker network create editionsmanager' before applying

services:
    frontend:
        image: kencochrane/taco-web:poc5
        instances: 1
        env:
            - MOBY_AMI=$MOBY_AMI
            - AWS_SUBNET_ID=$AWS_SUBNET_ID
            - AWS_DEFAULT_REGION=$AWS_DEFAULT_REGION
            - AWS_SECURITY_GROUP=$AWS_SECURITY_GROUP
            - AWS_SECURITY_GROUP_ID=$AWS_SECURITY_GROUP_ID
            - PUBLIC_HOSTNAME=$HOSTNAME
            - MANAGER_PASSWORD=$MANAGER_PASSWORD
            - CLUSTER_NAME=$CLUSTER_NAME
        networks:
            - mynetwork
        ports:
            - name: frontend
              protocol: tcp
              port: 9090
              node_port: 80
            - name: mystery
              protocol: tcp
              port: 1234
              node_port:
        mounts:
            - type: bind
              source: /var/run/libmachete.sock
              target: /var/run/libmachete.sock
              mask: rw
            - type: bind
              source: /root/.docker
              target: /root/.docker
              mask: rw
            - type: bind
              source: /root/.aws
              target: /root/.aws
              mask: rw
EOF
docker project -f /tmp/taco-web.yml init
