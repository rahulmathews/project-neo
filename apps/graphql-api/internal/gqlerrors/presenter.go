// Package gqlerrors maps resolver errors onto stable client-facing GraphQL
// error codes, masking everything unexpected as an internal error so raw
// database/internal error text never reaches clients.
package gqlerrors

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"

	"project-neo/graphql-api/internal/auth"
	"project-neo/graphql-api/internal/httpx"
	"project-neo/graphql-api/internal/validation"
	"project-neo/shared/repository"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const codeInternal = "INTERNAL"

// Presenter returns the gqlgen error presenter. Known-safe errors (repository
// sentinels, validation failures, auth) keep their message and gain a stable
// extensions.code; anything else is logged server-side with its request id
// and replaced by a generic internal error.
func Presenter(logger *slog.Logger) graphql.ErrorPresenterFunc {
	return func(ctx context.Context, err error) *gqlerror.Error {
		presented := graphql.DefaultErrorPresenter(ctx, err)

		if code, ok := knownCode(err); ok {
			setCode(presented, code)
			return presented
		}

		// Errors gqlgen raises itself (query parse/validation failures,
		// complexity rejections) wrap no resolver error and are client-safe.
		if presented.Unwrap() == nil {
			return presented
		}

		logger.Error(
			"graphql internal error",
			"error", err,
			"request_id", httpx.RequestIDFromCtx(ctx),
			"operation", operationName(ctx),
			"path", presented.Path.String(),
		)

		masked := &gqlerror.Error{
			Message: "internal server error",
			Path:    presented.Path,
		}
		setCode(masked, codeInternal)
		return masked
	}
}

// RecoverFunc logs resolver panics through slog (gqlgen's default prints to
// stderr, bypassing structured logging) and yields a masked error.
func RecoverFunc(logger *slog.Logger) graphql.RecoverFunc {
	return func(ctx context.Context, p interface{}) error {
		logger.Error(
			"panic in resolver",
			"panic", p,
			"request_id", httpx.RequestIDFromCtx(ctx),
			"operation", operationName(ctx),
			"stack", string(debug.Stack()),
		)
		return gqlerror.Errorf("internal server error")
	}
}

func knownCode(err error) (string, bool) {
	var fieldErrs validation.Errors
	var fieldErr validation.FieldError
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return "NOT_FOUND", true
	case errors.Is(err, repository.ErrForbidden):
		return "FORBIDDEN", true
	case errors.Is(err, repository.ErrConflict):
		return "CONFLICT", true
	case errors.Is(err, repository.ErrInvalidState):
		return "INVALID_STATE", true
	case errors.Is(err, auth.ErrUnauthenticated):
		return "UNAUTHENTICATED", true
	case errors.As(err, &fieldErrs), errors.As(err, &fieldErr):
		return "BAD_USER_INPUT", true
	}
	return "", false
}

func setCode(e *gqlerror.Error, code string) {
	if e.Extensions == nil {
		e.Extensions = map[string]interface{}{}
	}
	e.Extensions["code"] = code
}

func operationName(ctx context.Context) string {
	if !graphql.HasOperationContext(ctx) {
		return ""
	}
	return graphql.GetOperationContext(ctx).OperationName
}
