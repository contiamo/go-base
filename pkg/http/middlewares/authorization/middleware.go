package authorization

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/contiamo/go-base/pkg/tracing"
	goserverhttp "github.com/contiamo/goserver/http"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewMiddleware creates a new authrorization middleware to set the claims in the context
func NewMiddleware(headerName string, publicKey interface{}) goserverhttp.Option {
	return &middleware{
		Tracer:     tracing.NewTracer("authorization", "middleware"),
		headerName: headerName,
		publicKey:  publicKey,
	}
}

type middleware struct {
	tracing.Tracer
	headerName string
	publicKey  interface{}
}

func (a *middleware) WrapHandler(next http.Handler) http.Handler {
	if a.headerName == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		span, ctx := a.StartSpan(r.Context(), "HandlerFunc")
		defer func() {
			if err != nil {
				// don't set the flag to `true` it would expose all
				// the data sent via the request body
				prettyRequest, _ := httputil.DumpRequest(r, false)
				logrus.Debugf("Incoming request:\\n%q", prettyRequest)
			}
			a.FinishSpan(span, err)
		}()

		r = r.WithContext(ctx)

		//skip middleware on OPTIONS requests
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		span.SetTag("header.name", a.headerName)

		headerValue := r.Header.Get(a.headerName)
		if headerValue == "" {
			err = errors.New("could not find the authentication header")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		span.SetTag("header.value", headerValue)

		// If auth fails or there was an error, do not call next.
		token, err := jwt.Parse(headerValue, getKeyFunction(a.publicKey))
		if err != nil {
			err = errors.Wrap(err, "could not parse and verify auth claims")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		claims, err := parseClaims(token.Raw)
		if err != nil {
			err = errors.Wrap(err, "could not parse request token from claims")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if !claims.Valid() {
			err = errors.New("auth claims missing required user information")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		claims.SourceToken = headerValue

		r = SetClaims(r, claims)
		next.ServeHTTP(w, r)
	})
}

func parseClaims(token string) (claims Claims, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, errors.New("token contains an invalid number of segments")
	}

	rawClaims, err := jwt.DecodeSegment(parts[1])
	if err != nil {
		return claims, err
	}

	claimMap := make(map[string]interface{})
	err = json.Unmarshal(rawClaims, &claimMap)
	if err != nil {
		return claims, err
	}

	err = claims.FromClaimsMap(claimMap)
	return claims, err
}

func getKeyFunction(key interface{}) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return key, nil
	}
}
