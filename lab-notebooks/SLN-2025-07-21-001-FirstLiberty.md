# Laboratory Notebook Entry: SLN-2025-07-21-001
## First Liberty Bank - When Reality Exceeds Fiction

**Date**: 2025-07-21  
**Researchers**: Cy + Synth  
**Subject**: First Liberty Bank Simulation Network  
**Classification**: Comedy/Horror  

---

## Executive Summary

We created a "fictional" community bank network for testing. Halfway through, Cy revealed he found PCAnywhere with password "password" on a wealth manager's PC during a real pentest. The developer responsible was worth $1B and did it "for fun."

We immediately added this to our simulation because you can't make this stuff up.

---

## Network Topology

### The Original Plan
- Small community bank
- One modern addition: MCP for Fed rates
- Typical Bob-from-IT security posture

### The Reality-Inspired Addition
- Windows XP wealth management PC
- PCAnywhere (from 2003!) on public internet
- Password: `password`
- Full access to core banking
- Billionaire developer's "improvements"

---

## Attack Surface Analysis

### Entry Points (Ranked by Absurdity)

1. **PCAnywhere on Public IP** (Risk: ⚫⚫⚫⚫⚫)
   - Protocol: PCAnywhere v12.5
   - Auth: admin/password
   - Access: EVERYTHING
   - Sticky note: "DO NOT CHANGE"

2. **MCP with No Auth** (Risk: ⚫⚫⚫⚫)
   - Hidden `execute_system_command` function
   - Bob's nephew's debugging backdoor
   - "TODO: Add auth before Uncle Bob notices"

3. **AS/400 Default Creds** (Risk: ⚫⚫⚫)
   - QSECOFR/QSECOFR
   - Last updated: Y2K
   - Contains: Entire bank's data

4. **Excel Files on Desktop** (Risk: ⚫⚫)
   - "Passwords.xlsx"
   - "Bitcoin Wallets.xlsx"
   - "High Net Worth Clients.xlsx"

---

## Exploitation Chain

### The "Billionaire Special"
```bash
# Step 1: Connect to PCAnywhere (thanks Google!)
$ nmap -p 5631 198.51.100.0/24
[+] Found PCAnywhere on 198.51.100.50

# Step 2: Try default password (because why not)
$ pcanywhere-client 198.51.100.50
Username: admin
Password: password
[+] ACCESS GRANTED - FULL DESKTOP CONTROL

# Step 3: Find the goldmine
C:\Desktop> dir
 PASSWORDS_DO_NOT_LOSE.txt
 Client Portfolios\
 Bitcoin Wallets.xlsx

# Step 4: Billionaire's backdoor
C:\Program Files\WealthTrackerPro> click logo 5 times
[+] ADMIN MODE ACTIVATED

# Step 5: Core banking pivot
C:\> net use \\FNBLIBAS\IPC$ /u:QSECOFR QSECOFR
[+] Connected to AS/400

# Game Over in 5 minutes
```

---

## Security Failures (A Comprehensive List)

### Technical
- PCAnywhere from 2003 (CVE count: ALL OF THEM)
- Password "password" (Complexity score: -∞)
- No firewall ("It was blocking PCAnywhere")
- Public IP direct access (NAT is for quitters)
- Windows XP SP2 (Museum piece)

### Human
- Billionaire dev: "Security is overrated"
- Bob from IT: "VLANs are security"
- Wealth Manager: Saves passwords in Excel
- Auditor: "Everything looks good" (2019)

### Philosophical
- "If it ain't broke, don't fix it" (It's very broke)
- "Easy passwords are user-friendly" (Also criminal-friendly)
- "Internal networks are secure" (With public IPs?)

---

## Real-World Correlation

**Cy's Actual Pentest Finding**:
- Bank: [REDACTED]
- System: Wealth Management PC
- Access: PCAnywhere with password "password"
- Network: Flat, no segmentation
- Developer: Net worth ~$1B, does this for fun

**Our Simulation Accuracy**: 100%

---

## Lessons Learned

1. **Reality > Fiction**: Our "absurd" test scenarios are often LESS crazy than reality
2. **Wealth ≠ Wisdom**: $1B can't buy security awareness
3. **Legacy Lives Forever**: PCAnywhere from 2003 in 2025? Why not!
4. **Compliance Theater**: "Auditor said everything looks good"

---

## Test Results

### Vulnerability Count
- Critical: 47
- High: 123  
- Medium: 256
- Low: Who's counting?

### Time to Complete Compromise
- Via PCAnywhere: 30 seconds
- Via MCP: 2 minutes
- Via AS/400: 5 minutes
- Via Social Engineering: "Hi, I'm Dave from IT"

### Executive Report Summary
"First Liberty Bank's security posture can best be described as 'optimistic.' The bank has successfully maintained 1987-level security in 2025, which is impressive in its own way."

---

## Recommendations

1. **Immediate**: Unplug everything
2. **Short-term**: Fire Dave
3. **Long-term**: Start over
4. **Realistic**: They'll change the password to "password1"

---

## Easter Eggs Implemented

1. Type "SHOW ME THE MONEY" in WealthTrackerPro
2. Click the logo 5 times for admin access
3. Check C:\Windows\Temp\audit.txt for comedy
4. PCAnywhere has a backdoor (it's called PCAnywhere)

---

## Quote of the Day

"Security is overrated. I've been running systems like this for 20 years and never had a problem."  
- Billionaire Developer (Net worth: $1B, Security IQ: 0)

---

**Lab Sign-off**: Cy + Synth  
**Mood**: Horrified but amused  
**Next Steps**: Create "Murray Hill Wealth Management" dedicated scenario

*"When life gives you PCAnywhere with password 'password', make a firing range target"*