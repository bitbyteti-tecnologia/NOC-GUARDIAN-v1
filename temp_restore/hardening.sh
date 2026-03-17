#!/usr/bin/env bash
# hardening.sh
# - Aplica UFW, Fail2Ban, endurece SSH, sysctl e auto-updates.

set -e

# UFW
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# Fail2ban
apt-get update && apt-get install -y fail2ban unattended-upgrades
systemctl enable --now fail2ban

# SSH Hardening
sed -i 's/^#\?PermitRootLogin .*/PermitRootLogin no/' /etc/ssh/sshd_config
sed -i 's/^#\?PasswordAuthentication .*/PasswordAuthentication no/' /etc/ssh/sshd_config
sed -i 's/^#\?PubkeyAuthentication .*/PubkeyAuthentication yes/' /etc/ssh/sshd_config
systemctl restart ssh || systemctl restart sshd

# Kernel Security
cat <<SYSCTL >/etc/sysctl.d/99-guardian.conf
net.ipv4.conf.all.rp_filter=1
net.ipv4.conf.default.rp_filter=1
net.ipv4.conf.all.accept_source_route=0
net.ipv4.conf.default.accept_source_route=0
net.ipv4.icmp_echo_ignore_broadcasts=1
net.ipv4.icmp_ignore_bogus_error_responses=1
net.ipv4.conf.all.accept_redirects=0
net.ipv4.conf.default.accept_redirects=0
net.ipv4.conf.all.send_redirects=0
net.ipv6.conf.all.accept_redirects=0
kernel.dmesg_restrict=1
SYSCTL
sysctl --system

# Auto updates
dpkg-reconfigure -plow unattended-upgrades
