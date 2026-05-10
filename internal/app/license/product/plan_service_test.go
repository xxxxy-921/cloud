package product

import (
	"encoding/json"
	"errors"
	"metis/internal/app/license/domain"
	"metis/internal/app/license/testutil"
	"testing"

	"metis/internal/database"
	"gorm.io/gorm"
)

func newPlanService(db *database.DB) *PlanService {
	return &PlanService{
		planRepo:    &PlanRepo{DB: db},
		productRepo: &ProductRepo{DB: db},
	}
}

func TestPlanService_CreatePlan(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	planSvc := newPlanService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-plan", "")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Update product with a constraint schema
	schema := json.RawMessage(`[{"key":"core","label":"Core","features":[{"key":"seats","type":"number","min":1,"max":100}]}]`)
	if err := productSvc.UpdateConstraintSchema(product.ID, schema); err != nil {
		t.Fatalf("failed to set schema: %v", err)
	}

	t.Run("valid plan without constraints", func(t *testing.T) {
		plan, err := planSvc.CreatePlan(product.ID, "Basic", nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Name != "Basic" {
			t.Errorf("Name = %q, want %q", plan.Name, "Basic")
		}
	})

	t.Run("valid plan with matching constraints", func(t *testing.T) {
		values := json.RawMessage(`{"core":{"seats":10}}`)
		plan, err := planSvc.CreatePlan(product.ID, "Pro", values, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Name != "Pro" {
			t.Errorf("Name = %q, want %q", plan.Name, "Pro")
		}
	})

	t.Run("duplicate name", func(t *testing.T) {
		_, err := planSvc.CreatePlan(product.ID, "Basic", nil, 0)
		if !errors.Is(err, ErrPlanNameExists) {
			t.Errorf("expected ErrPlanNameExists, got %v", err)
		}
	})

	t.Run("invalid constraint values", func(t *testing.T) {
		values := json.RawMessage(`{"core":{"seats":999}}`)
		_, err := planSvc.CreatePlan(product.ID, "Enterprise", values, 0)
		if !errors.Is(err, ErrInvalidConstraintValues) {
			t.Errorf("expected ErrInvalidConstraintValues, got %v", err)
		}
	})
}

func TestPlanService_SetDefaultPlan(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	planSvc := newPlanService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-default", "")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	plan1, err := planSvc.CreatePlan(product.ID, "domain.Plan 1", nil, 0)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	plan2, err := planSvc.CreatePlan(product.ID, "domain.Plan 2", nil, 0)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Set plan1 as default
	if err := planSvc.SetDefaultPlan(plan1.ID, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Now set plan2 as default; plan1 should no longer be default
	if err := planSvc.SetDefaultPlan(plan2.ID, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int64
	db.Model(&domain.Plan{}).Where("product_id = ? AND is_default = ?", product.ID, true).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 default plan, got %d", count)
	}

	// Verify plan2 is the default
	var p2 domain.Plan
	db.First(&p2, plan2.ID)
	if !p2.IsDefault {
		t.Error("expected plan2 to be default")
	}

	var p1 domain.Plan
	db.First(&p1, plan1.ID)
	if p1.IsDefault {
		t.Error("expected plan1 to no longer be default")
	}

	t.Run("clear default leaves no default plan", func(t *testing.T) {
		if err := planSvc.SetDefaultPlan(plan2.ID, false); err != nil {
			t.Fatalf("clear default plan: %v", err)
		}

		var cleared domain.Plan
		if err := db.First(&cleared, plan2.ID).Error; err != nil {
			t.Fatalf("reload cleared default plan: %v", err)
		}
		if cleared.IsDefault {
			t.Fatalf("expected plan2 default flag cleared, got %+v", cleared)
		}

		var count int64
		if err := db.Model(&domain.Plan{}).Where("product_id = ? AND is_default = ?", product.ID, true).Count(&count).Error; err != nil {
			t.Fatalf("count default plans after clear: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected no default plans after clearing, got %d", count)
		}
	})
}

func TestPlanService_UpdatePlan(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	planSvc := newPlanService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-update", "")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	schema := json.RawMessage(`[{"key":"core","features":[{"key":"seats","type":"number","min":1,"max":100}]}]`)
	if err := productSvc.UpdateConstraintSchema(product.ID, schema); err != nil {
		t.Fatalf("failed to set schema: %v", err)
	}

	plan, err := planSvc.CreatePlan(product.ID, "Starter", nil, 0)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	t.Run("rename plan", func(t *testing.T) {
		newName := "Starter Plus"
		updated, err := planSvc.UpdatePlan(plan.ID, &newName, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Name != newName {
			t.Errorf("Name = %q, want %q", updated.Name, newName)
		}
	})

	t.Run("update with invalid constraints", func(t *testing.T) {
		values := json.RawMessage(`{"core":{"seats":0}}`)
		_, err := planSvc.UpdatePlan(plan.ID, nil, values, nil)
		if !errors.Is(err, ErrInvalidConstraintValues) {
			t.Errorf("expected ErrInvalidConstraintValues, got %v", err)
		}
	})
}

func TestPlanService_GuardsAndTypedConstraints(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	planSvc := newPlanService(db)

	if _, err := planSvc.CreatePlan(999, "ghost", nil, 0); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("CreatePlan missing product error = %v, want %v", err, ErrProductNotFound)
	}

	product, err := productSvc.CreateProduct("Typed Constraints", "prod-typed-constraints", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	schema := json.RawMessage(`[
		{"key":"core","features":[
			{"key":"seats","type":"number","min":1,"max":10},
			{"key":"edition","type":"enum","options":["basic","pro"]},
			{"key":"addons","type":"multiSelect","options":["backup","audit"]}
		]}
	]`)
	if err := productSvc.UpdateConstraintSchema(product.ID, schema); err != nil {
		t.Fatalf("update schema: %v", err)
	}

	validValues := json.RawMessage(`{"core":{"seats":5,"edition":"pro","addons":["backup","audit"]}}`)
	plan, err := planSvc.CreatePlan(product.ID, "Growth", validValues, 1)
	if err != nil {
		t.Fatalf("CreatePlan typed constraints: %v", err)
	}

	other, err := planSvc.CreatePlan(product.ID, "Enterprise", nil, 2)
	if err != nil {
		t.Fatalf("CreatePlan second plan: %v", err)
	}

	t.Run("duplicate rename is rejected", func(t *testing.T) {
		name := other.Name
		if _, err := planSvc.UpdatePlan(plan.ID, &name, nil, nil); !errors.Is(err, ErrPlanNameExists) {
			t.Fatalf("UpdatePlan duplicate rename error = %v, want %v", err, ErrPlanNameExists)
		}
	})

	t.Run("enum and multiselect validation stays enforced on update", func(t *testing.T) {
		badEnum := json.RawMessage(`{"core":{"seats":5,"edition":"enterprise"}}`)
		if _, err := planSvc.UpdatePlan(plan.ID, nil, badEnum, nil); !errors.Is(err, ErrInvalidConstraintValues) {
			t.Fatalf("UpdatePlan bad enum error = %v, want %v", err, ErrInvalidConstraintValues)
		}

		badMultiSelect := json.RawMessage(`{"core":{"seats":5,"edition":"basic","addons":[1]}}`)
		if _, err := planSvc.UpdatePlan(plan.ID, nil, badMultiSelect, nil); !errors.Is(err, ErrInvalidConstraintValues) {
			t.Fatalf("UpdatePlan bad multiselect error = %v, want %v", err, ErrInvalidConstraintValues)
		}

		goodUpdate := json.RawMessage(`{"core":{"seats":7,"edition":"basic","addons":["audit"]}}`)
		updated, err := planSvc.UpdatePlan(plan.ID, nil, goodUpdate, nil)
		if err != nil {
			t.Fatalf("UpdatePlan good typed constraints: %v", err)
		}
		if string(updated.ConstraintValues) != string(goodUpdate) {
			t.Fatalf("ConstraintValues = %s, want %s", string(updated.ConstraintValues), string(goodUpdate))
		}
	})

	t.Run("missing plan operations are rejected", func(t *testing.T) {
		if _, err := planSvc.UpdatePlan(999, nil, nil, nil); !errors.Is(err, ErrPlanNotFound) {
			t.Fatalf("UpdatePlan missing plan error = %v, want %v", err, ErrPlanNotFound)
		}
		if err := planSvc.SetDefaultPlan(999, true); !errors.Is(err, ErrPlanNotFound) {
			t.Fatalf("SetDefaultPlan missing plan error = %v, want %v", err, ErrPlanNotFound)
		}
	})

	t.Run("delete removes plan and second delete is rejected", func(t *testing.T) {
		deletable, err := planSvc.CreatePlan(product.ID, "Delete Me", nil, 3)
		if err != nil {
			t.Fatalf("CreatePlan deletable: %v", err)
		}

		if err := planSvc.DeletePlan(deletable.ID); err != nil {
			t.Fatalf("DeletePlan: %v", err)
		}

		if _, err := planSvc.planRepo.FindByID(deletable.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("expected deleted plan missing from repo, got %v", err)
		}

		if err := planSvc.DeletePlan(deletable.ID); !errors.Is(err, ErrPlanNotFound) {
			t.Fatalf("DeletePlan missing plan error = %v, want %v", err, ErrPlanNotFound)
		}
	})

	t.Run("blank plan names are rejected and trimmed on valid updates", func(t *testing.T) {
		if _, err := planSvc.CreatePlan(product.ID, "   ", nil, 4); !errors.Is(err, ErrInvalidPlanName) {
			t.Fatalf("CreatePlan blank name error = %v, want %v", err, ErrInvalidPlanName)
		}

		trimmed := "  Growth Plus  "
		updated, err := planSvc.UpdatePlan(plan.ID, &trimmed, nil, nil)
		if err != nil {
			t.Fatalf("UpdatePlan trimmed name: %v", err)
		}
		if updated.Name != "Growth Plus" {
			t.Fatalf("expected trimmed plan name, got %+v", updated)
		}

		blank := "   "
		if _, err := planSvc.UpdatePlan(plan.ID, &blank, nil, nil); !errors.Is(err, ErrInvalidPlanName) {
			t.Fatalf("UpdatePlan blank name error = %v, want %v", err, ErrInvalidPlanName)
		}
	})

	t.Run("set default toggles ownership across plans", func(t *testing.T) {
		first, err := planSvc.CreatePlan(product.ID, "Starter Default", nil, 5)
		if err != nil {
			t.Fatalf("CreatePlan first default candidate: %v", err)
		}
		second, err := planSvc.CreatePlan(product.ID, "Second Default", nil, 6)
		if err != nil {
			t.Fatalf("CreatePlan second default candidate: %v", err)
		}

		if err := planSvc.SetDefaultPlan(first.ID, true); err != nil {
			t.Fatalf("SetDefaultPlan first: %v", err)
		}
		if err := planSvc.SetDefaultPlan(second.ID, true); err != nil {
			t.Fatalf("SetDefaultPlan second: %v", err)
		}

		var reloadedFirst, reloadedSecond domain.Plan
		if err := db.First(&reloadedFirst, first.ID).Error; err != nil {
			t.Fatalf("reload first plan: %v", err)
		}
		if err := db.First(&reloadedSecond, second.ID).Error; err != nil {
			t.Fatalf("reload second plan: %v", err)
		}
		if reloadedFirst.IsDefault {
			t.Fatalf("expected first plan default flag cleared, got %+v", reloadedFirst)
		}
		if !reloadedSecond.IsDefault {
			t.Fatalf("expected second plan to own default flag, got %+v", reloadedSecond)
		}
	})
}

func TestPlanService_AcceptsNumericConstraintValueShapes(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	planSvc := newPlanService(db)

	product, err := productSvc.CreateProduct("Numeric Shapes", "prod-numeric-shapes", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	schema := json.RawMessage(`[{"key":"core","features":[{"key":"seats","type":"number","min":1,"max":100}]}]`)
	if err := productSvc.UpdateConstraintSchema(product.ID, schema); err != nil {
		t.Fatalf("update schema: %v", err)
	}

	cases := []struct {
		name   string
		values json.RawMessage
	}{
		{name: "integer value", values: json.RawMessage(`{"core":{"seats":7}}`)},
		{name: "floating value", values: json.RawMessage(`{"core":{"seats":7.5}}`)},
		{name: "json number style", values: json.RawMessage(`{"core":{"seats":99}}`)},
	}

	for i, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			plan, err := planSvc.CreatePlan(product.ID, "Numeric Plan "+string(rune('A'+i)), tc.values, i)
			if err != nil {
				t.Fatalf("CreatePlan %s: %v", tc.name, err)
			}
			if string(plan.ConstraintValues) != string(tc.values) {
				t.Fatalf("ConstraintValues = %s, want %s", plan.ConstraintValues, tc.values)
			}
		})
	}
}

func TestConstraintValidationHelpers_HandleNumericAndUnknownKeys(t *testing.T) {
	schema := json.RawMessage(`[
		{"key":"core","features":[
			{"key":"seats","type":"number","min":1,"max":10},
			{"key":"edition","type":"enum","options":["basic","pro"]},
			{"key":"addons","type":"multiSelect","options":["backup","audit"]}
		]}
	]`)

	t.Run("unknown module and feature keys are rejected", func(t *testing.T) {
		if err := validateConstraintValues(schema, json.RawMessage(`{"ghost":{"seats":1}}`)); !errors.Is(err, ErrInvalidConstraintValues) {
			t.Fatalf("unknown module error = %v, want %v", err, ErrInvalidConstraintValues)
		}
		if err := validateConstraintValues(schema, json.RawMessage(`{"core":{"unknown":1}}`)); !errors.Is(err, ErrInvalidConstraintValues) {
			t.Fatalf("unknown feature error = %v, want %v", err, ErrInvalidConstraintValues)
		}
	})

	t.Run("module enabled toggle is ignored but numeric values remain validated", func(t *testing.T) {
		if err := validateConstraintValues(schema, json.RawMessage(`{"core":{"enabled":true,"seats":3}}`)); err != nil {
			t.Fatalf("enabled toggle should be ignored, got %v", err)
		}
		if err := validateConstraintValues(schema, json.RawMessage(`{"core":{"enabled":true,"seats":"3"}}`)); !errors.Is(err, ErrInvalidConstraintValues) {
			t.Fatalf("string numeric value error = %v, want %v", err, ErrInvalidConstraintValues)
		}
	})

	t.Run("toFloat64 handles supported numeric shapes and rejects others", func(t *testing.T) {
		cases := []struct {
			name  string
			value any
			want  float64
			ok    bool
		}{
			{name: "float64", value: float64(1.5), want: 1.5, ok: true},
			{name: "float32", value: float32(2.5), want: 2.5, ok: true},
			{name: "int", value: int(3), want: 3, ok: true},
			{name: "int64", value: int64(4), want: 4, ok: true},
			{name: "json number", value: json.Number("5.5"), want: 5.5, ok: true},
			{name: "invalid", value: "6", want: 0, ok: false},
		}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				got, ok := toFloat64(tc.value)
				if ok != tc.ok || got != tc.want {
					t.Fatalf("toFloat64(%#v) = (%v,%v), want (%v,%v)", tc.value, got, ok, tc.want, tc.ok)
				}
			})
		}
	})
}
