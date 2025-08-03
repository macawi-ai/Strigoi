package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/macawi-ai/strigoi/internal/registry"
)

// TestResult tracks test outcomes
type TestResult struct {
	Name    string
	Passed  bool
	Message string
	Time    time.Duration
}

// TestSuite manages test execution
type TestSuite struct {
	Name    string
	Results []TestResult
}

func (ts *TestSuite) Run(name string, fn func() error) {
	start := time.Now()
	err := fn()
	result := TestResult{
		Name:   name,
		Passed: err == nil,
		Time:   time.Since(start),
	}
	if err != nil {
		result.Message = err.Error()
	}
	ts.Results = append(ts.Results, result)
}

func (ts *TestSuite) Report() {
	fmt.Printf("\n📊 TEST REPORT: %s\n", ts.Name)
	fmt.Println(strings.Repeat("=", 60))
	
	passed := 0
	for _, r := range ts.Results {
		status := "✅ PASS"
		if !r.Passed {
			status = "❌ FAIL"
		}
		fmt.Printf("%s %s (%.2fms)\n", status, r.Name, float64(r.Time.Microseconds())/1000)
		if r.Message != "" {
			fmt.Printf("   └─ %s\n", r.Message)
		}
		if r.Passed {
			passed++
		}
	}
	
	fmt.Printf("\nTotal: %d/%d passed (%.1f%%)\n", 
		passed, len(ts.Results), 
		float64(passed)/float64(len(ts.Results))*100)
}

func RunLevel1Tests() {
	// Create test database
	testDB := filepath.Join(".", "test_registry.duckdb")
	defer os.Remove(testDB)
	
	// Initialize registry
	reg, err := registry.NewRegistry(testDB)
	if err != nil {
		log.Fatal("Failed to create registry:", err)
	}
	defer reg.Close()
	
	ctx := context.Background()
	suite := &TestSuite{Name: "Level 1: Registry Core Functions"}
	
	// Test 1: ID Generation Format
	suite.Run("ID Generation Format", func() error {
		// Test MOD ID generation
		entity := &registry.Entity{
			EntityType:   registry.EntityTypeMOD,
			Name:         "Test Module",
			Description:  "Test module for validation",
			Status:       registry.StatusActive,
			Author:       "Test Suite",
			Organization: "Strigoi Test",
		}
		
		registered, err := reg.RegisterEntity(ctx, entity)
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}
		
		// Validate format: MOD-YYYY-##### 
		if !strings.HasPrefix(registered.BaseID, "MOD-2025-") {
			return fmt.Errorf("invalid MOD ID format: %s", registered.BaseID)
		}
		
		// Validate full entity ID includes version
		if !strings.HasSuffix(registered.EntityID, "-v1.0.0") {
			return fmt.Errorf("entity ID missing version: %s", registered.EntityID)
		}
		
		return nil
	})
	
	// Test 2: ID Generation Uniqueness
	suite.Run("ID Generation Uniqueness", func() error {
		ids := make(map[string]bool)
		
		// Generate 10 MOD IDs rapidly
		for i := 0; i < 10; i++ {
			entity := &registry.Entity{
				EntityType:   registry.EntityTypeMOD,
				Name:         fmt.Sprintf("Module %d", i),
				Status:       registry.StatusActive,
				Author:       "Test Suite",
				Organization: "Strigoi Test",
			}
			
			registered, err := reg.RegisterEntity(ctx, entity)
			if err != nil {
				return fmt.Errorf("registration %d failed: %w", i, err)
			}
			
			if ids[registered.BaseID] {
				return fmt.Errorf("duplicate ID generated: %s", registered.BaseID)
			}
			ids[registered.BaseID] = true
		}
		
		return nil
	})
	
	// Test 3: Entity Types ID Ranges
	suite.Run("Entity Type ID Ranges", func() error {
		// Test different entity types
		types := []registry.EntityType{
			registry.EntityTypeVUL,
			registry.EntityTypeATK,
			registry.EntityTypeSIG,
			registry.EntityTypeCFG,
		}
		
		for _, entityType := range types {
			entity := &registry.Entity{
				EntityType:   entityType,
				Name:         fmt.Sprintf("Test %s", entityType),
				Status:       registry.StatusActive,
				Author:       "Test Suite",
				Organization: "Strigoi Test",
			}
			
			registered, err := reg.RegisterEntity(ctx, entity)
			if err != nil {
				return fmt.Errorf("%s registration failed: %w", entityType, err)
			}
			
			// Verify correct prefix
			expectedPrefix := fmt.Sprintf("%s-2025-", entityType)
			if !strings.HasPrefix(registered.BaseID, expectedPrefix) {
				return fmt.Errorf("invalid %s ID format: %s", entityType, registered.BaseID)
			}
		}
		
		return nil
	})
	
	// Test 4: Basic CRUD - Create/Read
	suite.Run("CRUD Operations - Create/Read", func() error {
		// Create entity
		entity := &registry.Entity{
			EntityType:   registry.EntityTypeMOD,
			Name:         "CRUD Test Module",
			Description:  "Module for testing CRUD operations",
			Status:       registry.StatusActive,
			Severity:     registry.SeverityHigh,
			Author:       "CRUD Tester",
			Organization: "Test Org",
			Category:     "test",
			Tags:         []string{"test", "crud", "validation"},
		}
		
		created, err := reg.RegisterEntity(ctx, entity)
		if err != nil {
			return fmt.Errorf("create failed: %w", err)
		}
		
		// Read entity
		retrieved, err := reg.GetEntity(ctx, created.EntityID)
		if err != nil {
			return fmt.Errorf("read failed: %w", err)
		}
		
		// Validate fields match
		if retrieved.Name != created.Name {
			return fmt.Errorf("name mismatch: got %s, want %s", retrieved.Name, created.Name)
		}
		if retrieved.Description != created.Description {
			return fmt.Errorf("description mismatch")
		}
		if len(retrieved.Tags) != len(created.Tags) {
			return fmt.Errorf("tags mismatch: got %d tags, want %d", len(retrieved.Tags), len(created.Tags))
		}
		
		return nil
	})
	
	// Test 5: CRUD - Update
	suite.Run("CRUD Operations - Update", func() error {
		// Create entity
		entity := &registry.Entity{
			EntityType:   registry.EntityTypeVUL,
			Name:         "Update Test Vuln",
			Description:  "Original description",
			Status:       registry.StatusDraft,
			Author:       "Original Author",
			Organization: "Test Org",
		}
		
		created, err := reg.RegisterEntity(ctx, entity)
		if err != nil {
			return fmt.Errorf("create failed: %w", err)
		}
		
		// Update entity
		created.Description = "Updated description"
		created.Status = registry.StatusActive
		created.Severity = registry.SeverityCritical
		
		err = reg.UpdateEntity(ctx, created, "update", "Testing update functionality", "Test Suite")
		if err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		
		// Verify update
		updated, err := reg.GetEntity(ctx, created.EntityID)
		if err != nil {
			return fmt.Errorf("read after update failed: %w", err)
		}
		
		if updated.Description != "Updated description" {
			return fmt.Errorf("description not updated")
		}
		if updated.Status != registry.StatusActive {
			return fmt.Errorf("status not updated")
		}
		if updated.Severity != registry.SeverityCritical {
			return fmt.Errorf("severity not updated")
		}
		
		return nil
	})
	
	// Test 6: Metadata and Configuration Storage
	suite.Run("Metadata and Configuration Storage", func() error {
		// Create entity with complex metadata
		entity := &registry.Entity{
			EntityType:   registry.EntityTypeMOD,
			Name:         "Metadata Test",
			Status:       registry.StatusActive,
			Author:       "Test Suite",
			Organization: "Test Org",
			Metadata: map[string]interface{}{
				"version":     "1.2.3",
				"requirements": []string{"python3", "nmap"},
				"risk_score":  8.5,
				"tested":      true,
			},
			Configuration: map[string]interface{}{
				"timeout": 30,
				"retries": 3,
				"options": map[string]interface{}{
					"verbose": true,
					"threads": 4,
				},
			},
		}
		
		created, err := reg.RegisterEntity(ctx, entity)
		if err != nil {
			return fmt.Errorf("create with metadata failed: %w", err)
		}
		
		// Retrieve and verify
		retrieved, err := reg.GetEntity(ctx, created.EntityID)
		if err != nil {
			return fmt.Errorf("retrieve failed: %w", err)
		}
		
		// Check metadata
		if retrieved.Metadata == nil {
			return fmt.Errorf("metadata is nil")
		}
		if retrieved.Metadata["version"] != "1.2.3" {
			return fmt.Errorf("metadata version mismatch")
		}
		if retrieved.Metadata["risk_score"].(float64) != 8.5 {
			return fmt.Errorf("metadata risk_score mismatch")
		}
		
		// Check configuration
		if retrieved.Configuration == nil {
			return fmt.Errorf("configuration is nil")
		}
		if int(retrieved.Configuration["timeout"].(float64)) != 30 {
			return fmt.Errorf("configuration timeout mismatch")
		}
		
		return nil
	})
	
	// Test 7: Timestamp Management
	suite.Run("Timestamp Management", func() error {
		now := time.Now()
		discoveryDate := now.Add(-24 * time.Hour)
		analysisDate := now.Add(-12 * time.Hour)
		
		entity := &registry.Entity{
			EntityType:     registry.EntityTypeVUL,
			Name:           "Timestamp Test",
			Status:         registry.StatusActive,
			Author:         "Test Suite",
			Organization:   "Test Org",
			DiscoveryDate:  &discoveryDate,
			AnalysisDate:   &analysisDate,
		}
		
		created, err := reg.RegisterEntity(ctx, entity)
		if err != nil {
			return fmt.Errorf("create failed: %w", err)
		}
		
		// Verify timestamps
		if created.CreatedAt.IsZero() {
			return fmt.Errorf("created_at not set")
		}
		if created.UpdatedAt.IsZero() {
			return fmt.Errorf("updated_at not set")
		}
		
		retrieved, err := reg.GetEntity(ctx, created.EntityID)
		if err != nil {
			return fmt.Errorf("retrieve failed: %w", err)
		}
		
		if retrieved.DiscoveryDate == nil || retrieved.DiscoveryDate.IsZero() {
			return fmt.Errorf("discovery_date not preserved")
		}
		if retrieved.AnalysisDate == nil || retrieved.AnalysisDate.IsZero() {
			return fmt.Errorf("analysis_date not preserved")
		}
		
		return nil
	})
	
	// Test 8: Status Transitions
	suite.Run("Status Transitions", func() error {
		statuses := []registry.EntityStatus{
			registry.StatusDraft,
			registry.StatusTesting,
			registry.StatusActive,
			registry.StatusDeprecated,
			registry.StatusArchived,
		}
		
		entity := &registry.Entity{
			EntityType:   registry.EntityTypeMOD,
			Name:         "Status Test",
			Status:       registry.StatusDraft,
			Author:       "Test Suite",
			Organization: "Test Org",
		}
		
		created, err := reg.RegisterEntity(ctx, entity)
		if err != nil {
			return fmt.Errorf("create failed: %w", err)
		}
		
		// Test each status transition
		for _, status := range statuses[1:] { // Skip draft as it's the initial status
			created.Status = status
			err = reg.UpdateEntity(ctx, created, "status_change", 
				fmt.Sprintf("Changed to %s", status), "Test Suite")
			if err != nil {
				return fmt.Errorf("update to %s failed: %w", status, err)
			}
			
			// Verify
			retrieved, err := reg.GetEntity(ctx, created.EntityID)
			if err != nil {
				return fmt.Errorf("retrieve after %s update failed: %w", status, err)
			}
			
			if retrieved.Status != status {
				return fmt.Errorf("status not updated to %s, got %s", status, retrieved.Status)
			}
			
			// For archived status, set archived_at
			if status == registry.StatusArchived && retrieved.ArchivedAt == nil {
				now := time.Now()
				created.ArchivedAt = &now
				reg.UpdateEntity(ctx, created, "archive", "Archiving entity", "Test Suite")
			}
		}
		
		return nil
	})
	
	// Generate report
	suite.Report()
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "level2" {
		RunLevel2Tests()
	} else if len(os.Args) > 1 && os.Args[1] == "level1" {
		RunLevel1Tests()
	} else {
		// Run all tests
		fmt.Println("🧪 STRIGOI REGISTRY TEST SUITE")
		fmt.Println("==============================\n")
		RunLevel1Tests()
		fmt.Println()
		RunLevel2Tests()
	}
}