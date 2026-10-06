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

type adminKey struct{}

// WithAdmin marks a context's pages as an admin's, with how many people
// wait to be let in.
func WithAdmin(ctx context.Context, waiting int) context.Context {
	return context.WithValue(ctx, adminKey{}, waiting)
}

// Admin reports whether the signed-in user is an admin, and how many wait.
func Admin(ctx context.Context) (waiting int, ok bool) {
	waiting, ok = ctx.Value(adminKey{}).(int)
	return waiting, ok
}
