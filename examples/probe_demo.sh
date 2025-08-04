#!/bin/bash
# Strigoi Probe Modules Demonstration

echo "=== Strigoi Probe Modules Demo ==="
echo

# Test project setup
echo "[1] Setting up test project..."
mkdir -p demo_project
cd demo_project

# Create a vulnerable Node.js project
cat > package.json << 'EOF'
{
  "name": "vulnerable-app",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.17.1",
    "lodash": "^4.17.19",
    "axios": "^0.21.1",
    "jsonwebtoken": "^8.5.0"
  }
}
EOF

cat > app.js << 'EOF'
const express = require('express');
const config = require('./config');

const app = express();

// Hardcoded secret - bad practice!
const JWT_SECRET = 'super-secret-key-12345';

// API integration
const stripeKey = 'sk_test_4eC39HqLyjWDarjtT1zdp7dc';
const apiEndpoint = 'https://api.stripe.com/v1/charges';

// Debug endpoint - should not exist in production
app.get('/debug/info', (req, res) => {
  res.json({
    env: process.env,
    config: config,
    stack: new Error().stack
  });
});

// Verbose error handler
app.use((err, req, res, next) => {
  console.error('Full stack trace:', err.stack);
  res.status(500).json({
    error: err.message,
    stack: err.stack,
    file: __filename
  });
});

app.listen(3000);
EOF

cat > config.js << 'EOF'
module.exports = {
  database: {
    host: 'db.internal.com',
    password: 'admin123'
  },
  aws: {
    accessKeyId: 'AKIAIOSFODNN7EXAMPLE',
    secretAccessKey: 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY'
  },
  github: {
    token: 'ghp_1234567890abcdef1234567890abcdef1234'
  }
};
EOF

cd ..

echo
echo "[2] Running Probe/North (API Endpoints)..."
./strigoi probe north http://localhost:3000 --timeout 5s || echo "(Expected to fail if server not running)"

echo
echo "[3] Running Probe/South (Dependencies)..."
./strigoi probe south demo_project

echo
echo "[4] Running Probe/East (Data Flows)..."
./strigoi probe east demo_project --verbose

echo
echo "[5] Saving session for reuse..."
./strigoi session save security-scan-demo \
  --description "Demo security scan configuration" \
  --tags "demo,vulnerable" \
  --overwrite

echo
echo "[6] Module information..."
./strigoi module info probe/south
echo
./strigoi module info probe/east

echo
echo "=== Demo Complete ==="
echo
echo "Key findings:"
echo "- South: Identifies vulnerable dependencies (lodash 4.17.19 has known CVEs)"
echo "- East: Detects hardcoded secrets, API keys, and debug endpoints"
echo "- North: Maps API endpoints and attack surface"

# Cleanup
rm -rf demo_project