package admin

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/web"
)

func TestCategoryInputValidate(t *testing.T) {
	tests := []struct {
		name      string
		in        CategoryInput
		forCreate bool
		wantErr   bool
	}{
		{"valid create", CategoryInput{Code: "electronics_repair", Name: "Electronics"}, true, false},
		{"invalid code on create", CategoryInput{Code: "Bad Code!", Name: "Electronics"}, true, true},
		{"code not required on update", CategoryInput{Name: "Electronics"}, false, false},
		{"name too short", CategoryInput{Code: "ok", Name: "A"}, true, true},
		{"name too long", CategoryInput{Code: "ok", Name: string(make([]rune, 81))}, true, true},
		{"description too long", CategoryInput{Code: "ok", Name: "Electronics", Description: string(make([]rune, 501))}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.validate(tt.forCreate)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tt.in.NameTranslations, "validate must default NameTranslations")
			}
		})
	}
}

func TestCriterionInputValidate(t *testing.T) {
	tests := []struct {
		name      string
		in        CriterionInput
		forCreate bool
		wantErr   bool
	}{
		{"valid create with default scale", CriterionInput{Code: "repair_quality", Name: "Repair quality"}, true, false},
		{"invalid code on create", CriterionInput{Code: "!", Name: "Repair quality"}, true, true},
		{"name too short", CriterionInput{Code: "ok", Name: "A"}, true, true},
		{"explicit valid scale", CriterionInput{Code: "ok", Name: "Repair quality", ScaleMin: 0, ScaleMax: 10}, true, false},
		{"scale max not greater than min", CriterionInput{Code: "ok", Name: "Repair quality", ScaleMin: 3, ScaleMax: 3}, true, true},
		{"scale max exceeds 10", CriterionInput{Code: "ok", Name: "Repair quality", ScaleMin: 1, ScaleMax: 11}, true, true},
		{"negative scale min", CriterionInput{Code: "ok", Name: "Repair quality", ScaleMin: -1, ScaleMax: 5}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.validate(tt.forCreate)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tt.in.LabelTranslations, "validate must default LabelTranslations")
			}
		})
	}
}

func TestCriterionInputValidateDefaultsScaleWhenUnset(t *testing.T) {
	in := CriterionInput{Code: "ok", Name: "Repair quality"}
	require.NoError(t, in.validate(true))
	assert.Equal(t, 1, in.ScaleMin)
	assert.Equal(t, 5, in.ScaleMax)
}

func TestSuspendUserRejectsSelfSuspension(t *testing.T) {
	s := &Service{}
	id := uuid.New()

	err := s.SuspendUser(context.Background(), id, id, "note")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
}

func TestGrantRoleRejectsInvalidRole(t *testing.T) {
	s := &Service{}

	err := s.GrantRole(context.Background(), uuid.New(), uuid.New(), "superuser")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
}

func TestMergeTargetsRejectsSelfMerge(t *testing.T) {
	s := &Service{}
	id := uuid.New()

	err := s.MergeTargets(context.Background(), uuid.New(), id, id, "note")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
}
