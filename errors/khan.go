package errors

// this file is for compatibility with Webapp Khan errors.
// KhanWrap works like the Wrap there.

type errorKind string

// Error is a function that makes errorKind implement the error interface. This
// lets us use error.Is with kinds. We don't actually use the output of this
// function for anything.
func (e errorKind) Error() string {
	return string(e)
}

// String presents the value of the string, like "Not Found"
// The fmt package (and many others) look for this to print values.
func (e errorKind) String() string {
	return string(e)
}

const (
	// NotFoundKind means that some requested resource wasn't found. If the
	// resource couldn't be retrieved due to access control use
	// UnauthorizedKind instead. If the resource couldn't be found because
	// the input was invalid use InvalidInputKind instead.
	NotFoundKind errorKind = "not found"

	// InvalidInputKind means that there was a problem with the provided input.
	// This kind indicates inputs that are problematic regardless of the state
	// of the system. Use NotAllowedKind when the input is valid but
	// conflicts with the state of the system.
	InvalidInputKind errorKind = "invalid input error"

	// NotAllowedKind means that there was a problem due to the state of
	// the system not matching the requested operation or input. For
	// example, trying to create a username that is valid, but is already
	// taken by another user. Use InvalidInputKind when the input isn't
	// valid regardless of the state of the system. Use NotFoundKind when
	// the failure is due to not being able to find a resource.
	NotAllowedKind errorKind = "not allowed"

	// UnauthorizedKind means that there was an access control problem.
	UnauthorizedKind errorKind = "unauthorized error"

	// InternalKind means that the function failed for a reason unrelated
	// to its input or problems working with a remote system. Use this kind
	// when other error kinds aren't appropriate.
	InternalKind errorKind = "internal error"

	// NotImplementedKind means that the function isn't implemented.
	NotImplementedKind errorKind = "not implemented error"

	// GraphqlResponseKind means that the graphql server returned an
	// error code as part of the graphql response.  This kind of error
	// is only ever returned by gqlclient calls (e.g. Query or
	// ServiceAdminMutate).  It is set when the graphql call
	// successfully executes, but the graphql response struct
	// indicates the graphql request could not be executed due to an
	// error.  (e.g. mutation.MyMutation.Error.Code == "UNAUTHORIZED")
	GraphqlResponseKind errorKind = "graphql error response"

	// TransientKhanServiceKind means that there was a problem when contacting
	// another Khan service that might be resolvable by retrying.
	TransientKhanServiceKind errorKind = "transient khan service error"

	// KhanServiceKind means that there was a non-transient problem when
	// contacting another Khan service.
	KhanServiceKind errorKind = "khan service error"

	// TransientServiceKind means that there was a problem when making a
	// request to a non-Khan service, e.g. datastore that might be
	// resolvable by retrying.
	TransientServiceKind errorKind = "transient service error"

	// ServiceKind means that there was a non-transient problem when making a
	// request to a non-Khan service, e.g. datastore.
	ServiceKind errorKind = "service error"

	// UnspecifiedKind means that no error kind was specified. Note that there
	// isn't a constructor for this kind of error.
	UnspecifiedKind errorKind = "unspecified error"
)

// KhanWrap takes a khanError as input and some new field key/value pairs,
// and returns a new error that has the same "kind" as the existing
// error, plus the specified key/value pairs.  For convenience, rather
// than using errors.Fields{} to specify the key/value pairs, they
// are specified as alternating string/interface{} objects.
// Also for convenience, if nil is passed in then nil is returned.
//
// If there is an error in wrapping -- the input is not a khanError,
// a non-string key is specified -- then the wrapped error is actually
// an error.Internal() that indicates the problem with wrapping.
func KhanWrap(err error, args ...interface{}) error {
	if err == nil {
		return nil
	}

	if len(args)%2 != 0 {
		return newError(
			InternalKind,
			err,
			Fields{
				"fields":  args,
				"message": "Passed an odd number of field-args to errors.Wrap()",
			},
		)
	}

	fields := Fields{}
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			return newError(
				InternalKind,
				err,
				Fields{"key": args[i], "message": "Passed a non-string key-field to errors.Wrap()"},
			)
		}
		fields[key] = args[i+1]
	}

	khanKind, kindOfOk := getErrorKind(err)
	if kindOfOk { // root is errorKind
		if khanKind == UnspecifiedKind {
			// This probably can't happen, but just in case...
			return newError(InternalKind, args...)
		}
		return newError(khanKind, err, fields)
	}
	// "Internal" is the best default, but not always right.
	// e.g. for client.GCS() errors, "Service" would be better.
	// The solution is to change our GCS wrapper to return khanErrors,
	// like we do for our Datastore wrapper.
	return newError(InternalKind, err, fields)
}

func getErrorKind(err error) (errorKind, bool) {
	for tmpErr := err; tmpErr != nil; tmpErr = UnwrapOnce(tmpErr) {
		var khanKind errorKind
		if As(err, &khanKind) {
			return khanKind, true
		}
	}
	return UnspecifiedKind, false
}

// NotFound creates an error of kind NotFoundKind.  args can be
// (1) an error to wrap
// (2) a string to use as the error message
// (3) an errors.Fields{} object of key/value pairs to associate with the error
// (4) an errors.Source("source-location") to override the default source-loc
// If you specify any of these multiple times, only the last one wins.
func NotFound(args ...interface{}) error {
	return KhanWrap(NotFoundKind, args...)
}

// InvalidInput creates an error of kind InvalidKind.
func InvalidInput(args ...interface{}) error {
	return KhanWrap(InvalidInputKind, args...)
}

// NotAllowed creates an error of kind NotAllowedKind.
func NotAllowed(args ...interface{}) error {
	return KhanWrap(NotAllowedKind, args...)
}

// Unauthorized creates an error of kind UnauthorizedKind.
func Unauthorized(args ...interface{}) error {
	return KhanWrap(UnauthorizedKind, args...)
}

// Internal creates an error of kind InternalKind.
func Internal(args ...interface{}) error {
	return KhanWrap(InternalKind, args...)
}

// GraphqlResponse creates an error of kind GraphqlResponseKind.
func GraphqlResponse(args ...interface{}) error {
	return KhanWrap(GraphqlResponseKind, args...)
}

// NotImplemented creates an error of kind NotImplementedKind.
func NotImplemented(args ...interface{}) error {
	return KhanWrap(NotImplementedKind, args...)
}

// TransientKhanService creates an error of kind TransientKhanServiceKind.
func TransientKhanService(args ...interface{}) error {
	return KhanWrap(TransientKhanServiceKind, args...)
}

// KhanService creates an error of kind KhanServiceKind.
func KhanService(args ...interface{}) error {
	return KhanWrap(KhanServiceKind, args...)
}

// Service creates an error of kind ServiceKind.
func Service(args ...interface{}) error {
	return KhanWrap(ServiceKind, args...)
}

// TransientService creates an error of kind TransientServiceKind.
func TransientService(args ...interface{}) error {
	return KhanWrap(TransientServiceKind, args...)
}

func newError(kind errorKind, args ...interface{}) error {
	var message string
	var cause error
	var fields Fields
	for _, arg := range args {
		switch v := arg.(type) {
		case error:
			cause = v
		case string:
			message = v
		case Fields:
			fields = v
		case map[string]interface{}:
			fields = v
		}
	}

	if message != "" {
		if fields == nil {
			fields = Fields{"message": message}
		} else {
			fields["message"] = message
		}
	}
	if cause != nil && cause != kind {
		tmpErr := With(cause, kind)
		return WrapWithFieldsAndDepth(tmpErr, fields, 2)
	}

	return WrapWithFieldsAndDepth(kind, fields, 2)
}

//
// func (ke *khanError) MarshalLogObject(enc zapcore.ObjectEncoder) error {
//	enc.AddString("kind", string(ke.kind))
//	enc.AddString("message", ke.Error())
//	enc.AddString("stacktrace", fmt.Sprintf("%+v", ke.StackTrace()))
//	err := enc.AddReflected("fields", ke.fields)
//	if err != nil {
//		return errors.Wrapf(err, "Unable to AddReflected fields to log: %+v", ke.fields)
//	}
//	err = enc.AddReflected("cause", ke.cause)
//	if err != nil {
//		return errors.Wrapf(err, "Unable to AddReflected cause to log %+v", ke.cause)
//	}
//
//	return nil
//}
