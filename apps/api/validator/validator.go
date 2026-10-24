package validator

import (
	"errors"
	"fmt"
	"api-kms/enums"
	"strings"

	"api-kms/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var messages = map[string]string{
	"required":      "{field} wajib diisi",
	"email":         "{field} harus berupa email yang valid",
	"max":           "{field} maksimal {param} karakter",
	"gte":           "{field} harus lebih besar atau sama dengan {param}",
	"unique_column": "{field} {value} already exists",
}

func formatField(field string) string {
	return strings.ToLower(field)
}

func translateError(fieldError validator.FieldError) string {
	msg, ok := messages[fieldError.Tag()]

	fmt.Println(fieldError.Tag())
	fmt.Println(fieldError.ActualTag())
	if !ok {
		return fmt.Sprintf("%s tidak valid", fieldError.Field())
	}

	msg = strings.ReplaceAll(msg, "{field}", fieldError.Field())
	msg = strings.ReplaceAll(msg, "{param}", fieldError.Param())
	msg = strings.ReplaceAll(msg, "{value}", fmt.Sprintf("%v", fieldError.Value()))

	return formatField(msg)
}

func BindAndValidate(c *gin.Context, obj any) error {
	if err := c.ShouldBind(obj); err != nil {
		var validatorError validator.ValidationErrors

		if errors.As(err, &validatorError) {
			errorMap := make(map[string][]string)
			for _, fe := range validatorError {
				field := formatField(fe.Field())
				msg := translateError(fe)
				errorMap[field] = append(errorMap[field], msg)
			}

			return utils.NewHttpError("Validation error", enums.ERR_BAD_REQUEST, &errorMap)
		}

		return utils.NewHttpError(err.Error(), enums.ERR_BAD_REQUEST, nil)
	}

	return nil
}
