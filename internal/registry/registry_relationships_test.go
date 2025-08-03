package registry

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLevel3_EntityRelationships(t *testing.T) {
	// Create temporary database for testing
	tmpDir, err := os.MkdirTemp("", "strigoi_registry_rel_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_relationships.duckdb")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	defer registry.Close()

	ctx := context.Background()
	
	// Store entity IDs for use across test functions
	var moduleID, vulnID, attackID string

	t.Run("3.1_CreateTestEntities", func(t *testing.T) {
		// Create a module entity
		module := &Entity{
			EntityType:  EntityTypeMOD,
			Name:        "Config Scanner",
			Description: "Scans for configuration vulnerabilities",
			Status:      StatusActive,
			Author:      "Strigoi Team",
			Category:    "scanning",
			Tags:        []string{"config", "scanner"},
		}

		createdModule, err := registry.RegisterEntity(ctx, module)
		if err != nil {
			t.Errorf("Failed to register module: %v", err)
		}

		if createdModule.EntityType != EntityTypeMOD {
			t.Errorf("Expected EntityType MOD, got %s", createdModule.EntityType)
		}
		
		moduleID = createdModule.EntityID

		// Create a vulnerability entity
		vuln := &Entity{
			EntityType:  EntityTypeVUL,
			Name:        "Rogue MCP Sudo Tailgating",
			Description: "Vulnerability where MCP agents exploit sudo cache timing",
			Status:      StatusActive,
			Severity:    SeverityHigh,
			Author:      "Cy",
			Category:    "privilege-escalation",
			Tags:        []string{"sudo", "mcp", "timing"},
		}

		createdVuln, err := registry.RegisterEntity(ctx, vuln)
		if err != nil {
			t.Errorf("Failed to register vulnerability: %v", err)
		}

		if createdVuln.Severity != SeverityHigh {
			t.Errorf("Expected severity high, got %s", createdVuln.Severity)
		}
		
		vulnID = createdVuln.EntityID

		// Create an attack pattern entity
		attack := &Entity{
			EntityType:  EntityTypeATK,
			Name:        "Sudo Cache Timing Attack",
			Description: "Exploits sudo timestamp cache for privilege escalation",
			Status:      StatusActive,
			Severity:    SeverityHigh,
			Author:      "Red Team",
			Category:    "exploitation",
			Tags:        []string{"sudo", "timing", "escalation"},
		}

		createdAttack, err := registry.RegisterEntity(ctx, attack)
		if err != nil {
			t.Errorf("Failed to register attack: %v", err)
		}
		
		attackID = createdAttack.EntityID

		t.Logf("✓ Created entities: Module=%s, Vuln=%s, Attack=%s", 
			moduleID, vulnID, attackID)
	})

	t.Run("3.2_CreateBasicRelationships", func(t *testing.T) {
		// Use the stored entity IDs from the previous test
		if moduleID == "" || vulnID == "" || attackID == "" {
			t.Fatalf("Entity IDs not available from previous test")
		}

		// Create DETECTS relationship: Module → Vulnerability
		detectMetadata := map[string]interface{}{
			"detection_method": "behavioral_analysis",
			"confidence":       0.95,
			"false_positive_rate": 0.02,
		}

		rel1, err := registry.AddRelationship(ctx, moduleID, vulnID, 
			"DETECTS", 0.95, "Internal Testing", detectMetadata)
		if err != nil {
			t.Errorf("Failed to create DETECTS relationship: %v", err)
		}

		if rel1.RelationshipType != "DETECTS" {
			t.Errorf("Expected DETECTS relationship, got %s", rel1.RelationshipType)
		}

		// Create EXPLOITS relationship: Attack → Vulnerability
		exploitMetadata := map[string]interface{}{
			"exploitation_complexity": "medium",
			"prerequisites":           []string{"sudo_access", "timing_control"},
			"success_rate":           0.85,
		}

		rel2, err := registry.AddRelationship(ctx, attackID, vulnID, 
			"EXPLOITS", 0.85, "Red Team Analysis", exploitMetadata)
		if err != nil {
			t.Errorf("Failed to create EXPLOITS relationship: %v", err)
		}

		if rel2.RelationshipType != "EXPLOITS" {
			t.Errorf("Expected EXPLOITS relationship, got %s", rel2.RelationshipType)
		}

		t.Logf("✓ Created relationships: DETECTS and EXPLOITS")
	})

	t.Run("3.3_QueryRelationships", func(t *testing.T) {
		// Test GetRelationships for module
		moduleRels, err := registry.GetRelationships(ctx, moduleID)
		if err != nil {
			t.Errorf("Failed to get module relationships: %v", err)
		}

		if len(moduleRels) != 1 {
			t.Errorf("Expected 1 module relationship, got %d", len(moduleRels))
		}

		if moduleRels[0].RelationshipType != "DETECTS" {
			t.Errorf("Expected DETECTS relationship, got %s", moduleRels[0].RelationshipType)
		}

		// Test GetRelationships for vulnerability (should have 2: DETECTS and EXPLOITS)
		vulnRels, err := registry.GetRelationships(ctx, vulnID)
		if err != nil {
			t.Errorf("Failed to get vulnerability relationships: %v", err)
		}

		if len(vulnRels) != 2 {
			t.Errorf("Expected 2 vulnerability relationships, got %d", len(vulnRels))
		}

		// Test GetRelationshipsByType
		detectsRels, err := registry.GetRelationshipsByType(ctx, "DETECTS")
		if err != nil {
			t.Errorf("Failed to get DETECTS relationships: %v", err)
		}

		if len(detectsRels) != 1 {
			t.Errorf("Expected 1 DETECTS relationship, got %d", len(detectsRels))
		}

		exploitsRels, err := registry.GetRelationshipsByType(ctx, "EXPLOITS")
		if err != nil {
			t.Errorf("Failed to get EXPLOITS relationships: %v", err)
		}

		if len(exploitsRels) != 1 {
			t.Errorf("Expected 1 EXPLOITS relationship, got %d", len(exploitsRels))
		}

		t.Logf("✓ Relationship queries working correctly")
	})

	t.Run("3.4_SpecializedQueries", func(t *testing.T) {
		// Test FindVulnerabilitiesDetectedByModule
		detectedVulns, err := registry.FindVulnerabilitiesDetectedByModule(ctx, moduleID)
		if err != nil {
			t.Errorf("Failed to find vulnerabilities detected by module: %v", err)
		}

		if len(detectedVulns) != 1 {
			t.Errorf("Expected 1 detected vulnerability, got %d", len(detectedVulns))
		}

		if detectedVulns[0].EntityID != vulnID {
			t.Errorf("Expected %s, got %s", vulnID, detectedVulns[0].EntityID)
		}

		// Test FindModulesDetectingVulnerability
		detectingModules, err := registry.FindModulesDetectingVulnerability(ctx, vulnID)
		if err != nil {
			t.Errorf("Failed to find modules detecting vulnerability: %v", err)
		}

		if len(detectingModules) != 1 {
			t.Errorf("Expected 1 detecting module, got %d", len(detectingModules))
		}

		if detectingModules[0].EntityID != moduleID {
			t.Errorf("Expected %s, got %s", moduleID, detectingModules[0].EntityID)
		}

		t.Logf("✓ Specialized relationship queries working")
	})

	t.Run("3.5_AttackChainDiscovery", func(t *testing.T) {
		// Test FindAttackChains (Attack → Vulnerability → Module chains)
		chains, err := registry.FindAttackChains(ctx)
		if err != nil {
			t.Errorf("Failed to find attack chains: %v", err)
		}

		if len(chains) != 1 {
			t.Errorf("Expected 1 attack chain, got %d", len(chains))
		}

		chain := chains[0]
		if chain["attack_id"] != attackID {
			t.Errorf("Expected %s, got %v", attackID, chain["attack_id"])
		}

		if chain["vuln_id"] != vulnID {
			t.Errorf("Expected %s, got %v", vulnID, chain["vuln_id"])
		}

		if chain["module_id"] != moduleID {
			t.Errorf("Expected %s, got %v", moduleID, chain["module_id"])
		}

		// Check metadata is preserved
		if exploitMeta, ok := chain["exploit_metadata"].(map[string]interface{}); ok {
			if exploitMeta["exploitation_complexity"] != "medium" {
				t.Errorf("Expected medium complexity, got %v", exploitMeta["exploitation_complexity"])
			}
		} else {
			t.Errorf("Expected exploit metadata to be present")
		}

		t.Logf("✓ Attack chain discovery working: %s → %s → %s", 
			chain["attack_id"], chain["vuln_id"], chain["module_id"])
	})

	t.Run("3.6_RelationshipModification", func(t *testing.T) {
		// Test creating additional relationships
		// Create MITIGATES relationship: Module → Attack
		mitigateMetadata := map[string]interface{}{
			"mitigation_type": "detection_based",
			"effectiveness":   0.90,
		}

		_, err := registry.AddRelationship(ctx, moduleID, attackID, 
			"MITIGATES", 0.90, "Blue Team Analysis", mitigateMetadata)
		if err != nil {
			t.Errorf("Failed to create MITIGATES relationship: %v", err)
		}

		// Verify module now has 2 relationships (DETECTS + MITIGATES)
		moduleRels, err := registry.GetRelationships(ctx, moduleID)
		if err != nil {
			t.Errorf("Failed to get updated module relationships: %v", err)
		}

		if len(moduleRels) != 2 {
			t.Errorf("Expected 2 module relationships after adding MITIGATES, got %d", len(moduleRels))
		}

		// Test DeleteRelationship
		err = registry.DeleteRelationship(ctx, moduleID, attackID, "MITIGATES")
		if err != nil {
			t.Errorf("Failed to delete MITIGATES relationship: %v", err)
		}

		// Verify relationship was deleted
		moduleRelsAfterDelete, err := registry.GetRelationships(ctx, moduleID)
		if err != nil {
			t.Errorf("Failed to get module relationships after delete: %v", err)
		}

		if len(moduleRelsAfterDelete) != 1 {
			t.Errorf("Expected 1 module relationship after delete, got %d", len(moduleRelsAfterDelete))
		}

		t.Logf("✓ Relationship modification (add/delete) working")
	})

	t.Run("3.7_DependencyChains", func(t *testing.T) {
		// Create a second module that depends on the first
		depModule := &Entity{
			EntityType:  EntityTypeMOD,
			Name:        "Advanced Config Scanner",
			Description: "Enhanced configuration scanner with ML capabilities",
			Status:      StatusTesting,
			Author:      "Strigoi Team",
			Category:    "scanning",
			Tags:        []string{"config", "scanner", "ml"},
		}

		createdDepModule, err := registry.RegisterEntity(ctx, depModule)
		if err != nil {
			t.Errorf("Failed to register dependent module: %v", err)
		}

		// Create REQUIRES relationship
		requiresMetadata := map[string]interface{}{
			"dependency_type": "runtime",
			"minimum_version": "v1.0.0",
		}

		_, err = registry.AddRelationship(ctx, createdDepModule.EntityID, moduleID, 
			"REQUIRES", 1.0, "Architecture Design", requiresMetadata)
		if err != nil {
			t.Errorf("Failed to create REQUIRES relationship: %v", err)
		}

		// Verify dependency relationship
		reqRels, err := registry.GetRelationshipsByType(ctx, "REQUIRES")
		if err != nil {
			t.Errorf("Failed to get REQUIRES relationships: %v", err)
		}

		if len(reqRels) != 1 {
			t.Errorf("Expected 1 REQUIRES relationship, got %d", len(reqRels))
		}

		if reqRels[0].SourceEntityID != createdDepModule.EntityID {
			t.Errorf("Expected source to be dependent module, got %s", reqRels[0].SourceEntityID)
		}

		t.Logf("✓ Dependency chain created: %s REQUIRES %s", 
			createdDepModule.EntityID, reqRels[0].TargetEntityID)
	})

	t.Run("3.8_RelationshipMetadata", func(t *testing.T) {
		// Test that relationship metadata is preserved and queryable
		detectsRels, err := registry.GetRelationshipsByType(ctx, "DETECTS")
		if err != nil {
			t.Errorf("Failed to get DETECTS relationships: %v", err)
		}

		if len(detectsRels) != 1 {
			t.Errorf("Expected 1 DETECTS relationship, got %d", len(detectsRels))
		}

		rel := detectsRels[0]
		if rel.RelationshipMetadata == nil {
			t.Errorf("Expected relationship metadata to be present")
		}

		// Check specific metadata fields
		if confidence, ok := rel.RelationshipMetadata["confidence"].(float64); !ok || confidence != 0.95 {
			t.Errorf("Expected confidence 0.95, got %v", rel.RelationshipMetadata["confidence"])
		}

		if method, ok := rel.RelationshipMetadata["detection_method"].(string); !ok || method != "behavioral_analysis" {
			t.Errorf("Expected detection_method 'behavioral_analysis', got %v", rel.RelationshipMetadata["detection_method"])
		}

		t.Logf("✓ Relationship metadata preserved correctly")
	})

	t.Run("3.9_TimestampValidation", func(t *testing.T) {
		// Verify relationships have proper timestamps
		allRels, err := registry.GetRelationshipsByType(ctx, "DETECTS")
		if err != nil {
			t.Errorf("Failed to get relationships for timestamp validation: %v", err)
		}

		for _, rel := range allRels {
			if rel.CreatedAt.IsZero() {
				t.Errorf("Relationship CreatedAt timestamp is zero")
			}

			// Should be within last minute
			if time.Since(rel.CreatedAt) > time.Minute {
				t.Errorf("Relationship timestamp seems too old: %v", rel.CreatedAt)
			}
		}

		t.Logf("✓ Relationship timestamps valid")
	})

	t.Log("🎯 Level 3: Relationships & Dependencies - ALL TESTS PASSED")
}

func TestLevel3_EdgeCases(t *testing.T) {
	// Create temporary database for edge case testing
	tmpDir, err := os.MkdirTemp("", "strigoi_registry_edge_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_edge_cases.duckdb")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	defer registry.Close()

	ctx := context.Background()

	t.Run("3.10_EmptyRelationshipQueries", func(t *testing.T) {
		// Test queries when no relationships exist
		rels, err := registry.GetRelationships(ctx, "NONEXISTENT-ID")
		if err != nil {
			t.Errorf("Failed to query non-existent entity relationships: %v", err)
		}

		if len(rels) != 0 {
			t.Errorf("Expected 0 relationships for non-existent entity, got %d", len(rels))
		}

		// Test relationship type that doesn't exist
		typeRels, err := registry.GetRelationshipsByType(ctx, "NONEXISTENT_TYPE")
		if err != nil {
			t.Errorf("Failed to query non-existent relationship type: %v", err)
		}

		if len(typeRels) != 0 {
			t.Errorf("Expected 0 relationships for non-existent type, got %d", len(typeRels))
		}

		t.Logf("✓ Empty relationship queries handle correctly")
	})

	t.Run("3.11_NullMetadata", func(t *testing.T) {
		// Create entities for testing
		entity1 := &Entity{
			EntityType: EntityTypeMOD,
			Name:       "Test Module 1",
			Status:     StatusActive,
			Author:     "Test",
		}

		entity2 := &Entity{
			EntityType: EntityTypeVUL,
			Name:       "Test Vulnerability 1",
			Status:     StatusActive,
			Author:     "Test",
		}

		e1, _ := registry.RegisterEntity(ctx, entity1)
		e2, _ := registry.RegisterEntity(ctx, entity2)

		// Create relationship with nil metadata
		_, err := registry.AddRelationship(ctx, e1.EntityID, e2.EntityID, 
			"DETECTS", 1.0, "Test", nil)
		if err != nil {
			t.Errorf("Failed to create relationship with nil metadata: %v", err)
		}

		// Create relationship with empty metadata
		_, err = registry.AddRelationship(ctx, e1.EntityID, e2.EntityID, 
			"EXPLOITS", 1.0, "Test", map[string]interface{}{})
		if err != nil {
			t.Errorf("Failed to create relationship with empty metadata: %v", err)
		}

		t.Logf("✓ Null/empty metadata handled correctly")
	})

	t.Log("🎯 Level 3: Edge Cases - ALL TESTS PASSED")
}