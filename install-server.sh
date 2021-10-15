#! /bin/bash

installMyProxy() {
    mkdir -p /etc/myproxy
    curl -Lo- https://github.com/sunshineplan/myproxy/releases/latest/download/release-linux.tar.gz | tar zxC /etc/myproxy
    cd /etc/myproxy
    chmod +x myproxy
}

configMyProxy() {
    touch /etc/myproxy/secrets
    read -p 'Please enter host (default: 0.0.0.0): ' host
    [ -z $host ] && host=0.0.0.0
    read -p 'Please enter port (default: 8080): ' port
    [ -z $port ] && port=8080
    read -p 'Please enter pre-shared key: ' psk
    read -p 'Please enter secrets path: ' secrets
    read -p 'Please enter cert path: ' cert
    read -p 'Please enter privkey path: ' privkey
    read -p 'Please enter access log path (default: /var/log/myproxy/access.log): ' access
    [ -z $access ] && access=/var/log/myproxy/access.log
    read -p 'Please enter error log path (default: /var/log/myproxy/error.log): ' error
    [ -z $error ] && error=/var/log/myproxy/error.log
    sed "s/\$host/$host/" /etc/myproxy/config.ini.server > /etc/myproxy/config.ini
    sed -i "s/\$port/$port/" /etc/myproxy/config.ini
    sed -i "s|\$psk|$psk|" /etc/myproxy/config.ini
    sed -i "s,\$secrets,$secrets," /etc/myproxy/config.ini
    sed -i "s,\$cert,$cert," /etc/myproxy/config.ini
    sed -i "s,\$privkey,$privkey," /etc/myproxy/config.ini
    sed -i "s,\$access,$access," /etc/myproxy/config.ini
    sed -i "s,\$error,$error," /etc/myproxy/config.ini
}

configSysctl() {
    cat >/etc/sysctl.d/90-tcp-keepalive-sysctl.conf <<-EOF
		net.ipv4.tcp_keepalive_time = 600
		net.ipv4.tcp_keepalive_intvl = 60
		net.ipv4.tcp_keepalive_probes = 20
		EOF
    sysctl --system
}

writeLogrotateScrip() {
    mkdir -p /var/log/myproxy
    cat >/etc/logrotate.d/myproxy <<-EOF
		/var/log/myproxy/access.log {
		    copytruncate
		    rotate 15
		    daily
		    compress
		    delaycompress
		    missingok
		    notifempty
		}

		/var/log/myproxy/error.log {
		    copytruncate
		    rotate 12
		    monthly
		    compress
		    delaycompress
		    missingok
		    notifempty
		}
		EOF
}

main() {
    installMyProxy
    configMyProxy
    configSysctl
    writeLogrotateScrip
    ./myproxy install || exit 1
    service myproxy start
}

main
