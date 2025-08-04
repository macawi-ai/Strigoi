#!/bin/bash
# Session management demonstration

echo "=== Strigoi Session Management Demo ==="
echo

# Save a session without encryption
echo "[1] Saving unencrypted session..."
./strigoi session save demo-api-scan \
    --description "Demo API endpoint scan" \
    --tags "demo,api,test" \
    --overwrite

echo
echo "[2] List saved sessions..."
./strigoi session list

echo
echo "[3] Show detailed session info..."
./strigoi session info demo-api-scan

echo
echo "[4] Save encrypted session..."
./strigoi session save secure-scan \
    --description "Encrypted production scan" \
    --tags "production,secure" \
    --passphrase "demo123" \
    --overwrite

echo
echo "[5] List sessions with long format..."
./strigoi session list --long

echo
echo "[6] Load encrypted session (will fail without passphrase)..."
./strigoi session load secure-scan || echo "Failed as expected"

echo
echo "[7] Load encrypted session with passphrase..."
./strigoi session load secure-scan --passphrase "demo123"

echo
echo "[8] Filter sessions by tag..."
./strigoi session list --tags "production"

echo
echo "=== Demo Complete ==="