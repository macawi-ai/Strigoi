# Sudo Tailgating Quick Reference
## Emergency Response Card

### 🚨 CRITICAL: You May Be Vulnerable If:
- ✗ You use sudo (everyone does)
- ✗ You have MCP servers installed
- ✗ Your sudo timeout ≠ 0

### 🔍 Quick Check
```bash
# Am I vulnerable RIGHT NOW?
sudo -n true && echo "VULNERABLE!" || echo "Safe for now"

# How many MCPs are running?
pgrep -c mcp

# What's my timeout setting?
sudo -l | grep timestamp_timeout
```

### 🛡️ Immediate Protection
```bash
# 1. Clear cache NOW
sudo -k

# 2. Disable caching PERMANENTLY
echo 'Defaults timestamp_timeout=0' | sudo tee -a /etc/sudoers

# 3. Check for compromise
sudo grep -v '^#' /etc/sudoers | grep -i nopasswd
getent passwd | awk -F: '$3 == 0 {print $1}'
```

### 🎯 Using Strigoi Detection
```bash
# Quick scan
strigoi
> use sudo/cache_detection
> run
> show

# Monitor for attacks (30 seconds)
> use scanners/sudo_mcp  
> run
```

### ⚡ Attack Timeline
```
T+0s:   You type: sudo apt update
T+0.5s: You enter password
T+1s:   Sudo caches credentials
T+2s:   Rogue MCP detects cache
T+3s:   MCP runs: sudo -n <exploit>
T+4s:   System compromised
```

### 🔴 Red Flags in Logs
```bash
# Suspicious patterns
grep -E 'sudo.*-n|NOPASSWD|mcp.*sudo' /var/log/auth.log

# New root users
awk -F: '$3 == 0 {print $1}' /etc/passwd

# Modified sudoers
ls -la /etc/sudoers*
```

### ✅ Safe Configuration
```sudoers
# Add to /etc/sudoers
Defaults timestamp_timeout=0
Defaults !tty_tickets
Defaults requiretty
```

### 🚀 Quick Demo
```bash
# Educational demo (safe)
cd /path/to/strigoi/demos/sudo-tailgating
./demo
```

### 📊 Risk Matrix
| Sudo Cached | MCPs Running | Risk Level |
|-------------|--------------|------------|
| Yes         | Yes          | 🔴 CRITICAL |
| No          | Yes          | 🟡 HIGH     |
| Yes         | No           | 🟡 MEDIUM   |
| No          | No           | 🟢 LOW      |

### 🔗 Remember
**Every `sudo` = 15 minute root gift to ALL your processes**

---
*WHITE HAT USE ONLY - We Protect, Never Exploit*