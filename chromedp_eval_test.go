package dashboard_test

import (
	"context"

	"github.com/chromedp/chromedp"
)

// chromedp v0.21 turned actions into values (Action[T]) and replaced the
// pointer-out-parameter form of Evaluate with Evaluate[T]. These two wrappers
// keep the "evaluate into this variable" shape the browser tests are written
// in, so the migration stayed mechanical instead of restructuring every step
// list into a sequence of single-action Run calls.

// evalInto evaluates expr and stores the decoded result in out.
func evalInto[T any](expr string, out *T) chromedp.Action[chromedp.Void] {
	return chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
		value, err := chromedp.Evaluate[T](expr)(ctx, t)
		if err != nil {
			return err
		}

		*out = value

		return nil
	})
}

// evalDiscard evaluates expr for its side effect and drops the result.
func evalDiscard(expr string) chromedp.Action[chromedp.Void] {
	return chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
		_, err := chromedp.Evaluate[chromedp.Void](expr)(ctx, t)

		return err
	})
}
