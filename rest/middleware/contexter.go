package middleware

import (
	"net/http"

	"github.com/synapse-garden/sg-proto/auth"

	"golang.org/x/net/context"
)

type Contexter func(*http.Request, *auth.Context) *http.Request

func ByFields(fields ...auth.CtxField) Contexter {
	return func(r *http.Request, ctx *auth.Context) *http.Request {
		httpCtx := r.Context()
		for _, field := range fields {
			httpCtx = context.WithValue(httpCtx, field, ctx.ByField(field))
		}
		return r.WithContext(httpCtx)
	}
}

func CtxSetUserID(r *http.Request, ctx *auth.Context) *http.Request {
	return r.WithContext(context.WithValue(
		r.Context(), auth.CtxUserID, ctx.UserID,
	))
}

func CtxSetToken(r *http.Request, ctx *auth.Context) *http.Request {
	return r.WithContext(context.WithValue(
		r.Context(), auth.CtxToken, ctx.Token,
	))
}

func CtxSetRefreshToken(r *http.Request, ctx *auth.Context) *http.Request {
	return r.WithContext(context.WithValue(
		r.Context(), auth.CtxRefreshToken, ctx.RefreshToken,
	))
}

func CtxGetUserID(r *http.Request) string {
	return r.Context().Value(auth.CtxUserID).(string)
}

func CtxGetToken(r *http.Request) auth.Token {
	return r.Context().Value(auth.CtxToken).(auth.Token)
}

func CtxGetRefreshToken(r *http.Request) auth.Token {
	return r.Context().Value(auth.CtxRefreshToken).(auth.Token)
}
