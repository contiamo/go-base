package authorization

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	chttp "github.com/contiamo/go-base/v4/pkg/http"
	"github.com/contiamo/go-base/v4/pkg/tracing"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewMiddleware creates a new authrorization middleware to set the claims in the context
func NewMiddleware(headerName string, publicKey interface{}) chttp.Option {
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
	parser := new(jwt.Parser)
	// To only allow specific singing methods uncomment below
	// *Note* that jwt.SigningMethodNone will only be accepted if`publichKey`
	// is set to `jwt.UnsafeAllowNoneSignatureType`, otherwise the jwt package
	// will reject the token. in particular, we want to disallow the None method:
	// parser.ValidMethods = []string{jwt.SigningMethodRS512.Alg()}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		span, ctx := a.StartSpan(r.Context(), "HandlerFunc")
		defer func() {
			if err != nil {
				// don't set the flag to `true` it would expose all
				// the data sent via the request body
				prettyRequest, _ := httputil.DumpRequest(r, false)
				logrus.WithError(err).
					WithField("request", string(prettyRequest)).
					Debugf("incoming request")
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

		span.SetTag("header.value", sanitizeHeaderValue(headerValue))

		// If auth fails or there was an error, do not call next.

		// will parse claims into a jwt.MapClaims and run the default validation
		// this will verify exp, iat, and nbf (if they exist)
		token, err := parser.Parse(headerValue, getKeyFunction(a.publicKey))
		if err != nil {
			err = errors.Wrap(err, "could not parse and verify auth claims")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		claims, err := parseClaims(token)
		if err != nil {
			err = errors.Wrap(err, "could not parse request token from claims")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		err = claims.Validate()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		claims.SourceToken = headerValue

		r = SetClaims(r, claims)
		next.ServeHTTP(w, r)
	})
}

// sanitizeHeaderValue partially scrubs the header value so that the full value
// is not logged or reusable.
func sanitizeHeaderValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return trimmed
	}
	// need at least enough space for the JWT header
	parts := strings.Split(trimmed, ".")
	header := parts[0]

	// the encoded length for a JWT will be at least 36 characters, if the header
	// isn't this long, then it probably wasn't a real JWT and we use the default
	// behavior
	if len(header) >= 36 && len(parts) == 3 {
		return header + ".****"
	}

	// return half + 4 stars
	return trimmed[0:len(trimmed)/2] + "****"
}

func parseClaims(token *jwt.Token) (claims Claims, err error) {
	rawClaims, err := json.Marshal(token.Claims)
	if err != nil {
		return claims, err
	}

	err = json.Unmarshal(rawClaims, &claims)
	if err != nil {
		return claims, err
	}

	return claims, err
}

func getKeyFunction(key interface{}) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return key, nil
	}
}
