/*
Copyright 2024 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package session

import (
	"context"
	"errors"
)

// ErrNoSession is the error that no session found in context.
var ErrNoSession = errors.New("no session found in context")

// sessionKey is the key for the session in the context.
type sessionKey struct{}

// WithContext returns a new context with the session.
func WithContext(ctx context.Context, sess Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, sess)
}

// FromContext returns the session from the context.
// If no session is found in the context, it returns ErrNoSession.
func FromContext(ctx context.Context) (Session, error) {
	sess, ok := ctx.Value(sessionKey{}).(Session)
	if !ok {
		return nil, ErrNoSession
	}
	return sess, nil
}
