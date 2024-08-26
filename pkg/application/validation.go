package application

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/bketelsen/incus-compose/pkg/incus/client"
	"github.com/go-playground/validator/v10"
)

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterValidation("project-exists", ProjectExists)
	validate.RegisterValidation("profile-exists", ProfileExists)
	validate.RegisterValidation("pool-exists", PoolExists)

}

// use a single instance of Validate, it caches struct info
var validate *validator.Validate

// ProjectExists implements validator.Func
func ProjectExists(fl validator.FieldLevel) bool {

	if fl.Field().String() != "" {
		client, err := client.NewIncusClient()
		if err != nil {
			slog.Error(err.Error())

			return false
		}
		pp, err := client.GetProjectNames()
		if err != nil {
			slog.Error(err.Error())

			return false
		}
		return slices.Contains(pp, fl.Field().String())
	}
	// if the project is empty, we assume it is valid
	return true

}

// ProfileExists implements validator.Func
func ProfileExists(fl validator.FieldLevel) bool {

	if fl.Field().String() != "" {
		client, err := client.NewIncusClient()
		if err != nil {
			slog.Error(err.Error())

			return false
		}
		pp, err := client.GetProfileNames()
		if err != nil {
			slog.Error(err.Error())

			return false
		}
		return slices.Contains(pp, fl.Field().String())
	}
	// if the project is empty, we assume it is valid
	return true

}

// PoolExists implements validator.Func
func PoolExists(fl validator.FieldLevel) bool {

	if fl.Field().String() != "" {
		client, err := client.NewIncusClient()
		if err != nil {
			slog.Error(err.Error())

			return false
		}
		pp, err := client.GetStoragePoolNames()
		if err != nil {
			slog.Error(err.Error())

			return false
		}
		return slices.Contains(pp, fl.Field().String())
	}
	// if the project is empty, we assume it is valid
	return true

}

func (app *Compose) Validate() error {

	// validate the struct
	err := validate.Struct(app)
	if err != nil {

		// this check is only needed when your code could produce
		// an invalid value for validation such as interface with nil
		// value most including myself do not usually have code like this.
		if _, ok := err.(*validator.InvalidValidationError); ok {
			slog.Error(err.Error())
			return err
		}

		for _, err := range err.(validator.ValidationErrors) {
			slog.Debug(err.Error())

			// fmt.Println("namespace", err.Namespace())
			// fmt.Println("field", err.Field())
			// fmt.Println("structNamespace", err.StructNamespace())
			// fmt.Println("struct field", err.StructField())
			// fmt.Println("tag", err.Tag())
			// fmt.Println("actual tag", err.ActualTag())
			// fmt.Println("kind", err.Kind())
			// fmt.Println("type", err.Type())
			// fmt.Println("value", err.Value())
			// fmt.Println("param", err.Param())
			// fmt.Println()

			switch err.Tag() {
			case "required":
				slog.Error("Field is required", slog.String("field", err.Field()))
			case "project-exists":
				slog.Error("Project does not exist", slog.String("project", err.Value().(string)))
			case "profile-exists":
				slog.Error("Profile does not exist", slog.String("profile", err.Value().(string)))
			case "pool-exists":
				slog.Error("Storage pool does not exist", slog.String("pool", err.Value().(string)))
			default:
				slog.Error("Validation error", slog.String("error", err.Error()))
			}

		}

		// from here you can create your own error messages in whatever language you wish
		return err
	}

	// custom validations
	for _, svc := range app.Services {
		if len(svc.DependsOn) > 0 {
			for _, dep := range svc.DependsOn {
				if _, ok := app.Services[dep]; !ok {
					slog.Error("Service does not exist", slog.String("service", dep))
					return fmt.Errorf("service dependency %s does not exist", dep)
				}
			}
		}
	}
	return nil
}
