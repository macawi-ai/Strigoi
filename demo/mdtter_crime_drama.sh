#!/bin/bash
# MDTTER Crime Drama - "The Case of the Morphing Menace"
# A narrative-driven demo showing how MDTTER catches sophisticated attacks

set -e

# Colors for dramatic effect
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Dramatic pause function
dramatic_pause() {
    sleep "${1:-2}"
}

clear

echo -e "${PURPLE}┌─────────────────────────────────────────────────────────────┐${NC}"
echo -e "${PURPLE}│${WHITE}           🔍 MDTTER CRIME DRAMA PRESENTS 🔍                 ${PURPLE}│${NC}"
echo -e "${PURPLE}│${CYAN}         'The Case of the Morphing Menace'                   ${PURPLE}│${NC}"
echo -e "${PURPLE}│${WHITE}    How Multi-Dimensional Analysis Caught the Uncatchable    ${PURPLE}│${NC}"
echo -e "${PURPLE}└─────────────────────────────────────────────────────────────┘${NC}"
echo
dramatic_pause 3

echo -e "${YELLOW}[NARRATOR]${NC} Our story begins on a quiet Tuesday morning at MegaCorp..."
echo -e "${YELLOW}[NARRATOR]${NC} The legacy SIEM hummed quietly, seeing nothing unusual..."
echo
dramatic_pause

# Act 1: The Reconnaissance
echo -e "${CYAN}═══ ACT 1: THE STRANGER AT THE DOOR ═══${NC}"
echo
echo -e "${WHITE}[09:42:17]${NC} Legacy SIEM logs:"
echo -e "  └─ ${RED}HTTP OPTIONS${NC} request from 192.168.1.100 → 10.0.0.50"
echo -e "     ${WHITE}Status: Normal scan, low priority alert${NC}"
echo
dramatic_pause

echo -e "${GREEN}[MDTTER SEES]${NC} 🌌 Something deeper..."
echo -e "  ├─ ${YELLOW}VAM: 0.42${NC} - Moderate novelty detected"
echo -e "  ├─ Behavioral embedding shows ${CYAN}reconnaissance pattern${NC}"
echo -e "  ├─ Trajectory begins at origin with ${RED}unusual curvature${NC}"
echo -e "  └─ ${PURPLE}Intent probability: Reconnaissance 78%${NC}"
echo
echo -e "${YELLOW}[NARRATOR]${NC} MDTTER's behavioral manifold detected something the flat logs missed..."
echo -e "${YELLOW}[NARRATOR]${NC} This wasn't a routine scan. The attack trajectory had begun."
echo
dramatic_pause 3

# Act 2: The Credential Theft
echo -e "${CYAN}═══ ACT 2: THE STOLEN KEY ═══${NC}"
echo
echo -e "${WHITE}[09:43:52]${NC} Legacy SIEM logs:"
echo -e "  └─ ${RED}HTTP GET${NC} /api/config - API key in header"
echo -e "     ${WHITE}Status: Logged as normal API usage${NC}"
echo
dramatic_pause

echo -e "${GREEN}[MDTTER SEES]${NC} 🚨 A critical shift!"
echo -e "  ├─ ${RED}VAM: 0.68${NC} - Approaching defensive threshold!"
echo -e "  ├─ ${YELLOW}Velocity change: 47°${NC} - BEHAVIORAL SHIFT DETECTED"
echo -e "  ├─ Trajectory ${RED}curves sharply${NC} toward credential theft region"
echo -e "  ├─ ${PURPLE}Curvature spike: 0.82${NC} - Complex maneuver in progress"
echo -e "  └─ ${RED}⚡ Intent shift: Recon → Initial Access (82%)${NC}"
echo
echo -e "${YELLOW}[NARRATOR]${NC} The flat logs saw an API call. MDTTER saw a thief grabbing keys."
echo -e "${YELLOW}[NARRATOR]${NC} The behavioral manifold was morphing, the attack evolving..."
echo
dramatic_pause 3

# Act 3: The Lateral Movement
echo -e "${CYAN}═══ ACT 3: THE INSIDE JOB ═══${NC}"
echo
echo -e "${WHITE}[09:45:21]${NC} Legacy SIEM logs:"
echo -e "  └─ Multiple ${RED}HTTP POST${NC} requests to internal servers"
echo -e "     ${WHITE}Status: Internal traffic, no alerts${NC}"
echo
dramatic_pause

echo -e "${GREEN}[MDTTER SEES]${NC} 🔄 Topology transformation!"
echo -e "  ├─ ${RED}VAM: 0.85${NC} - ${RED}🔴 DEFENSIVE MORPH TRIGGERED!${NC}"
echo -e "  ├─ ${YELLOW}New topology edges created${NC}: 3 internal nodes compromised"
echo -e "  ├─ Attack surface ${RED}expanding rapidly${NC}"
echo -e "  ├─ ${PURPLE}Intent evolution: Lateral Movement 91%${NC}"
echo -e "  └─ ${CYAN}Predicted next: Privilege Escalation (67% probability)${NC}"
echo
echo -e "${YELLOW}[NARRATOR]${NC} While the SIEM dozed, MDTTER watched the attacker spread like wildfire."
echo -e "${YELLOW}[NARRATOR]${NC} The topology was screaming - but only MDTTER could hear it."
echo
dramatic_pause 3

# Act 4: The Heist
echo -e "${CYAN}═══ ACT 4: THE GREAT EXFILTRATION ═══${NC}"
echo
echo -e "${WHITE}[09:48:44]${NC} Legacy SIEM logs:"
echo -e "  └─ Large ${RED}HTTPS POST${NC} to external IP"
echo -e "     ${WHITE}Status: Logged, no context for severity${NC}"
echo
dramatic_pause

echo -e "${GREEN}[MDTTER SEES]${NC} 💥 The climax!"
echo -e "  ├─ ${RED}VAM: 0.94${NC} - EXTREME NOVELTY!"
echo -e "  ├─ ${YELLOW}Distance from normal: 8.7σ${NC} - Far outside baseline"
echo -e "  ├─ Complete ${RED}kill chain visible${NC} in trajectory:"
echo -e "  │   └─ Recon → Access → Lateral → Collection → ${RED}EXFILTRATION${NC}"
echo -e "  ├─ ${PURPLE}50MB payload detected${NC} leaving network"
echo -e "  └─ ${CYAN}Behavioral manifold shows classic data theft pattern${NC}"
echo
dramatic_pause 2

# The Reveal
echo -e "${CYAN}═══ THE REVEAL ═══${NC}"
echo
echo -e "${YELLOW}[NARRATOR]${NC} Let's see what each system caught..."
echo
dramatic_pause

echo -e "${RED}┌─ LEGACY SIEM REPORT ─────────────────────┐${NC}"
echo -e "${RED}│${NC} Events logged: 6                         ${RED}│${NC}"
echo -e "${RED}│${NC} Alerts raised: 0                         ${RED}│${NC}"
echo -e "${RED}│${NC} Attack detected: NO                      ${RED}│${NC}"
echo -e "${RED}│${NC} Context: None                            ${RED}│${NC}"
echo -e "${RED}│${NC} Prediction: N/A                          ${RED}│${NC}"
echo -e "${RED}└──────────────────────────────────────────┘${NC}"
echo
dramatic_pause

echo -e "${GREEN}┌─ MDTTER REPORT ──────────────────────────┐${NC}"
echo -e "${GREEN}│${NC} Events analyzed: 6                       ${GREEN}│${NC}"
echo -e "${GREEN}│${NC} Dimensional context: 128                 ${GREEN}│${NC}"
echo -e "${GREEN}│${NC} Attack detected: YES (at stage 2)        ${GREEN}│${NC}"
echo -e "${GREEN}│${NC} Kill chain mapped: Complete              ${GREEN}│${NC}"
echo -e "${GREEN}│${NC} Defensive morphs: 2                      ${GREEN}│${NC}"
echo -e "${GREEN}│${NC} Data exfiltrated: 50MB customer records  ${GREEN}│${NC}"
echo -e "${GREEN}└──────────────────────────────────────────┘${NC}"
echo
dramatic_pause 3

# The Aha Moments
echo -e "${PURPLE}═══ THE 'AHA!' MOMENTS ONLY MDTTER SAW ═══${NC}"
echo
echo -e "1️⃣  ${YELLOW}Stage 1:${NC} OPTIONS scan had ${RED}unusual behavioral embedding${NC}"
echo -e "    → Not a security researcher, but targeted reconnaissance"
echo
echo -e "2️⃣  ${YELLOW}Stage 2:${NC} API key access showed ${RED}47° velocity change${NC}"
echo -e "    → Attacker pivoting from recon to exploitation"
echo
echo -e "3️⃣  ${YELLOW}Stage 3:${NC} Lateral movement created ${RED}new topology edges${NC}"
echo -e "    → Internal network being mapped for crown jewels"
echo
echo -e "4️⃣  ${YELLOW}Stage 4:${NC} Exfiltration trajectory matched ${RED}known theft patterns${NC}"
echo -e "    → Could have been blocked if detected earlier!"
echo
dramatic_pause 3

# The Lesson
echo -e "${CYAN}═══ THE LESSON ═══${NC}"
echo
echo -e "${YELLOW}[NARRATOR]${NC} In the flat world of legacy SIEMs, this looked like normal traffic."
echo -e "${YELLOW}[NARRATOR]${NC} In MDTTER's multi-dimensional space, it was a heist in progress."
echo
echo -e "${WHITE}The difference?${NC}"
echo -e "  • ${GREEN}Behavioral context${NC} vs raw events"
echo -e "  • ${GREEN}Predictive trajectories${NC} vs reactive logging"
echo -e "  • ${GREEN}Topology awareness${NC} vs isolated alerts"
echo -e "  • ${GREEN}Intent evolution${NC} vs static categories"
echo
dramatic_pause 2

echo -e "${PURPLE}┌─────────────────────────────────────────────────────────────┐${NC}"
echo -e "${PURPLE}│${WHITE}     🐺 MDTTER: See the Hunt, Not Just the Tracks 🐺        ${PURPLE}│${NC}"
echo -e "${PURPLE}└─────────────────────────────────────────────────────────────┘${NC}"
echo

# Interactive prompt
echo -e "${CYAN}[INTERACTIVE]${NC} Want to explore the attack trajectory?"
echo -e "  ${YELLOW}1)${NC} View behavioral manifold evolution"
echo -e "  ${YELLOW}2)${NC} Examine topology morphing timeline"
echo -e "  ${YELLOW}3)${NC} Analyze intent probability shifts"
echo -e "  ${YELLOW}4)${NC} Export for 3D visualization"
echo
read -p "Select option (or press Enter to exit): " choice

case $choice in
    1)
        echo -e "\n${GREEN}Launching behavioral manifold explorer...${NC}"
        # Would launch interactive manifold viewer
        ;;
    2)
        echo -e "\n${GREEN}Showing topology evolution timeline...${NC}"
        # Would show topology changes over time
        ;;
    3)
        echo -e "\n${GREEN}Analyzing intent probability matrix...${NC}"
        # Would show intent evolution details
        ;;
    4)
        echo -e "\n${GREEN}Exporting trajectory data for 3D visualization...${NC}"
        echo "{\"trajectory\": [...], \"manifold\": {...}, \"topology\": [...]}" > mdtter_export.json
        echo "Data exported to mdtter_export.json"
        ;;
esac

echo
echo -e "${WHITE}Thank you for watching 'The Case of the Morphing Menace'${NC}"
echo -e "${CYAN}A true story, happening right now in networks everywhere.${NC}"
echo
echo -e "${YELLOW}Don't let your attacks go undetected.${NC}"
echo -e "${GREEN}See in dimensions. Hunt with MDTTER.${NC}"