#!/bin/bash
# Run MDTTER Live Visualization Demo
# Shows Gemini's vision of attacks as trajectories through behavioral manifolds

set -e

DEMO_DIR="$(dirname "$0")"
PROJECT_ROOT="$(dirname "$DEMO_DIR")"

echo "🌌 Building MDTTER Visualization Demo..."
cd "$PROJECT_ROOT"

# Build the visualization demo
go build -o "$DEMO_DIR/mdtter_viz" "$DEMO_DIR/mdtter_live_visualization.go"

echo "✅ Build complete"
echo
echo "🚀 Launching MDTTER Live Attack Visualization..."
echo "   Watch as the attack evolves through behavioral space!"
echo

# Run with dramatic effect
"$DEMO_DIR/mdtter_viz"

# After the demo, show what's possible with full 3D
echo
echo "🎨 NEXT LEVEL: 3D Visualization Integration"
echo "═══════════════════════════════════════════"
echo
echo "What you just saw in ASCII can be rendered as:"
echo "• Interactive 3D trajectories through behavioral space"
echo "• Real-time manifold morphing as attacks evolve"
echo "• VAM threshold surfaces you can rotate and explore"
echo "• Attack clustering showing similar behavioral patterns"
echo
echo "📊 Export data for D3.js visualization:"
echo "   ./mdtter_viz --export-d3 > attack_topology.json"
echo
echo "🔗 Then open in browser with WebGL for full experience"
echo

# Create a sample D3.js visualization template
cat > "$DEMO_DIR/mdtter_3d_template.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>MDTTER 3D Attack Visualization</title>
    <script src="https://d3js.org/d3.v7.min.js"></script>
    <script src="https://unpkg.com/three@0.150.0/build/three.min.js"></script>
    <style>
        body { margin: 0; font-family: Arial, sans-serif; background: #000; }
        #viz { width: 100vw; height: 100vh; }
        #info { position: absolute; top: 10px; left: 10px; color: #0ff; }
    </style>
</head>
<body>
    <div id="viz"></div>
    <div id="info">
        <h2>MDTTER: Attack Trajectory Visualization</h2>
        <p>🌌 Behavioral Manifold Explorer</p>
        <p>Red surface: VAM > 0.7 defensive trigger boundary</p>
        <p>Use mouse to rotate, scroll to zoom</p>
    </div>
    <script>
        // Placeholder for full 3D implementation
        // Would load attack_topology.json and render:
        // - Attack trajectory as glowing path
        // - VAM threshold as translucent surface
        // - Intent transitions as color gradients
        // - Topology nodes as interactive spheres
        
        console.log("Full 3D visualization available in production version");
        console.log("Contact Strigoi team for WebGL implementation");
    </script>
</body>
</html>
EOF

echo
echo "📁 3D template created at: $DEMO_DIR/mdtter_3d_template.html"
echo
echo "🐺 Sister Gemini's vision is real. The pack sees in dimensions legacy systems cannot imagine."