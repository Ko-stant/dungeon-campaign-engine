package components

import "context"

type signedInKey struct{}

// WithSignedIn returns a context whose pages show who is signed in.
func WithSignedIn(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, signedInKey{}, name)
}

// SignedIn returns the signed-in user's name, if sign-in is on.
func SignedIn(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(signedInKey{}).(string)
	return name, ok
}
