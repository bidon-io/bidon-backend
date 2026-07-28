package rendering

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBoolPointer(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer", func(t *testing.T) {
		t.Parallel()
		var p *bool
		assert.Nil(t, validateBoolPointer(reflect.ValueOf(p)))
	})

	t.Run("non-nil true", func(t *testing.T) {
		t.Parallel()
		v := true
		assert.Equal(t, true, validateBoolPointer(reflect.ValueOf(&v)))
	})

	t.Run("non-nil false", func(t *testing.T) {
		t.Parallel()
		v := false
		assert.Equal(t, false, validateBoolPointer(reflect.ValueOf(&v)))
	})

	t.Run("non-pointer falls through", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "x", validateBoolPointer(reflect.ValueOf("x")))
	})
}

func TestValidators_emptyValuesPass(t *testing.T) {
	t.Parallel()

	// Empty values are allowed by the validators themselves (omitempty on tags is separate).
	require.NoError(t, validate.Var("", "hexcolor"))
	require.NoError(t, validate.Var("", "httpurl"))
}

func TestValidateCustomAssetURL_withoutStructParent(t *testing.T) {
	t.Parallel()

	// When the field is validated outside a CloseButtonConfig parent, only URL shape matters.
	require.NoError(t, validate.Var("https://cdn.example.com/x.png", "custom_asset_url"))
	require.Error(t, validate.Var("not-a-url", "custom_asset_url"))
	require.NoError(t, validate.Var("", "custom_asset_url"))
}

func TestValidateHexColor_viaVar(t *testing.T) {
	t.Parallel()

	require.NoError(t, validate.Var("#abc", "hexcolor"))
	require.NoError(t, validate.Var("#AABBCC", "hexcolor"))
	require.Error(t, validate.Var("abc", "hexcolor"))     // wrong length
	require.Error(t, validate.Var("FFFF", "hexcolor"))    // length ok, missing '#'
	require.Error(t, validate.Var("FFFFFFF", "hexcolor")) // length ok, missing '#'
	require.Error(t, validate.Var("#abcd", "hexcolor"))   // wrong length
	require.Error(t, validate.Var("#gggggg", "hexcolor")) // invalid chars
}
