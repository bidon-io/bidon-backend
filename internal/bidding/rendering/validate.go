package rendering

import (
	"net/url"
	"reflect"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

func init() {
	if err := validate.RegisterValidation("hexcolor", validateHexColor); err != nil {
		panic(err)
	}
	if err := validate.RegisterValidation("httpurl", validateHTTPURL); err != nil {
		panic(err)
	}
	if err := validate.RegisterValidation("custom_asset_url", validateCustomAssetURL); err != nil {
		panic(err)
	}
	validate.RegisterCustomTypeFunc(validateBoolPointer, reflect.TypeOf((*bool)(nil)))
}

// Validate checks rendering config against struct validation tags.
func Validate(cfg *Config) error {
	if cfg == nil {
		return nil
	}
	return validate.Struct(cfg)
}

func validateHexColor(fl validator.FieldLevel) bool {
	color := fl.Field().String()
	if color == "" {
		return true
	}
	if len(color) != 4 && len(color) != 7 {
		return false
	}
	if color[0] != '#' {
		return false
	}
	for _, r := range color[1:] {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func validateHTTPURL(fl validator.FieldLevel) bool {
	raw := fl.Field().String()
	if raw == "" {
		return true
	}
	return isHTTPURL(raw)
}

func validateCustomAssetURL(fl validator.FieldLevel) bool {
	raw := fl.Field().String()
	parent := fl.Parent()
	if !parent.IsValid() || parent.Kind() != reflect.Struct {
		return raw == "" || isHTTPURL(raw)
	}
	style := parent.FieldByName("Style")
	if !style.IsValid() || CloseButtonStyle(style.String()) != CloseButtonStyleCustom {
		return raw == "" || isHTTPURL(raw)
	}
	return isHTTPURL(raw)
}

func isHTTPURL(raw string) bool {
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func validateBoolPointer(fieldValue reflect.Value) any {
	if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Bool {
		if fieldValue.IsNil() {
			return nil
		}
		return fieldValue.Elem().Bool()
	}
	return fieldValue.Interface()
}
